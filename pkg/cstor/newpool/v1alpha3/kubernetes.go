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

package v1alpha3

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	errors "github.com/openebs/maya/pkg/errors/v1alpha1"
	kclient "github.com/openebs/maya/pkg/kubernetes/client/v1alpha1"

	apis "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	clientset "github.com/openebs/maya/pkg/client/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/types"
)

//TODO: While using these packages UnitTest must be written to corresponding function

// getClientsetFn is a typed function that
// abstracts fetching of internal clientset
type getClientsetFn func() (clientset *clientset.Clientset, err error)

// getClientsetFromPathFn is a typed function that
// abstracts fetching of clientset from kubeConfigPath
type getClientsetForPathFn func(kubeConfigPath string) (clientset *clientset.Clientset, err error)

// listFn is a typed function that abstracts
// listing of csp
type listFn func(cli *clientset.Clientset, namespace string, opts metav1.ListOptions) (*apis.NewTestCStorPoolList, error)

// getFn is a typed function that
// abstracts fetching of csp
type getFn func(cli *clientset.Clientset, namespace, name string, opts metav1.GetOptions) (*apis.NewTestCStorPool, error)

// createFn is a typed function that abstracts
// creation of csp
type createFn func(cli *clientset.Clientset, namespace string, csp *apis.NewTestCStorPool) (*apis.NewTestCStorPool, error)

// deleteFn is a typed function that abstracts
// deletion of csps
type deleteFn func(cli *clientset.Clientset, namespace string, name string, deleteOpts *metav1.DeleteOptions) error

// deleteFn is a typed function that abstracts
// deletion of csp's collection
type deleteCollectionFn func(cli *clientset.Clientset, namespace string, listOpts metav1.ListOptions, deleteOpts *metav1.DeleteOptions) error

// patchFn is a typed function that abstracts
// to patch csp claim
type patchFn func(cli *clientset.Clientset, namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*apis.NewTestCStorPool, error)

// Kubeclient enables kubernetes API operations
// on csp instance
type Kubeclient struct {
	// clientset refers to csp
	// clientset that will be responsible to
	// make kubernetes API calls
	clientset *clientset.Clientset
	// kubeconfig path to get kubernetes clientset
	kubeConfigPath string
	namespace      string
	// functions useful during mocking
	getClientset        getClientsetFn
	getClientsetForPath getClientsetForPathFn
	list                listFn
	get                 getFn
	create              createFn
	del                 deleteFn
	delCollection       deleteCollectionFn
	patch               patchFn
}

// KubeclientBuildOption defines the abstraction
// to build a kubeclient instance
type KubeclientBuildOption func(*Kubeclient)

// WithDefaults sets the default options
// of kubeclient instance
func (k *Kubeclient) WithDefaults() {
	if k.getClientset == nil {
		k.getClientset = func() (clients *clientset.Clientset, err error) {
			config, err := kclient.New().Config()
			if err != nil {
				return nil, err
			}
			return clientset.NewForConfig(config)
		}
	}
	if k.getClientsetForPath == nil {
		k.getClientsetForPath = func(kubeConfigPath string) (clients *clientset.Clientset, err error) {
			config, err := kclient.New(kclient.WithKubeConfigPath(kubeConfigPath)).Config()
			if err != nil {
				return nil, err
			}
			return clientset.NewForConfig(config)
		}
	}
	if k.list == nil {
		k.list = func(cli *clientset.Clientset, namespace string, opts metav1.ListOptions) (*apis.NewTestCStorPoolList, error) {
			return cli.OpenebsV1alpha1().NewTestCStorPools(namespace).List(opts)
		}
	}

	if k.get == nil {
		k.get = func(cli *clientset.Clientset, namespace, name string, opts metav1.GetOptions) (*apis.NewTestCStorPool, error) {
			return cli.OpenebsV1alpha1().NewTestCStorPools(namespace).Get(name, opts)
		}
	}
	if k.create == nil {
		k.create = func(cli *clientset.Clientset, namespace string, csp *apis.NewTestCStorPool) (*apis.NewTestCStorPool, error) {
			return cli.OpenebsV1alpha1().NewTestCStorPools(namespace).Create(csp)
		}
	}
	if k.del == nil {
		k.del = func(cli *clientset.Clientset, namespace string, name string, deleteOpts *metav1.DeleteOptions) error {
			return cli.OpenebsV1alpha1().NewTestCStorPools(namespace).Delete(name, deleteOpts)
		}
	}
	if k.delCollection == nil {
		k.delCollection = func(cli *clientset.Clientset, namespace string, listOpts metav1.ListOptions, deleteOpts *metav1.DeleteOptions) error {
			return cli.OpenebsV1alpha1().NewTestCStorPools(namespace).DeleteCollection(deleteOpts, listOpts)
		}
	}
	if k.patch == nil {
		k.patch = func(cli *clientset.Clientset, namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*apis.NewTestCStorPool, error) {
			return cli.OpenebsV1alpha1().NewTestCStorPools(namespace).Patch(name, pt, data, subresources...)
		}
	}
}

// WithKubeClient sets the kubernetes client against
// the kubeclient instance
func WithKubeClient(c *clientset.Clientset) KubeclientBuildOption {
	return func(k *Kubeclient) {
		k.clientset = c
	}
}

// WithKubeConfigPath sets the kubeConfig path
// against client instance
func WithKubeConfigPath(kubeConfigPath string) KubeclientBuildOption {
	return func(k *Kubeclient) {
		k.kubeConfigPath = kubeConfigPath
	}
}

// NewKubeClient returns a new instance of kubeclient meant for
// csp operations
func NewKubeClient(opts ...KubeclientBuildOption) *Kubeclient {
	k := &Kubeclient{}
	for _, o := range opts {
		o(k)
	}
	k.WithDefaults()
	return k
}

func (k *Kubeclient) getClientsetForPathOrDirect() (*clientset.Clientset, error) {
	if k.kubeConfigPath != "" {
		return k.getClientsetForPath(k.kubeConfigPath)
	}
	return k.getClientset()
}

// WithNamespace sets the kubernetes namespace against
// the provided namespace
func (k *Kubeclient) WithNamespace(namespace string) *Kubeclient {
	k.namespace = namespace
	return k
}

// getClientOrCached returns either a new instance
// of kubernetes client or its cached copy
func (k *Kubeclient) getClientsetOrCached() (*clientset.Clientset, error) {
	if k.clientset != nil {
		return k.clientset, nil
	}
	c, err := k.getClientsetForPathOrDirect()
	if err != nil {
		return nil, err
	}
	k.clientset = c
	return k.clientset, nil
}

// List returns a list of disk
// instances present in kubernetes cluster
func (k *Kubeclient) List(opts metav1.ListOptions) (*apis.NewTestCStorPoolList, error) {
	cli, err := k.getClientsetOrCached()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list csp in namespace {%s}", k.namespace)
	}
	return k.list(cli, k.namespace, opts)
}

// Get returns a disk object
func (k *Kubeclient) Get(name string, opts metav1.GetOptions) (*apis.NewTestCStorPool, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("failed to get csp: missing csp name")
	}
	cli, err := k.getClientsetOrCached()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get csp {%s} in namespace {%s}", name, k.namespace)
	}
	return k.get(cli, k.namespace, name, opts)
}

// Create creates a csp in specified namespace in kubernetes cluster
func (k *Kubeclient) Create(csp *apis.NewTestCStorPool) (*apis.NewTestCStorPool, error) {
	if csp == nil {
		return nil, errors.New("failed to create csp: nil csp object")
	}
	cli, err := k.getClientsetOrCached()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create csp {%s} in namespace {%s}", csp.Name, csp.Namespace)
	}
	return k.create(cli, k.namespace, csp)
}

// DeleteCollection deletes a collection of csp objects.
func (k *Kubeclient) DeleteCollection(listOpts metav1.ListOptions, deleteOpts *metav1.DeleteOptions) error {
	cli, err := k.getClientsetOrCached()
	if err != nil {
		return errors.Wrapf(err, "failed to delete the collection of csps")
	}
	return k.delCollection(cli, k.namespace, listOpts, deleteOpts)
}

// Delete deletes a csp instance from the
// kubecrnetes cluster
func (k *Kubeclient) Delete(name string, deleteOpts *metav1.DeleteOptions) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("failed to delete csp: missing csp name")
	}
	cli, err := k.getClientsetOrCached()
	if err != nil {
		return errors.Wrapf(err, "failed to delete csp {%s} in namespace {%s}", name, k.namespace)
	}
	return k.del(cli, k.namespace, name, deleteOpts)
}

// Patch patches the csp claim if present in kubernetes cluster
func (k *Kubeclient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*apis.NewTestCStorPool, error) {
	if len(name) == 0 {
		return nil, errors.New("failed to patch csp : missing csp name")
	}
	cli, err := k.getClientsetOrCached()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch csp: {%s}", name)
	}
	return k.patch(cli, k.namespace, name, pt, data, subresources...)
}
