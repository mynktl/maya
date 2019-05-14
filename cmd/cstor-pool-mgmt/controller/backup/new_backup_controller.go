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
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/openebs/maya/cmd/cstor-pool-mgmt/controller/common"
	backupapi "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	backup "github.com/openebs/maya/pkg/backup/v1alpha1"
	backupclientset "github.com/openebs/maya/pkg/client/generated/openebs.io/backup/v1alpha1/clientset/internalclientset"
	backupScheme "github.com/openebs/maya/pkg/client/generated/openebs.io/backup/v1alpha1/clientset/internalclientset/scheme"
	informers "github.com/openebs/maya/pkg/client/generated/openebs.io/backup/v1alpha1/informer/externalversions"
)

const controllerName = "CStorBackup"

// Controller is the controller implementation for BackupCStor resources.
type Controller struct {
	// kubeclientset is a standard kubernetes clientset.
	kubeclientset kubernetes.Interface

	// clientset is a backup custom resource package generated for custom API group.
	clientset backupclientset.Interface

	// BackupSynced is used for caches sync to get populated
	BackupSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new cStor Backup controller instance
func NewController(
	kubeclientset kubernetes.Interface,
	clientset backupclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	bkpInformerFactory informers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the BackupCStor resources.
	BackupInformer := bkpInformerFactory.Openebs().V1alpha1().CStorBackups()

	err := backupScheme.AddToScheme(scheme.Scheme)
	if err != nil {
		glog.Fatalf("Error adding backup scheme : %s", err.Error())
		return nil
	}

	// Create event broadcaster
	glog.V(4).Info("Creating backup event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)

	// StartEventWatcher starts sending events received from this EventBroadcaster to the given
	// event handler function. The return value can be ignored or used to stop recording, if
	// desired. Events("") denotes empty namespace
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})

	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})

	controller := &Controller{
		kubeclientset: kubeclientset,
		clientset:     clientset,
		BackupSynced:  BackupInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "CStorBackup"),
		recorder:      recorder,
	}

	glog.Info("Setting up event handlers for CStorBackup")

	// Clean any pending backup for this cstor pool
	controller.cleanupOldBackup(clientset)

	// Instantiating QueueLoad before entering workqueue.
	q := common.QueueLoad{}

	// Set up an event handler for when BackupCStor resources change.
	BackupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.backupAddFunc(obj, &q)
		},
		UpdateFunc: func(oldVar, newVar interface{}) {
			controller.backupUpdateFunc(oldVar, newVar, &q)
		},
		DeleteFunc: func(obj interface{}) {
			controller.backupDeleteFunc(obj)
		},
	})
	glog.Infof("done with new")
	return controller
}

// enqueueCStorBackup takes a BackupCStor resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than BackupCStor.
func (c *Controller) enqueueCStorBackup(obj *backup.CStorBackup, q common.QueueLoad) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj.GetBackupAPIObject()); err != nil {
		runtime.HandleError(err)
		return
	}
	q.Key = key
	c.workqueue.AddRateLimited(q)
}

// cleanupOldBackup set fail status to old pending backup
func (c *Controller) cleanupOldBackup(backupclient backupclientset.Interface) {
	//TODO add option for lable
	/*
			bkplabel := "cstorpool.openebs.io/uid=" + os.Getenv(string(common.OpenEBSIOCStorID))
		bkplistop := metav1.ListOptions{
			LabelSelector: bkplabel,
		}
	*/
	bkplist, err := backup.NewCStorBackupListBuilder().
		WithClientSet(nil).
		Build()
	if err != nil {
		glog.Errorf("Failed to build backup list : %s", err.Error())
		return
	}

	for _, bkp := range bkplist.Item {
		switch bkp.GetStatus() {
		case backupapi.BKPCStorStatusInProgress:
			//Backup was in in-progress state
			laststat := findLastBackupStat(bkp)
			bkp.SetStatus(laststat)
			_, err = bkp.UpdateCR(bkp)
			if err != nil {
				glog.Errorf("Failed to update status for backup{%s} to {%s} : %s",
					bkp.GetObjName(),
					laststat,
					err.Error())
			}

		case backupapi.BKPCStorStatusDone:
			continue

		default:
			bkp.SetStatus(backupapi.BKPCStorStatusFailed)
			_, err = bkp.UpdateCR(bkp)
			if err != nil {
				glog.Errorf("Failed to update status for backup{%s} to {%s} : %s",
					bkp.GetObjName(),
					backupapi.BKPCStorStatusFailed,
					err.Error())
			}
		}
	}
}

// handleBKPAddEvent is to handle add operation of backup controller
func (c *Controller) backupAddFunc(newb interface{}, q *common.QueueLoad) {
	b := newb.(*backupapi.CStorBackup)

	bkp, err := backup.NewCStorBackupBuilder().BuildFromAPIObject(b)
	if err != nil {
		glog.Errorf("Failed to build object for backup{%s}: %s", b.Name, err.Error())
		return
	}

	if !bkp.IsRightCStorPoolMgmt() {
		return
	}

	q.Operation = common.QOpAdd

	c.recorder.Event(
		bkp.GetBackupAPIObject(),
		corev1.EventTypeNormal,
		string(common.SuccessSynced),
		string(common.MessageCreateSynced))
	c.enqueueCStorBackup(bkp, *q)
	glog.Infof("CStorBackup add-event for backup{%s}", bkp.GetObjName())
}

// handleBKPUpdateEvent is to handle add operation of backup controller
func (c *Controller) backupUpdateFunc(oldb, newb interface{}, q *common.QueueLoad) {
	newbkp := newb.(*backupapi.CStorBackup)
	oldbkp := oldb.(*backupapi.CStorBackup)

	if newbkp.ResourceVersion == oldbkp.ResourceVersion {
		return
	}

	bkp, err := backup.NewCStorBackupBuilder().BuildFromAPIObject(newbkp)
	if err != nil {
		glog.Errorf("Failed to build object for backup{%s}: %s", newbkp.Name, err.Error())
	}

	if !bkp.IsRightCStorPoolMgmt() {
		return
	}

	if bkp.IsDestroyEvent() {
		q.Operation = common.QOpDestroy
		glog.Infof("Destroy event for backup{%s}", newbkp.ObjectMeta.Name)
		//TODO change event type during event record operation
		c.recorder.Event(
			bkp.GetBackupAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessSynced),
			string(common.MessageDestroySynced))
	} else {
		glog.Infof("Modify event for backup{%s}", bkp.GetObjName())
		q.Operation = common.QOpSync
		c.recorder.Event(
			bkp.GetBackupAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessSynced),
			string(common.MessageModifySynced))
	}
	c.enqueueCStorBackup(bkp, *q)
}

func (c *Controller) backupDeleteFunc(obj interface{}) {
	bkp := obj.(*backupapi.CStorBackup)

	b, err := backup.NewCStorBackupBuilder().BuildFromAPIObject(bkp)
	if err != nil {
		glog.Errorf("Failed to build object for backup{%s}: %s", bkp.Name, err.Error())
		return
	}

	if !b.IsRightCStorPoolMgmt() {
		return
	}
	glog.Infof("Delete event for backup{%s}", b.GetObjName())
}

func findLastBackupStat(bkp *backup.CStorBackup) backupapi.CStorBackupStatus {
	lastsnap, err := bkp.GetLastTransferredSnapName()
	if err != nil {
		// Unable to fetch the last backup, so we will return fail state
		glog.Errorf("Failed to fetch last completed backup.. setting failed state for {%s} : %v", bkp.GetObjName(), err)
		return backupapi.BKPCStorStatusFailed
	}

	//TODO
	// let's check if snapname matches with current snapshot name
	if lastsnap == bkp.GetSnapName() {
		return backupapi.BKPCStorStatusDone
	}
	// lastbackup snap/prevsnap doesn't match with bkp snapname
	return backupapi.BKPCStorStatusFailed
}
