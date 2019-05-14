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

package backup

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openebs/maya/cmd/cstor-pool-mgmt/controller/common"
	"github.com/openebs/maya/cmd/cstor-pool-mgmt/volumereplica"
	backupapi "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	backup "github.com/openebs/maya/pkg/backup/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

// processBackup fetch the CStorBackup resource using key
// and execute backup process
func (c *Controller) processBackup(key string, operation common.QueueOperation) error {
	glog.Infof("processing backup for key{%s} with op{%s}", key, operation)

	bkp, err := c.getResource(key)
	if err != nil {
		return err
	}

	err = c.handleOps(operation, bkp)
	if err != nil {
		return errors.Errorf("Failed process backup operation: %s", err.Error())
	}

	glog.Infof("handled backup operation{%s} for key{%s} successfully", operation, key)
	return nil
}

// handleOps will execute a function according to a given operation
func (c *Controller) handleOps(operation common.QueueOperation, bkp *backup.CStorBackup) error {
	switch operation {
	case common.QOpAdd:
		return c.handleOpsAdd(bkp)
	case common.QOpDestroy:
		/* TODO: Handle backup destroy event
		 */
		return nil
	case common.QOpSync:
		return c.handleOpsSync(bkp)
	}
	return nil
}

// handleOpsAdd will change the state of backup to Init state.
func (c *Controller) handleOpsAdd(bkp *backup.CStorBackup) error {
	var status = backupapi.BKPCStorStatusInit

	if !bkp.IsPendingStatus() {
		status = backupapi.BKPCStorStatusInvalid
	}

	err := bkp.UpdateCRStatus(bkp.GetObjName(), status)
	if err != nil {
		return err
	}

	return nil
}

// handleOpsSync will perform the backup if a given backup is in init state
func (c *Controller) handleOpsSync(bkp *backup.CStorBackup) error {
	var ret error
	// If the backup is in init state then only we will complete the backup
	if bkp.IsInitStatus() {
		err := bkp.UpdateCRStatus(bkp.GetObjName(), backupapi.BKPCStorStatusInProgress)
		if err != nil {
			return err
		}

		err = volumereplica.CreateVolumeBackup(bkp.GetBackupAPIObject())
		if err != nil {
			errors.Wrapf(ret, "Failed to execute backup{%s} : %s", bkp.GetObjName(), err.Error())
			c.recorder.Event(
				bkp.GetBackupAPIObject(),
				corev1.EventTypeNormal,
				string(common.SuccessCreated),
				string(common.MessageResourceCreated))
			//TODO change log event
			// change it to multicall
			err = bkp.UpdateCRStatus(bkp.GetObjName(), backupapi.BKPCStorStatusFailed)
			if err != nil {
				errors.Wrapf(ret,
					"Failed to set status{%s} for backup{%s}",
					backupapi.BKPCStorStatusFailed,
					bkp.GetObjName())
			}
			return ret
		}

		c.recorder.Event(
			bkp.GetBackupAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessCreated),
			string(common.MessageResourceCreated))
		//TODO glog.Infof("backup creation successful: %v, %v", bkp.ObjectMeta.Name, string(bkp.GetUID()))
		err = bkp.UpdateCompletedBackup()
		if err != nil {
			errors.Wrapf(ret,
				"Failed to update completed-backup for backup{%s} : %s",
				bkp.GetObjName(), err.Error())
			err = bkp.UpdateCRStatus(bkp.GetObjName(), backupapi.BKPCStorStatusFailed)
			if err != nil {
				errors.Wrapf(ret,
					"Failed to set status{%s} for backup{%s}",
					backupapi.BKPCStorStatusFailed,
					bkp.GetObjName())
			}
			return ret
		}

		return bkp.UpdateCRStatus(bkp.GetObjName(), backupapi.BKPCStorStatusDone)
	}
	return nil
}

// getResource returns a CStorBackup object corresponding to the resource key
func (c *Controller) getResource(key string) (*backup.CStorBackup, error) {
	// Convert the key(namespace/name) string into a distinct name
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key{%s}", key))
		return nil, nil
	}

	return backup.NewCStorBackupBuilder().
		WithNameSpace(ns).
		WithClientSet(nil).
		BuildFromAPIObjectName(name)
}
