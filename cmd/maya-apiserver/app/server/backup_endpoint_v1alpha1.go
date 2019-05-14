/*
Copyright 2019 The OpenEBS Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	backupapi "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	"github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	backup "github.com/openebs/maya/pkg/backup/v1alpha1"
	openebs "github.com/openebs/maya/pkg/client/generated/clientset/internalclientset"
	snapshot "github.com/openebs/maya/pkg/snapshot/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type backupAPIOps struct {
	req  *http.Request
	resp http.ResponseWriter
}

const (
	// OpenEBSNs is OpenEBS name-space
	OpenEBSNs = "OPENEBS_NAMESPACE"

	// CStorPoolAppLabel is label key-value for cstor pool application
	CStorPoolAppLabel = "app=cstor-pool"

	// PoolUUID is key for pool UUID
	PoolUUID = "cstorpool.openebs.io/uid"

	// OpenEBSPV is key for OpenEBS PV
	OpenEBSPV = "openebs.io/persistent-volume"

	// OpenEBSBackup is key for OpenEBS backup
	OpenEBSBackup = "openebs.io/backup"

	// CStorPoolMgmtContainer is name of cstor-pool management container
	CStorPoolMgmtContainer = "cstor-pool-mgmt"
)

// backupV1alpha1SpecificRequest deals with backup API requests
func (s *HTTPServer) backupV1alpha1SpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	backupOp := &backupAPIOps{
		req:  req,
		resp: resp,
	}

	switch req.Method {
	case "POST":
		return backupOp.create()
	case "GET":
		return backupOp.get()
	}
	return nil, CodedError(405, ErrInvalidMethod)
}

// Create is http handler which handles backup create request
func (bOps *backupAPIOps) create() (interface{}, error) {
	cbkp := &backupapi.CStorBackup{}
	err := decodeBody(bOps.req, cbkp)
	if err != nil {
		return nil, err
	}

	bkp, err := backup.NewCStorBackupBuilder().
		WithCheck(backup.IsBackupNameSet()).
		WithCheck(backup.IsSnapNameSet()).
		WithCheck(backup.IsBackupDestSet()).
		WithCheck(backup.IsPrevSnapNameSet()).
		WithCheck(backup.IsNamespaceSet()).
		WithCheck(backup.IsVolumeNameSet()).
		WithClientSet(nil).
		BuildFromAPIObject(cbkp)

	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to parse backup request : %s", err.Error()))

	}

	if err = createSnapshot(bkp); err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to create snapshot : %s", err.Error()))
	}

	if err = deployBackup(bkp); err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to deploy backup : %s", err.Error()))
	}
	glog.Infof("Backup:{%s} scheduled on CSP:{%s} for volume:{%s}",
		bkp.GetSnapName(),
		bkp.GetLabel(PoolUUID),
		bkp.GetVolumeName())
	return nil, nil
}

// get is http handler which handles backup get request
func (bOps *backupAPIOps) get() (interface{}, error) {
	cbkp := &backupapi.CStorBackup{}

	err := decodeBody(bOps.req, cbkp)
	if err != nil {
		return nil, err
	}

	bkp, err := backup.NewCStorBackupBuilder().
		WithCheck(backup.IsBackupNameSet()).
		WithCheck(backup.IsNamespaceSet()).
		WithCheck(backup.IsVolumeNameSet()).
		WithNameSpace(cbkp.Namespace).
		WithClientSet(nil).
		BuildFromAPIObjectName(backup.DeriveObjName(cbkp))

	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to parse backup request object: %v", err))
	}

	nb, err := fetchUpdatedBackup(bkp)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to fetch backup update : %v", err))
	}

	out, err := json.Marshal(nb.GetBackupAPIObject())
	if err == nil {
		_, err = bOps.resp.Write(out)
		if err != nil {
			return nil, CodedError(400, fmt.Sprintf("Failed to write response data : %v", err))
		}
		return nil, nil
	}

	return nil, CodedError(400, fmt.Sprintf("Failed to encode response data : %s", err.Error()))
}

// fetchUpdatedBackup validates backup execution path
// and returns updated backup object
func fetchUpdatedBackup(b *backup.CStorBackup) (*backup.CStorBackup, error) {
	nb, err := b.GetCR(b.GetObjName())
	if err != nil {
		return nil, errors.Errorf("Failed to fetch backup{%s} : %s", b.GetObjName(), err.Error())
	}

	//TODO check error and return in-progress if any network related error
	if !nb.IsFailedStatus() && !nb.IsDoneStatus() {
		// check if node is running or not
		bkpNodeDown, nodeError := checkIfBKPPoolNodeDown(b)
		// check if cstor-pool-mgmt container is running or not
		bkpPodDown, podError := checkIfBKPPoolPodDown(b)

		if nodeError != nil || podError != nil {
			return nil, errors.Errorf("Error occured while checking restore status nodeError:%v podError:%v",
				nodeError, podError)
		}

		if bkpNodeDown || bkpPodDown {
			// Backup is stalled, let's find last-backup status
			laststat, _ := findLastBackupStat(b)
			// Update Backup status according to last-backup
			nb.SetStatus(laststat)
			b.UpdateCR(nb)

			// Get updated backup object
			lb, err := b.GetCR(b.GetObjName())
			if err != nil {
				return nil, errors.Errorf("Failed to fetch backup{%s} : %s", b.GetObjName(), err.Error())
			}
			return lb, nil
		}
	}
	return nb, nil
}

// checkIfBKPPoolNodeDown will check if pool node on which
// given backup is being executed is running or not
func checkIfBKPPoolNodeDown(bkp *backup.CStorBackup) (bool, error) {
	var nodeDown = true

	k8sclient, err := getK8sClient()
	if err != nil {
		return nodeDown, errors.Errorf("Failed to fetch clientset : %s", err.Error())
	}

	pod, err := findPodFromCStorID(k8sclient, bkp.GetLabel(PoolUUID))
	if err != nil {
		//TODO wrap error
		return nodeDown, errors.Errorf("Failed to fetch Pod info : %s", err.Error())
	}

	if pod.Spec.NodeName == "" {
		//TOTO wrap error
		return nodeDown, errors.Errorf("NodeName is missing for pod")
	}

	node, err := k8sclient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		//TODO wrap error
		return nodeDown, errors.Errorf("Failed to fetch node info for pod{%s}: %s", pod.Name, err.Error())
	}

	for _, nodestat := range node.Status.Conditions {
		if nodestat.Type == corev1.NodeReady && nodestat.Status != corev1.ConditionTrue {
			//TODO wrap error
			return nodeDown, nil
		}
	}
	return !nodeDown, nil
}

// checkIfBKPPoolPodDown will check if pool pod on which
// given backup is being executed is running or not
func checkIfBKPPoolPodDown(bkp *backup.CStorBackup) (bool, error) {
	var podDown = true

	k8sclient, err := getK8sClient()
	if err != nil {
		return podDown, errors.Errorf("Failed to fetch clientset : %s", err.Error())
	}

	pod, err := findPodFromCStorID(k8sclient, bkp.GetLabel(PoolUUID))
	if err != nil {
		return podDown, errors.Errorf("Failed to fetch Pod info : %s", err.Error())
	}

	for _, containerstatus := range pod.Status.ContainerStatuses {
		if containerstatus.Name == CStorPoolMgmtContainer {
			return !containerstatus.Ready, nil
		}
	}

	return podDown, nil
}

// findPodFromCStorID will find the Pod having given cstorID
func findPodFromCStorID(k8sclient *kubernetes.Clientset, cstorID string) (corev1.Pod, error) {
	podlistops := metav1.ListOptions{
		LabelSelector: CStorPoolAppLabel,
	}

	openebsNs := os.Getenv(OpenEBSNs)
	if openebsNs == "" {
		return corev1.Pod{}, errors.New("Failed to fetch operator namespace")
	}

	podlist, err := k8sclient.CoreV1().Pods(openebsNs).List(podlistops)
	if err != nil {
		return corev1.Pod{}, errors.New("Failed to fetch pod list")
	}

	for _, pod := range podlist.Items {
		for _, env := range pod.Spec.Containers[0].Env {
			if env.Name == "OPENEBS_IO_CSTOR_ID" && env.Value == cstorID {
				return pod, nil
			}
		}
	}
	return corev1.Pod{}, errors.Errorf("No Pod exists for CstorID{%s}", cstorID)
}

// createSnapshot will create a snapshot for given backup
func createSnapshot(b *backup.CStorBackup) error {
	snapOps, err := snapshot.Snapshot(&v1alpha1.SnapshotOptions{
		VolumeName: b.GetVolumeName(),
		Namespace:  b.GetNamespace(),
		CasType:    string(v1alpha1.CstorVolume),
		Name:       b.GetSnapName(),
	})
	if err != nil {
		return err
	}

	_, err = snapOps.Create()
	if err != nil {
		return err
	}
	return nil
}

// findHealthyCVR will find a healthy CVR for a given volume
func findHealthyCVR(volume string) (v1alpha1.CStorVolumeReplica, error) {
	client, err := getOpenEBSClient()
	if err != nil {
		//TODO wrap error
		return v1alpha1.CStorVolumeReplica{}, err
	}

	listOptions := v1.ListOptions{
		LabelSelector: OpenEBSPV + "=" + volume,
	}

	cvrList, err := client.OpenebsV1alpha1().CStorVolumeReplicas("").List(listOptions)
	if err != nil {
		//TODO wrap error
		return v1alpha1.CStorVolumeReplica{}, errors.Errorf("Failed fetch CVR list : %s", err.Error())
	}

	// Select a healthy cvr for backup
	for _, cvr := range cvrList.Items {
		if cvr.Status.Phase == v1alpha1.CVRStatusOnline {
			return cvr, nil
		}
	}

	//TODO wrap error
	return v1alpha1.CStorVolumeReplica{}, errors.New("unable to find healthy CVR")
}

// deployBackup will set the execution details and
// create the CR for given backup
func deployBackup(b *backup.CStorBackup) error {
	// find healthy CVR
	cvr, err := findHealthyCVR(b.GetVolumeName())
	if err != nil {
		return err
	}

	b.SetLabel(
		map[string]string{
			PoolUUID:      cvr.ObjectMeta.Labels[PoolUUID],
			OpenEBSPV:     cvr.ObjectMeta.Labels[OpenEBSPV],
			OpenEBSBackup: b.GetBackupName(),
		},
	)

	// update last backup snapshot name
	err = b.SetPrevSnapNameFromLastBackup()
	if err != nil {
		return errors.Errorf("Failed to set prev-snapname for backup{%s} : %s", b.GetObjName(), err.Error())
	}

	// Initialize backup status as pending
	b.SetStatus(backupapi.BKPCStorStatusPending)
	_, err = b.CreateCR(b)
	return errors.Errorf("Failed to create backup{%s} CR : %s", b.GetObjName(), err.Error())
}

// getK8sClient return kubernets client for CR operation
func getK8sClient() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Errorf("Failed to find cluster config")
	}

	// creates the in-cluster kubernetes clientset
	return kubernetes.NewForConfig(cfg)
}

// getInClusterOECS is used to initialize and return a new http client capable
// of invoking OpenEBS CRD APIs within the cluster
func getOpenEBSClient() (*openebs.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Errorf("Failed to find cluster config")
	}

	// creates the in-cluster openebs clientset
	return openebs.NewForConfig(cfg)
}

// findLastBackupStatus returns status of given backup
// by checking last completed backup
func findLastBackupStat(bkp *backup.CStorBackup) (backupapi.CStorBackupStatus, error) {
	lastsnap, err := bkp.GetLastTransferredSnapName()
	if err != nil {
		// Unable to fetch the last backup, so we will return fail state
		return backupapi.BKPCStorStatusFailed, errors.Errorf("Failed to fetch last snap for backup{%s} : %s", bkp.GetObjName(), err.Error())
	}

	//TODO
	// let's check if snapname matches with current snapshot name
	if lastsnap == bkp.GetSnapName() {
		return backupapi.BKPCStorStatusDone, nil
	}
	// lastbackup snap/prevsnap doesn't match with bkp snapname
	return backupapi.BKPCStorStatusFailed, nil
}
