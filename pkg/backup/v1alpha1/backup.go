package backup

import (
	"os"
	"reflect"

	"github.com/openebs/maya/cmd/cstor-pool-mgmt/controller/common"
	apis "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
)

// IsRightCStorPoolMgmt check if CStorBackup metaID matches with pool CStorID
func IsRightCStorPoolMgmt(bkp *CStorBackup) bool {
	return os.Getenv(string(common.OpenEBSIOCStorID)) == string(bkp.object.ObjectMeta.UID)
}

// GetBackupAPIObject returns api object from CStorBackup object
func (b *CStorBackup) GetBackupAPIObject() *apis.CStorBackup {
	return b.object
}

// GetObjName returns name of the CStorObject
func (b *CStorBackup) GetObjName() string {
	return b.object.Name
}

// GetBackupName returns backup name
func (b *CStorBackup) GetBackupName() string {
	return b.object.Spec.BackupName
}

// GetVolumeName returns volume name
func (b *CStorBackup) GetVolumeName() string {
	return b.object.Spec.VolumeName
}

// GetSnapName returns snapshot name
func (b *CStorBackup) GetSnapName() string {
	return b.object.Spec.SnapName
}

// GetPrevSnapName returns previous snapshot name
func (b *CStorBackup) GetPrevSnapName() string {
	return b.object.Spec.PrevSnapName
}

// SetPrevSnapName updates previous snapshot name with given snap name
func (b *CStorBackup) SetPrevSnapName(prevsnapname string) {
	b.object.Spec.PrevSnapName = prevsnapname
	return
}

// GetBackupDest returns backup destination address
func (b *CStorBackup) GetBackupDest() string {
	return b.object.Spec.BackupDest
}

// GetNamespace returns namespace of object
func (b *CStorBackup) GetNamespace() string {
	return b.object.Namespace
}

// GetStatus returns status of object
func (b *CStorBackup) GetStatus() apis.CStorBackupStatus {
	return b.object.Status
}

// SetStatus updates status of object to given status
func (b *CStorBackup) SetStatus(status apis.CStorBackupStatus) {
	b.object.Status = status
	return
}

// GetLabel returns label value for given key
func (b *CStorBackup) GetLabel(label string) string {
	return b.object.Labels[label]
}

// GetLabels returns labels of the CStorBackup object
func (b *CStorBackup) GetLabels() map[string]string {
	return b.object.Labels
}

// SetLabel updates the label of object with given label
//TODO check append
func (b *CStorBackup) SetLabel(label map[string]string) {
	b.object.ObjectMeta.Labels = label
	return
}

// SetObjName set CStorBackup object name
// object name includes snapshot name and volume name
func (b *CStorBackup) SetObjName() {
	b.object.Name = b.object.Spec.SnapName + "-" + b.object.Spec.VolumeName
}

// DeriveObjName derives backup name for object from snapname and volumename
func DeriveObjName(a *apis.CStorBackup) string {
	return a.Spec.SnapName + "-" + a.Spec.VolumeName
}

// GetCompletedBackupName is to get last sucessfull completed backup name
// for given CStorBackup object
func (b *CStorBackup) GetCompletedBackupName() string {
	return b.object.Spec.BackupName + "-" + b.object.Spec.VolumeName
}

// IsPendingStatus is to check if the backup is in a pending state.
func (b *CStorBackup) IsPendingStatus() bool {
	return string(b.object.Status) == string(apis.BKPCStorStatusPending)
}

// IsInProgressStatus is to check if the backup is in in-progress state.
func (b *CStorBackup) IsInProgressStatus() bool {
	return string(b.object.Status) == string(apis.BKPCStorStatusInProgress)
}

// IsInitStatus is to check if the backup is in init state.
func (b *CStorBackup) IsInitStatus() bool {
	return string(b.object.Status) == string(apis.BKPCStorStatusInit)
}

// IsDoneStatus is to check if the backup is completed or not
func (b *CStorBackup) IsDoneStatus() bool {
	return string(b.object.Status) == string(apis.BKPCStorStatusDone)
}

// IsFailedStatus is to check if the backup is failed or not
func (b *CStorBackup) IsFailedStatus() bool {
	return string(b.object.Status) == string(apis.BKPCStorStatusFailed)
}

// IsDestroyEvent is to check if the call is for backup destroy.
func (b *CStorBackup) IsDestroyEvent() bool {
	return b.object.ObjectMeta.DeletionTimestamp != nil
}

// IsOnlyStatusChange is to check the only status change of CStorBackup object.
func (b *CStorBackup) IsOnlyStatusChange(newbkp *CStorBackup) bool {
	return reflect.DeepEqual(b.object.Spec, newbkp.object.Spec) &&
		!reflect.DeepEqual(b.object.Status, newbkp.object.Status)
}

// SetPrevSnapNameFromLastBackup updates previous snap name to
// last sucessful transferred snapshot name
func (b *CStorBackup) SetPrevSnapNameFromLastBackup() error {
	lastsnap, err := b.GetLastTransferredSnapName()
	if err != nil {
		//TODO wrap error
		return err
	}
	b.SetPrevSnapName(lastsnap)
	return nil
}

// GetCR return CStorBackup object using given name
func (b *CStorBackup) GetCR(name string) (*CStorBackup, error) {
	ba, err := b.Get(name, b.object.Namespace)
	if err == nil {
		return NewCStorBackupBuilder().
			WithClientSet(nil).
			BuildFromAPIObject(ba)
	}
	return nil, err
}

// CreateCR create CStorBackup CR from given object
func (b *CStorBackup) CreateCR(obj *CStorBackup) (*CStorBackup, error) {
	ba, err := b.Create(obj.GetBackupAPIObject())
	if err == nil {
		return NewCStorBackupBuilder().
			WithClientSet(nil).
			BuildFromAPIObject(ba)
	}
	return nil, err
}

// UpdateCR updates the given CStorBackup object
func (b *CStorBackup) UpdateCR(obj *CStorBackup) (*CStorBackup, error) {
	ba, err := b.Update(obj.GetBackupAPIObject())
	if err == nil {
		return NewCStorBackupBuilder().
			WithClientSet(nil).
			BuildFromAPIObject(ba)
	}
	return nil, err
}

// UpdateCRStatus updates the status of `name` backup with given status
func (b *CStorBackup) UpdateCRStatus(name string, status apis.CStorBackupStatus) error {
	ba, err := b.Get(name, b.object.Namespace)
	if err == nil {
		ba.Status = status
		_, err := b.Update(ba)
		return err
	}
	return err
}
