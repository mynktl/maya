package restore

import (
	"os"
	"reflect"

	"github.com/openebs/maya/cmd/cstor-pool-mgmt/controller/common"
	apis "github.com/openebs/maya/pkg/apis/openebs.io/restore/v1alpha1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	// PoolUUID is key for pool UUID
	PoolUUID = "cstorpool.openebs.io/uid"
)

// IsRightCStorPoolMgmt check if CStorRestore metaID matches with pool CStorID
func (r *CStorRestore) IsRightCStorPoolMgmt() bool {
	return os.Getenv(string(common.OpenEBSIOCStorID)) == r.GetLabel(PoolUUID)
}

// GetRestoreAPIObject returns api object from CStorRestore object
func (r *CStorRestore) GetRestoreAPIObject() *apis.CStorRestore {
	return r.object
}

// GetObjName returns name of the CStorObject
func (r *CStorRestore) GetObjName() string {
	return r.object.Name
}

// GetRestoreName returns restore name
func (r *CStorRestore) GetRestoreName() string {
	return r.object.Spec.RestoreName
}

// GetVolumeName returns volume name
func (r *CStorRestore) GetVolumeName() string {
	return r.object.Spec.VolumeName
}

// GetRestoreSrc returns restore source address
func (r *CStorRestore) GetRestoreSrc() string {
	return r.object.Spec.RestoreSrc
}

// GetNamespace returns namespace of object
func (r *CStorRestore) GetNamespace() string {
	return r.object.Namespace
}

// GetStatus returns status of object
func (r *CStorRestore) GetStatus() apis.CStorRestoreStatus {
	return r.object.Status
}

// SetStatus updates status of object to given status
func (r *CStorRestore) SetStatus(status apis.CStorRestoreStatus) {
	r.object.Status = status
	return
}

// GetLabel returns label value for given key
func (r *CStorRestore) GetLabel(label string) string {
	return r.object.Labels[label]
}

// GetLabels returns labels of the CStorRestore object
func (r *CStorRestore) GetLabels() map[string]string {
	return r.object.Labels
}

// SetLabel updates the label of object with given label
//TODO check append
func (r *CStorRestore) SetLabel(label map[string]string) {
	r.object.ObjectMeta.Labels = label
	return
}

// SetObjName set CStorRestore object name
// object name includes snapshot name and volume name
func (r *CStorRestore) SetObjName() {
	r.object.Name = r.object.Spec.RestoreName + "-" + string(uuid.NewUUID())
}

// RegenerateObjName regenerate CStorRestore object's name
func (r *CStorRestore) RegenerateObjName() {
	r.SetObjName()
}

// Copy spec from new CStorRestore object
func (r *CStorRestore) CopySpec(newobj *CStorRestore) {
	r.object.Spec = newobj.object.Spec
}

// IsPendingStatus is to check if the restore is in a pending state.
func (r *CStorRestore) IsPendingStatus() bool {
	return string(r.object.Status) == string(apis.RSTCStorStatusPending)
}

// IsInProgressStatus is to check if the restore is in in-progress state.
func (r *CStorRestore) IsInProgressStatus() bool {
	return string(r.object.Status) == string(apis.RSTCStorStatusInProgress)
}

// IsInitStatus is to check if the restore is in init state.
func (r *CStorRestore) IsInitStatus() bool {
	return string(r.object.Status) == string(apis.RSTCStorStatusInit)
}

// IsDoneStatus is to check if the restore is completed or not
func (r *CStorRestore) IsDoneStatus() bool {
	return string(r.object.Status) == string(apis.RSTCStorStatusDone)
}

// IsFailedStatus is to check if the restore is failed or not
func (r *CStorRestore) IsFailedStatus() bool {
	return string(r.object.Status) == string(apis.RSTCStorStatusFailed)
}

// IsDestroyEvent is to check if the call is for restore destroy.
func (r *CStorRestore) IsDestroyEvent() bool {
	return r.object.ObjectMeta.DeletionTimestamp != nil
}

// IsOnlyStatusChange is to check the only status change of CStorRestore object.
func (r *CStorRestore) IsOnlyStatusChange(newbkp *CStorRestore) bool {
	return reflect.DeepEqual(r.object.Spec, newbkp.object.Spec) &&
		!reflect.DeepEqual(r.object.Status, newbkp.object.Status)
}

// GetCR return CStorRestore object using given name
func (r *CStorRestore) GetCR(name string) (*CStorRestore, error) {
	ra, err := r.Get(name, r.object.Namespace)
	if err == nil {
		return NewCStorRestoreBuilder().
			WithClientSet(nil).
			BuildFromAPIObject(ra)
	}
	return nil, err
}

// CreateCR create CStorRestore CR from given object
func (r *CStorRestore) CreateCR(obj *CStorRestore) (*CStorRestore, error) {
	ra, err := r.Create(obj.GetRestoreAPIObject())
	if err == nil {
		return NewCStorRestoreBuilder().
			WithClientSet(nil).
			BuildFromAPIObject(ra)
	}
	return nil, err
}

// UpdateCR updates the given CStorRestore object
func (r *CStorRestore) UpdateCR(obj *CStorRestore) (*CStorRestore, error) {
	ra, err := r.Update(obj.GetRestoreAPIObject())
	if err == nil {
		return NewCStorRestoreBuilder().
			WithClientSet(nil).
			BuildFromAPIObject(ra)
	}
	return nil, err
}

// UpdateCRStatus updates the status of `name` restore with given status
func (r *CStorRestore) UpdateCRStatus(name string, status apis.CStorRestoreStatus) error {
	ra, err := r.Get(name, r.object.Namespace)
	if err == nil {
		ra.Status = status
		_, err := r.Update(ra)
		return err
	}
	return err
}
