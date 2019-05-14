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

const cstorBackupYamls = `
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: cstorbackups.openebs.io
spec:
  group: openebs.io
  version: v1alpha1
  scope: Namespaced
  names:
    plural: cstorbackups
    singular: cstorbackup
    kind: CStorBackup
    shortNames:
    - bkp
    - bkps
    - backups
    - backup
  additionalPrinterColumns:
    - JSONPath: .spec.volumeName
      name: volume
      description: volume on which backup performed
      type: string
    - JSONPath: .spec.backupName
      name: backup/schedule
      description: Backup/schedule name
      type: string
    - JSONPath: .status
      name: Status
      description: Backup status
      type: string
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: cstorbackupcompleteds.openebs.io
spec:
  group: openebs.io
  version: v1alpha1
  scope: Namespaced
  names:
    plural: cstorbackupcompleteds
    singular: cstorbackupcompleted
    kind: CStorBackupCompleted
    shortNames:
    - bkpcompleted
    - backupcompleted
  additionalPrinterColumns:
    - JSONPath: .spec.volumeName
      name: volume
      description: volume on which backup performed
      type: string
    - JSONPath: .spec.backupName
      name: backup/schedule
      description: Backup/schedule name
      type: string
    - JSONPath: .spec.prevSnapName
      name: lastSnap
      description: Last successful backup snapshot
      type: string
---`

// CstorSnapshotArtifacts returns the cstor snapshot related artifacts
// corresponding to latest version
func CStorBackupArtifacts() (list artifactList) {
	list.Items = append(list.Items, ParseArtifactListFromMultipleYamls(cstorBackups{})...)
	return
}

type cstorBackups struct{}

// FetchYamls returns all the yamls related to cstor backup in a string
// format
//
// NOTE:
//  This is an implementation of MultiYamlFetcher
func (b cstorBackups) FetchYamls() string {
	return cstorBackupYamls
}
