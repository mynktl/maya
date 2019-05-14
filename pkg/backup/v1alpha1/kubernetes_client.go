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

package backup

import (
	apis "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Get is Kubernetes client implementation to get backup.
func (k *KubeClient) Get(name, ns string) (*apis.CStorBackup, error) {
	return k.client.
		OpenebsV1alpha1().
		CStorBackups(ns).
		Get(name, v1.GetOptions{})
}

// List is kubernetes client implementation to list backup.
func (k *KubeClient) List(ns string, opts v1.ListOptions) (*apis.CStorBackupList, error) {
	return k.client.
		OpenebsV1alpha1().
		CStorBackups(ns).
		List(opts)
}

// Create is kubernetes client implementation to create backup.
func (k *KubeClient) Create(bkpobj *apis.CStorBackup) (*apis.CStorBackup, error) {
	return k.client.
		OpenebsV1alpha1().
		CStorBackups(bkpobj.GetNamespace()).
		Create(bkpobj)
}

// Update is kubernetes client implementation to update backup.
func (k *KubeClient) Update(bkpobj *apis.CStorBackup) (*apis.CStorBackup, error) {
	return k.client.
		OpenebsV1alpha1().
		CStorBackups(bkpobj.GetNamespace()).
		Update(bkpobj)
}
