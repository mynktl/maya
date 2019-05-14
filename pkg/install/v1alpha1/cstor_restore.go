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
package v1alpha1

const cstorRestoreYamls = `
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: cstorrestores.openebs.io
spec:
  group: openebs.io
  version: v1alpha1
  scope: Namespaced
  names:
    plural: cstorrestores
    singular: cstorrestore
    kind: CStorRestore
    shortNames:
    - crst
    - crsts
    - crestores
    - crestore
  additionalPrinterColumns:
    - JSONPath: .spec.restoreName
      name: backup
      description: backup name which is  restored
      type: string
    - JSONPath: .spec.volumeName
      name: volume
      description: volume on which restore performed
      type: string
    - JSONPath: .status
      name: Status
      description: Restore status
      type: string
---`

// CStorRestoreArtifacts returns the cstor restore related artifacts
// corresponding to latest version
func CStorRestoreArtifacts() (list artifactList) {
	list.Items = append(list.Items, ParseArtifactListFromMultipleYamls(cstorRestores{})...)
	return
}

type cstorRestores struct{}

// FetchYamls returns all the yamls related to cstor restore in a string
// format
//
// NOTE:
//  This is an implementation of MultiYamlFetcher
func (b cstorRestores) FetchYamls() string {
	return cstorRestoreYamls
}
