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

package restore

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openebs/maya/cmd/cstor-pool-mgmt/controller/common"
	"github.com/openebs/maya/cmd/cstor-pool-mgmt/volumereplica"
	restoreapi "github.com/openebs/maya/pkg/apis/openebs.io/restore/v1alpha1"
	restore "github.com/openebs/maya/pkg/restore/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

// processRestore fetch the CStorRestore resource using key
// and execute restore process
func (c *Controller) processRestore(key string, operation common.QueueOperation) error {
	glog.Infof("Processing restore for key{%s} with op{%s}", key, operation)
	rst, err := c.getResource(key)
	if err != nil {
		return err
	}

	err = c.handleOps(operation, rst)
	if err != nil {
		return errors.Wrapf(err, "Failed to process restore for key{%s} : %s", key, err.Error())
	}

	glog.Infof("successfully process restore operation{%s} for key{%s}", operation, key)
	return nil
}

// handleOps will execute a function according to a given operation
func (c *Controller) handleOps(operation common.QueueOperation, rst *restore.CStorRestore) error {
	switch operation {
	case common.QOpAdd:
		return c.handleOpsAdd(rst)
	case common.QOpDestroy:
		/*TODO
		status, err := c.rstDestroyEventHandler(rstGot)
		return status, err
		glog.Infof("Processing restore delete event %v, %v", rstGot.ObjectMeta.Name, string(rstGot.GetUID()))
		*/
		return nil
	case common.QOpSync:
		return c.handleOpsSync(rst)
	}
	return nil
}

// handleOpsAdd will change the state of restore to Init state.
func (c *Controller) handleOpsAdd(rst *restore.CStorRestore) error {
	var status = restoreapi.RSTCStorStatusInit

	if !rst.IsPendingStatus() {
		status = restoreapi.RSTCStorStatusInvalid
	}

	err := rst.UpdateCRStatus(rst.GetObjName(), status)
	if err != nil {
		return errors.Wrapf(err, "Failed to update restore{%s} status to {%s}", rst.GetObjName(), status)
	}

	return nil
}

// handleOpsSync will perform the restore if a given restore is in init state
func (c *Controller) handleOpsSync(rst *restore.CStorRestore) error {
	var ret error
	// If the restore is in init state then only we will complete the restore
	if rst.IsInitStatus() {
		err := rst.UpdateCRStatus(rst.GetObjName(), restoreapi.RSTCStorStatusInProgress)
		if err != nil {
			return errors.Errorf("Failed to update restore{%s} status to {%s} : %s",
				rst.GetObjName(), restoreapi.RSTCStorStatusInProgress, err.Error())
		}

		err = volumereplica.CreateVolumeRestore(rst.GetRestoreAPIObject())
		if err != nil {
			errors.Wrapf(ret, "Failed to execute restore{%s} : %s", rst.GetObjName(), err.Error())
			c.recorder.Event(
				rst.GetRestoreAPIObject(),
				corev1.EventTypeNormal,
				string(common.SuccessCreated),
				string(common.MessageResourceCreated))

			err := rst.UpdateCRStatus(rst.GetObjName(), restoreapi.RSTCStorStatusFailed)
			if err != nil {
				errors.Wrapf(ret,
					"Failed to update restore{%s} status to {%s} : %s",
					rst.GetObjName(),
					restoreapi.RSTCStorStatusFailed,
					err.Error())
			}
			return ret
		}

		c.recorder.Event(
			rst.GetRestoreAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessCreated),
			string(common.MessageResourceCreated))
		err = rst.UpdateCRStatus(rst.GetObjName(), restoreapi.RSTCStorStatusDone)
		if err != nil {
			return errors.Errorf("Failed to update restore{%s} status to {%s} : %s",
				rst.GetObjName(),
				restoreapi.RSTCStorStatusDone,
				err.Error())
		}
		return nil
	}
	return nil
}

// getResource returns a restore object corresponding to the resource key
func (c *Controller) getResource(key string) (*restore.CStorRestore, error) {
	// Convert the key(namespace/name) string into a distinct name
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil, nil
	}

	return restore.NewCStorRestoreBuilder().
		WithNameSpace(ns).
		WithClientSet(nil).
		BuildFromAPIObjectName(name)
}
