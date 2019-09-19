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
package app

import (
	"testing"
	"time"

	"github.com/openebs/maya/cmd/cstor-pool-mgmt/controller/common"
	apis "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	openebsFakeClientset "github.com/openebs/maya/pkg/client/generated/clientset/versioned/fake"
	informers "github.com/openebs/maya/pkg/client/generated/informers/externalversions"

	"github.com/openebs/maya/pkg/signals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

var testNS = "test"

// TestRun is to run cStorPoolInstance controller and check if it crashes or return back.
func TestRun(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Pool Instance controllers.
	poolController := NewCStorPoolInstanceController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	stopCh := signals.SetupSignalHandler()
	done := make(chan bool)
	go func(chan bool) {
		poolController.Run(2, stopCh)
		done <- true
	}(done)

	select {
	case <-time.After(3 * time.Second):

	case <-done:
		t.Fatalf("CStorPool controller returned - failure")

	}
}

// TestProcessNextWorkItemModify is to test a cStorPoolInstance resource for modify event.
func TestProcessNextWorkItemModify(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Pool Instance controllers.
	poolController := NewCStorPoolInstanceController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	testPoolResource := map[string]struct {
		expectedOutput bool
		test           *apis.CStorPoolInstance
	}{
		"img2PoolResource": {
			expectedOutput: true,
			test: &apis.CStorPoolInstance{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "pool2",
					UID:        types.UID("abcd"),
					Finalizers: []string{"cstorpoolinstance.openebs.io/finalizer"},
				},
				Spec: apis.CStorPoolInstanceSpec{
					PoolConfig: apis.PoolConfig{
						CacheFile:            "/tmp/pool2.cache",
						DefaultRaidGroupType: "striped",
						OverProvisioning:     false,
					},
				},
				Status: apis.CStorPoolStatus{},
			},
		},
	}

	_, err := poolController.clientset.OpenebsV1alpha1().CStorPoolInstances(testNS).Create(testPoolResource["img2PoolResource"].test)
	if err != nil {
		t.Fatalf("Unable to create resource : %v", testPoolResource["img2PoolResource"].test.ObjectMeta.Name)
	}

	poolController.workqueue.AddRateLimited(common.QueueLoad{
		Key:       "pool2",
		Operation: "modify",
	})

	obtainedOutput := poolController.processNextWorkItem()
	if obtainedOutput != testPoolResource["img2PoolResource"].expectedOutput {
		t.Fatalf("Expected:%v, Got:%v", testPoolResource["img2PoolResource"].expectedOutput,
			obtainedOutput)
	}
}

// TestProcessNextWorkItemDestroy is to test a cStorPoolInstance resource for destroy event.
func TestProcessNextWorkItemDestroy(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset()
	fakeOpenebsClient := openebsFakeClientset.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Second*30)
	openebsInformerFactory := informers.NewSharedInformerFactory(fakeOpenebsClient, time.Second*30)

	// Instantiate the cStor Pool Instance controllers.
	poolController := NewCStorPoolInstanceController(fakeKubeClient, fakeOpenebsClient, kubeInformerFactory,
		openebsInformerFactory)

	testPoolResource := map[string]struct {
		expectedOutput bool
		test           *apis.CStorPoolInstance
	}{
		"img2PoolResource": {
			expectedOutput: true,
			test: &apis.CStorPoolInstance{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "pool2",
					UID:        types.UID("abcd"),
					Finalizers: []string{"cstorpoolinstance.openebs.io/finalizer"},
				},
				Spec: apis.CStorPoolInstanceSpec{
					PoolConfig: apis.PoolConfig{
						CacheFile:            "/tmp/pool2.cache",
						DefaultRaidGroupType: "striped",
						OverProvisioning:     false,
					},
				},
				Status: apis.CStorPoolStatus{},
			},
		},
	}

	_, err := poolController.clientset.OpenebsV1alpha1().CStorPoolInstances(testNS).Create(testPoolResource["img2PoolResource"].test)
	if err != nil {
		t.Fatalf("Unable to create resource : %v", testPoolResource["img2PoolResource"].test.ObjectMeta.Name)
	}

	var q common.QueueLoad
	q.Key = "pool2"
	q.Operation = "destroy"
	poolController.workqueue.AddRateLimited(common.QueueLoad{
		Key:       "pool2",
		Operation: "destroy",
	})

	obtainedOutput := poolController.processNextWorkItem()
	if obtainedOutput != testPoolResource["img2PoolResource"].expectedOutput {
		t.Fatalf("Expected:%v, Got:%v", testPoolResource["img2PoolResource"].expectedOutput,
			obtainedOutput)
	}
}
