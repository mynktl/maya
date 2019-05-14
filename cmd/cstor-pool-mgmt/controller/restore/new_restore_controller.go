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
	restoreapi "github.com/openebs/maya/pkg/apis/openebs.io/restore/v1alpha1"
	restoreclientset "github.com/openebs/maya/pkg/client/generated/openebs.io/restore/v1alpha1/clientset/internalclientset"
	restoreScheme "github.com/openebs/maya/pkg/client/generated/openebs.io/restore/v1alpha1/clientset/internalclientset/scheme"
	informers "github.com/openebs/maya/pkg/client/generated/openebs.io/restore/v1alpha1/informer/externalversions"
	restore "github.com/openebs/maya/pkg/restore/v1alpha1"
)

const controllerName = "CStorRestore"

// Controller is the controller implementation for CStorRestore resources.
type Controller struct {
	// kubeclientset is a standard kubernetes clientset.
	kubeclientset kubernetes.Interface

	// clientset is a openebs custom resource package generated for custom API group.
	clientset restoreclientset.Interface

	// RestoreSynced is used for caches sync to get populated
	RestoreSynced cache.InformerSynced

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

// NewController returns a new cStor restore controller instance
func NewController(
	kubeclientset kubernetes.Interface,
	clientset restoreclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	rstInformerFactory informers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the CStorRestore resources.
	RestoreInformer := rstInformerFactory.Openebs().V1alpha1().CStorRestores()

	err := restoreScheme.AddToScheme(scheme.Scheme)
	if err != nil {
		glog.Errorf("Failed to add restore scheme : %s", err.Error())
		return nil
	}

	// Create event broadcaster
	glog.V(4).Info("Creating restore event broadcaster")
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
		RestoreSynced: RestoreInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "CStorRestore"),
		recorder:      recorder,
	}

	glog.Info("Setting up event handlers for restore")

	// Clean any pending restore for this cstor pool
	controller.cleanupOldRestore(clientset)

	// Instantiating QueueLoad before entering workqueue.
	q := common.QueueLoad{}

	// Set up an event handler for when cStorReplica resources change.
	RestoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.restoreAddFunc(obj, &q)
		},
		UpdateFunc: func(oldVar, newVar interface{}) {
			controller.restoreUpdateFunc(oldVar, newVar, &q)
		},
		DeleteFunc: func(obj interface{}) {
			controller.restoreDeleteFunc(obj)
		},
	})
	glog.Infof("done with new")
	return controller
}

// enqueueCStorRestore takes a CStorRestore resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than CStorRestore.
func (c *Controller) enqueueCStorRestore(rst *restore.CStorRestore, q common.QueueLoad) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(rst.GetRestoreAPIObject()); err != nil {
		runtime.HandleError(err)
		return
	}
	q.Key = key
	c.workqueue.AddRateLimited(q)
}

// restoreAddFunc is to handle add operation of restore controller
func (c *Controller) restoreAddFunc(newr interface{}, q *common.QueueLoad) {
	r := newr.(*restoreapi.CStorRestore)

	rst, err := restore.NewCStorRestoreBuilder().
		BuildFromAPIObject(r)
	if err != nil {
		glog.Errorf("Failed to build object for restore{%s}: %s", r.Name, err.Error())
		return
	}

	if !rst.IsRightCStorPoolMgmt() {
		return
	}

	q.Operation = common.QOpAdd
	c.recorder.Event(
		rst.GetRestoreAPIObject(),
		corev1.EventTypeNormal,
		string(common.SuccessSynced),
		string(common.MessageCreateSynced))
	c.enqueueCStorRestore(rst, *q)
	glog.Infof("CStorRestore add-event for restore{%s}", rst.GetObjName())
}

func (c *Controller) restoreUpdateFunc(oldr, newr interface{}, q *common.QueueLoad) {
	newrst := newr.(*restoreapi.CStorRestore)
	oldrst := oldr.(*restoreapi.CStorRestore)

	// If there is no change in status then we will ignore the event
	if newrst.Status == oldrst.Status {
		return
	}

	rst, err := restore.NewCStorRestoreBuilder().BuildFromAPIObject(newrst)
	if err != nil {
		glog.Errorf("Failed to build object for restore{%s}: %s", newrst.Name, err.Error())
		return
	}

	if !rst.IsRightCStorPoolMgmt() {
		return
	}

	if rst.IsDestroyEvent() {
		q.Operation = common.QOpDestroy
		glog.Infof("Destroy event for restore{%s}", rst.GetObjName())
		//TODO change event type during event record operation
		c.recorder.Event(
			rst.GetRestoreAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessSynced),
			string(common.MessageDestroySynced))
	} else {
		glog.Infof("Modify event for restore{%s}", rst.GetObjName())
		q.Operation = common.QOpSync
		c.recorder.Event(
			rst.GetRestoreAPIObject(),
			corev1.EventTypeNormal,
			string(common.SuccessSynced),
			string(common.MessageModifySynced))
	}
	c.enqueueCStorRestore(rst, *q)
}

// cleanupOldRestore set fail status to old pending restore
func (c *Controller) cleanupOldRestore(clientset restoreclientset.Interface) {
	//TODO add option for lable
	/*
		rstlabel := "cstorpool.openebs.io/uid=" + os.Getenv(string(common.OpenEBSIOCStorID))
		rstlistop := metav1.ListOptions{
			LabelSelector: rstlabel,
		}
	*/

	rstlist, err := restore.NewCStorRestoreListBuilder().
		WithClientSet(nil).
		Build()
	if err != nil {
		glog.Errorf("Failed to build restore list : %s", err.Error())
		return
	}

	for _, rst := range rstlist.Item {
		switch rst.GetStatus() {
		case restoreapi.RSTCStorStatusDone:
			continue

		default:
			//Set restore status as failed
			rst.SetStatus(restoreapi.RSTCStorStatusFailed)
			_, err = rst.UpdateCR(rst)
			if err != nil {
				glog.Errorf("Failed to update status for restore{%s} to {%s} : %s",
					rst.GetObjName(),
					restoreapi.RSTCStorStatusFailed,
					err.Error())
			}
		}
	}
}

func (c *Controller) restoreDeleteFunc(obj interface{}) {
	rst := obj.(*restoreapi.CStorRestore)

	r, err := restore.NewCStorRestoreBuilder().BuildFromAPIObject(rst)
	if err != nil {
		glog.Errorf("Failed to build object for restore{%s}: %s", rst.Name, err.Error())
		return
	}

	if !r.IsRightCStorPoolMgmt() {
		return
	}
	glog.Infof("Delete event for restore{%s}", r.GetObjName())
	//TODO add delete event handling
}
