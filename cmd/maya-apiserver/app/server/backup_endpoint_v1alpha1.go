package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	backupapi "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	"github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	backup "github.com/openebs/maya/pkg/backup/v1alpha1"
	openebs "github.com/openebs/maya/pkg/client/generated/clientset/internalclientset"
	snapshot "github.com/openebs/maya/pkg/snapshot/v1alpha1"
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
		return nil, CodedError(400, fmt.Sprintf("Failed to parse backup request '%v'", err))

	}

	if err = createSnapshot(bkp); err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to create snapshot '%v'", err))
	}

	if err = deployBackup(bkp); err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to deploy backup:%v", err))
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

	glog.Infof("bkp %v %v %v", bkp, bkp.GetBackupAPIObject(), bkp.KubeClient)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to parse backup request object: %v", err))
	}

	nb, err := fetchUpdatedBackup(bkp)
	if err != nil {
		//TODO wrap error
		return nil, CodedError(400, fmt.Sprintf("Failed to fetch backup update:%v", err))
	}

	out, err := json.Marshal(nb.GetBackupAPIObject())
	if err == nil {
		_, err = bOps.resp.Write(out)
		if err != nil {
			return nil, CodedError(400, fmt.Sprintf("Failed to send response data"))
		}
		return nil, nil
	}

	return nil, CodedError(400, fmt.Sprintf("Failed to encode response data"))
}

// fetchUpdatedBackup validates backup execution path
// and returns updated backup object
func fetchUpdatedBackup(b *backup.CStorBackup) (*backup.CStorBackup, error) {
	nb, err := b.GetCR(b.GetObjName())
	if err != nil {
		//TODO wrap error
		return nil, err
	}
	glog.Infof("got nb %v %v", nb, nb.GetBackupAPIObject())
	//TODO check error and return in-progress if any network related error
	if !nb.IsFailedStatus() && !nb.IsDoneStatus() {
		// check if node is running or not
		bkpNodeDown, _ := checkIfBKPPoolNodeDown(b)
		// check if cstor-pool-mgmt container is running or not
		bkpPodDown, _ := checkIfBKPPoolPodDown(b)
		if bkpNodeDown || bkpPodDown {
			// Backup is stalled, let's find last-backup status
			laststat, _ := findLastBackupStat(b)
			// Update Backup status according to last-backup
			nb.SetStatus(laststat)
			b.UpdateCR(nb)

			// Get updated backup object
			lb, err := b.GetCR(b.GetObjName())
			if err != nil {
				//TODO wrap error
				return nil, err
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
		//TODO wrap error
		return nodeDown, err
	}

	pod, err := findPodFromCStorID(k8sclient, bkp.GetLabel(PoolUUID))
	if err != nil {
		//TODO wrap error
		return nodeDown, err
	}

	if pod.Spec.NodeName == "" {
		//TOTO wrap error
		return nodeDown, err
	}

	node, err := k8sclient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		//TODO wrap error
		return nodeDown, err
	}
	for _, nodestat := range node.Status.Conditions {
		if nodestat.Type == corev1.NodeReady && nodestat.Status != corev1.ConditionTrue {
			//TODO wrap error
			return nodeDown, err
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
		//TODO wrap error
		return podDown, err
	}

	pod, err := findPodFromCStorID(k8sclient, bkp.GetLabel(PoolUUID))
	if err != nil {
		//TODO wrap error
		return podDown, err
	}

	for _, containerstatus := range pod.Status.ContainerStatuses {
		if containerstatus.Name == CStorPoolMgmtContainer {
			return !containerstatus.Ready, err
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
		glog.Errorf("Failed to fetch pod list :%v", err)
		return corev1.Pod{}, errors.New("Failed to fetch pod list")
	}

	for _, pod := range podlist.Items {
		for _, env := range pod.Spec.Containers[0].Env {
			if env.Name == "OPENEBS_IO_CSTOR_ID" && env.Value == cstorID {
				return pod, nil
			}
		}
	}
	return corev1.Pod{}, errors.New("No Pod exists")
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
		//TODO wrap error
		return err
	}

	_, err = snapOps.Create()
	if err != nil {
		//TODO wrap error
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
		return v1alpha1.CStorVolumeReplica{}, err
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
		//TODO wrap error
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
		//TODO wrap error
		return err
	}

	// Initialize backup status as pending
	b.SetStatus(backupapi.BKPCStorStatusPending)
	_, err = b.CreateCR(b)
	return err
}

// getK8sClient return kubernets client for CR operation
func getK8sClient() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		//TODO wrap error
		return nil, err
	}

	// creates the in-cluster kubernetes clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// getInClusterOECS is used to initialize and return a new http client capable
// of invoking OpenEBS CRD APIs within the cluster
func getOpenEBSClient() (*openebs.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		//TODO wrap error
		return nil, err
	}

	// creates the in-cluster openebs clientset
	clientset, err := openebs.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// findLastBackupStatus returns status of given backup
// by checking last completed backup
func findLastBackupStat(bkp *backup.CStorBackup) (backupapi.CStorBackupStatus, error) {
	lastsnap, err := bkp.GetLastTransferredSnapName()
	if err != nil {
		// Unable to fetch the last backup, so we will return fail state
		//TODO wrap error
		return backupapi.BKPCStorStatusFailed, err
		//glog.Infof("LastBackup resource created for backup:%s volume:%s", bk.Spec.BackupName, bk.Spec.VolumeName)
	}

	//TODO
	// let's check if snapname matches with current snapshot name
	if lastsnap == bkp.GetSnapName() {
		return backupapi.BKPCStorStatusDone, nil
	}
	// lastbackup snap/prevsnap doesn't match with bkp snapname
	return backupapi.BKPCStorStatusFailed, nil
}
