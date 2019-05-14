/*
Copyright 2019 The OpenEBS Authors

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

	"github.com/golang/glog"
	restoreapi "github.com/openebs/maya/pkg/apis/openebs.io/restore/v1alpha1"
	restore "github.com/openebs/maya/pkg/restore/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type restoreAPIOps struct {
	req  *http.Request
	resp http.ResponseWriter
}

// restoreV1alpha1SpecificRequest deals with restore API requests
func (s *HTTPServer) restoreV1alpha1SpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	restoreOp := &restoreAPIOps{
		req:  req,
		resp: resp,
	}

	switch req.Method {
	case "POST":
		return restoreOp.create()
	case "GET":
		return restoreOp.get()
	}
	return nil, CodedError(405, ErrInvalidMethod)
}

// Create is http handler which handles restore-create request
func (rOps *restoreAPIOps) create() (interface{}, error) {
	var err error
	crestore := &restoreapi.CStorRestore{}
	err = decodeBody(rOps.req, crestore)
	if err != nil {
		return nil, err
	}

	restore, err := restore.NewCStorRestoreBuilder().
		WithCheck(restore.IsRestoreNameSet()).
		WithCheck(restore.IsVolumeNameSet()).
		WithCheck(restore.IsNamespaceSet()).
		WithCheck(restore.IsRestoreSrcSet()).
		WithClientSet(nil).
		BuildFromAPIObject(crestore)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to parse restore request : %s", err.Error()))
	}

	return createRestoreResource(restore)
}

// createRestoreResource create restore CR for volume's CVR
func createRestoreResource(rst *restore.CStorRestore) (interface{}, error) {
	//Get List of cvr's related to this pvc
	listOptions := v1.ListOptions{
		LabelSelector: "openebs.io/persistent-volume=" + rst.GetVolumeName(),
	}

	clientset, err := getOpenEBSClient()
	if err != nil {
		return nil, CodedError(500, err.Error())
	}

	cvrList, err := clientset.OpenebsV1alpha1().CStorVolumeReplicas("").List(listOptions)
	if err != nil {
		return nil, CodedError(500, fmt.Sprintf("Failed to fetch CVR list : %s", err.Error()))
	}

	for _, cvr := range cvrList.Items {
		rst.RegenerateObjName()
		oldrst, err := rst.GetCR(rst.GetObjName())
		if err != nil {
			rst.SetStatus(restoreapi.RSTCStorStatusPending)
			rst.SetLabel(map[string]string{
				"cstorpool.openebs.io/uid":     cvr.ObjectMeta.Labels["cstorpool.openebs.io/uid"],
				"openebs.io/persistent-volume": cvr.ObjectMeta.Labels["openebs.io/persistent-volume"],
				"openebs.io/restore":           rst.GetRestoreName(),
			})

			_, err = rst.CreateCR(rst)
			if err != nil {
				glog.Errorf("Failed to create restore CR(volume:%s CSP:%s) : %s",
					rst.GetVolumeName(), cvr.ObjectMeta.Labels["cstorpool.openebs.io/uid"],
					err.Error())
				return nil, CodedError(500, err.Error())
			}
			glog.Infof("Restore{%s} created for volume{%s} poolUUID{%s}", rst.GetObjName(),
				rst.GetVolumeName(),
				rst.GetLabel("cstorpool.openebs.io/uid"))
		} else {
			oldrst.SetStatus(restoreapi.RSTCStorStatusPending)
			oldrst.CopySpec(rst)
			oldrst.SetLabel(map[string]string{
				"cstorpool.openebs.io/uid":     cvr.ObjectMeta.Labels["cstorpool.openebs.io/uid"],
				"openebs.io/persistent-volume": cvr.ObjectMeta.Labels["openebs.io/persistent-volume"],
				"openebs.io/restore":           rst.GetRestoreName(),
			})
			_, err = rst.UpdateCR(oldrst)
			if err != nil {
				glog.Errorf("Failed to re-initialize old existing restore CR(volume:%s CSP:%s) : %s",
					rst.GetVolumeName(), cvr.ObjectMeta.Labels["cstorpool.openebs.io/uid"],
					err.Error())
				return nil, CodedError(500, err.Error())
			}
			glog.Infof("Re-initializing old restore{%s} for volume{%s} poolUUID{%s}", rst.GetObjName(),
				rst.GetVolumeName(),
				rst.GetLabel("cstorpool.openebs.io/uid"))
		}
	}

	return "", nil
}

// get is http handler which handles backup get request
func (rOps *restoreAPIOps) get() (interface{}, error) {
	var err error
	var rstatus restoreapi.CStorRestoreStatus
	var resp []byte

	crst := &restoreapi.CStorRestore{}

	err = decodeBody(rOps.req, crst)
	if err != nil {
		return nil, err
	}

	rst, err := restore.NewCStorRestoreBuilder().
		WithCheck(restore.IsRestoreNameSet()).
		WithCheck(restore.IsNamespaceSet()).
		WithCheck(restore.IsVolumeNameSet()).
		WithClientSet(nil).
		BuildFromAPIObject(crst)

	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to parse restore request : %s", err.Error()))
	}

	rstatus, err = getRestoreStatus(rst)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to fetch status : %s", err.Error()))
	}

	resp, err = json.Marshal(rstatus)
	if err == nil {
		_, err = rOps.resp.Write(resp)
		if err != nil {
			return nil, CodedError(400, fmt.Sprintf("Failed to write response data : %s", err.Error()))
		}
		return nil, nil
	}

	return nil, CodedError(400, fmt.Sprintf("Failed to encode response data : %s", err.Error()))
}

func getRestoreStatus(rst *restore.CStorRestore) (restoreapi.CStorRestoreStatus, error) {
	rstStatus := restoreapi.RSTCStorStatusEmpty
	/*
		listOptions := v1.ListOptions{
			LabelSelector: "openebs.io/restore=" + rst.GetRestoreName() + ",openebs.io/persistent-volume=" + rst.GetVolumeName(),
		}
	*/
	//TODO add option for list builder
	rlist, err := restore.NewCStorRestoreListBuilder().
		WithNamespace(rst.GetNamespace()).
		WithClientSet(nil).
		Build()
	if err != nil {
		return restoreapi.RSTCStorStatusEmpty, err
	}

	for _, nr := range rlist.Item {
		rstStatus = getCVRRestoreStatus(nr)

		switch rstStatus {
		case restoreapi.RSTCStorStatusInProgress:
			rstStatus = restoreapi.RSTCStorStatusInProgress
		case restoreapi.RSTCStorStatusFailed:
			if nr.GetStatus() != rstStatus {
				// Restore for given CVR may failed due to node failure or pool failure
				// Let's update status for given CVR's restore to failed
				// Update Backup status according to last-backup
				nr.SetStatus(rstStatus)
				nr.UpdateCR(nr)
			}
			rstStatus = restoreapi.RSTCStorStatusFailed
		case restoreapi.RSTCStorStatusDone:
			if rstStatus != restoreapi.RSTCStorStatusFailed {
				rstStatus = restoreapi.RSTCStorStatusDone
			}
		}

		glog.Infof("Restore{%v} status is {%s}", nr.GetObjName(), nr.GetStatus())

		if rstStatus == restoreapi.RSTCStorStatusInProgress {
			break
		}
	}
	return rstStatus, nil
}

func getCVRRestoreStatus(rst *restore.CStorRestore) restoreapi.CStorRestoreStatus {
	if !rst.IsFailedStatus() && !rst.IsDoneStatus() {
		// check if node is running or not
		bkpNodeDown, nodeError := checkIfRSTPoolNodeDown(rst)
		// check if cstor-pool-mgmt container is running or not
		bkpPodDown, podError := checkIfRSTPoolPodDown(rst)

		if nodeError != nil || podError != nil {
			glog.Errorf("Error occured while checking restore status node:%v pod:%v", nodeError, podError)
			return restoreapi.RSTCStorStatusInProgress
		}
		if bkpNodeDown || bkpPodDown {
			// Backup is stalled, assume status as failed
			return restoreapi.RSTCStorStatusFailed
		}
	}
	return rst.GetStatus()
}

// checkIfRSTPoolNodeDown will check if pool node on which
// given restore is being executed is running or not
func checkIfRSTPoolNodeDown(rst *restore.CStorRestore) (bool, error) {
	var nodeDown = true

	k8sclient, err := getK8sClient()
	if err != nil {
		return nodeDown, error.Errorf("Failed to fetch clientset : %s", err.Error())
	}

	pod, err := findPodFromCStorID(k8sclient, rst.GetLabel(PoolUUID))
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

// checkIfRSTPoolPodDown will check if pool pod on which
// given restore is being executed is running or not
func checkIfRSTPoolPodDown(rst *restore.CStorRestore) (bool, error) {
	var podDown = true

	k8sclient, err := getK8sClient()
	if err != nil {
		//TODO wrap error
		return podDown, errors.Errorf("Failed to fetch clientset : %s", err.Error())
	}

	pod, err := findPodFromCStorID(k8sclient, rst.GetLabel(PoolUUID))
	if err != nil {
		//TODO wrap error
		return podDown, errors.Errorf("Failed to fetch Pod info : %s", err.Error())
	}

	for _, containerstatus := range pod.Status.ContainerStatuses {
		if containerstatus.Name == CStorPoolMgmtContainer {
			return !containerstatus.Ready, nil
		}
	}

	return podDown, nil
}
