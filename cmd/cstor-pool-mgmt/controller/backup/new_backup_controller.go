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

const backupControllerName = "CStorBackup"

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

// NewCStorBackupController returns a new cStor Backup controller instance
func NewCStorBackupController(
	kubeclientset kubernetes.Interface,
	clientset backupclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	bkpInformerFactory informers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the BackupCStor resources.
	BackupInformer := bkpInformerFactory.Backup().V1alpha1().CStorBackups()

	err := backupScheme.AddToScheme(scheme.Scheme)
	if err != nil {
		glog.Fatalf("Error adding scheme to openebs scheme: %s", err.Error())
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

	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: backupControllerName})

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
			glog.Infof("Received Update for backup")
			controller.backupAddFunc(obj, &q)
		},
		UpdateFunc: func(oldVar, newVar interface{}) {
			glog.Infof("Received Update for backup")
			controller.backupUpdateFunc(oldVar, newVar, &q)
		},
		DeleteFunc: func(obj interface{}) {
			glog.Infof("Received Update for backup")
			controller.backupDeleteFunc(obj)
		},
	})
	glog.Infof("done with new")
	return controller
}

// enqueueBackupCStor takes a BackupCStor resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than BackupCStor.
func (c *Controller) enqueueBackupCStor(obj *backup.CStorBackup, q common.QueueLoad) {
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
		return
	}

	for _, bkp := range bkplist.Item {
		switch bkp.GetStatus() {
		case backupapi.BKPCStorStatusInProgress:
			//Backup was in in-progress state
			//TODO builder
			laststat, _ := findLastBackupStat(bkp)
			//TODO handler error
			//TODO create a copy and update istead of using same resource
			bkp.SetStatus(laststat)
			glog.Infof("adasd bkp %v %v", bkp, bkp.KubeClient)
			_, err = bkp.UpdateCR(bkp)
			if err != nil {
				//TOO log error
			}

		case backupapi.BKPCStorStatusDone:
			continue
		default:
			//TODO create a copy and update istead of using same resource
			bkp.SetStatus(backupapi.BKPCStorStatusFailed)
			_, err = bkp.UpdateCR(bkp)
			if err != nil {
				//TODO log error
			}
		}
	}
}

// handleBKPAddEvent is to handle add operation of backup controller
func (c *Controller) backupAddFunc(newb interface{}, q *common.QueueLoad) {
	b := newb.(*backupapi.CStorBackup)

	bkp, err := backup.NewCStorBackupBuilder().BuildFromAPIObject(b)
	if err != nil {
		//TODO log error
	}

	/*
		if !c.IsRightCStorPoolMgmt(bkp.GetBackupAPIObject()) {
			return
		}
	*/

	q.Operation = common.QOpAdd
	c.recorder.Event(
		bkp.GetBackupAPIObject(),
		corev1.EventTypeNormal,
		string(common.SuccessSynced),
		string(common.MessageCreateSynced))
	c.enqueueBackupCStor(bkp, *q)
	glog.Infof("BackupCStor event added for backup {%v}", bkp.GetObjName())
}

// handleBKPUpdateEvent is to handle add operation of backup controller
func (c *Controller) backupUpdateFunc(oldb, newb interface{}, q *common.QueueLoad) {
	// Note : UpdateFunc is called in following three cases:
	newbkp := newb.(*backupapi.CStorBackup)
	oldbkp := oldb.(*backupapi.CStorBackup)
	/*
		if !newbkp.IsRightCStorPoolMgmt() {
			return
		}
	*/
	glog.Infof("Received Update for backup:%s", oldbkp.ObjectMeta.Name)

	if newbkp.ResourceVersion == oldbkp.ResourceVersion {
		glog.Infof("same version no change")
		return
	}

	bkp, err := backup.NewCStorBackupBuilder().BuildFromAPIObject(newbkp)
	if err != nil {
		//TODO log error
	}

	if bkp.IsDestroyEvent() {
		q.Operation = common.QOpDestroy
		glog.Infof("BackupCstor Destroy event : %v, %v", newbkp.ObjectMeta.Name, string(newbkp.ObjectMeta.UID))
		//TODO change event type during event record operation
		c.recorder.Event(
			bkp.GetBackupAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessSynced),
			string(common.MessageDestroySynced))
	} else {
		glog.Infof("BackupCstor Modify event : %v, %v", newbkp.ObjectMeta.Name, string(newbkp.ObjectMeta.UID))
		q.Operation = common.QOpSync
		c.recorder.Event(
			bkp.GetBackupAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessSynced),
			string(common.MessageModifySynced))
	}
	c.enqueueBackupCStor(bkp, *q)
}

func (c *Controller) backupDeleteFunc(obj interface{}) {
	bkp := obj.(*backupapi.CStorBackup)
	/*
		if !IsRightCStorPoolMgmt(bkp) {
			return
		}
	*/
	glog.Infof("BackupCStor Resource delete event: %v, %v", bkp.ObjectMeta.Name, string(bkp.ObjectMeta.UID))
	//TODO add delete event handling
}

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
