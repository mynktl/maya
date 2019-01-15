/*
Copyright 2018 The OpenEBS Authors

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCStorRestoreDatas implements CStorRestoreDataInterface
type FakeCStorRestoreDatas struct {
	Fake *FakeOpenebsV1alpha1
	ns   string
}

var cstorrestoredatasResource = schema.GroupVersionResource{Group: "openebs.io", Version: "v1alpha1", Resource: "cstorrestoredatas"}

var cstorrestoredatasKind = schema.GroupVersionKind{Group: "openebs.io", Version: "v1alpha1", Kind: "CStorRestoreData"}

// Get takes name of the cStorRestoreData, and returns the corresponding cStorRestoreData object, and an error if there is any.
func (c *FakeCStorRestoreDatas) Get(name string, options v1.GetOptions) (result *v1alpha1.CStorRestoreData, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(cstorrestoredatasResource, c.ns, name), &v1alpha1.CStorRestoreData{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CStorRestoreData), err
}

// List takes label and field selectors, and returns the list of CStorRestoreDatas that match those selectors.
func (c *FakeCStorRestoreDatas) List(opts v1.ListOptions) (result *v1alpha1.CStorRestoreDataList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(cstorrestoredatasResource, cstorrestoredatasKind, c.ns, opts), &v1alpha1.CStorRestoreDataList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.CStorRestoreDataList{ListMeta: obj.(*v1alpha1.CStorRestoreDataList).ListMeta}
	for _, item := range obj.(*v1alpha1.CStorRestoreDataList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cStorRestoreDatas.
func (c *FakeCStorRestoreDatas) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(cstorrestoredatasResource, c.ns, opts))

}

// Create takes the representation of a cStorRestoreData and creates it.  Returns the server's representation of the cStorRestoreData, and an error, if there is any.
func (c *FakeCStorRestoreDatas) Create(cStorRestoreData *v1alpha1.CStorRestoreData) (result *v1alpha1.CStorRestoreData, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(cstorrestoredatasResource, c.ns, cStorRestoreData), &v1alpha1.CStorRestoreData{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CStorRestoreData), err
}

// Update takes the representation of a cStorRestoreData and updates it. Returns the server's representation of the cStorRestoreData, and an error, if there is any.
func (c *FakeCStorRestoreDatas) Update(cStorRestoreData *v1alpha1.CStorRestoreData) (result *v1alpha1.CStorRestoreData, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(cstorrestoredatasResource, c.ns, cStorRestoreData), &v1alpha1.CStorRestoreData{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CStorRestoreData), err
}

// Delete takes name of the cStorRestoreData and deletes it. Returns an error if one occurs.
func (c *FakeCStorRestoreDatas) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(cstorrestoredatasResource, c.ns, name), &v1alpha1.CStorRestoreData{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCStorRestoreDatas) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(cstorrestoredatasResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.CStorRestoreDataList{})
	return err
}

// Patch applies the patch and returns the patched cStorRestoreData.
func (c *FakeCStorRestoreDatas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CStorRestoreData, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(cstorrestoredatasResource, c.ns, name, data, subresources...), &v1alpha1.CStorRestoreData{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CStorRestoreData), err
}
