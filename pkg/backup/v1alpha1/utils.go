package backup

import (
	"github.com/golang/glog"
	apis "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// isNotFound returns true if the original
// cause of error was due to castemplate's
// not found error or kubernetes not found
// error
func isNotFound(err error) bool {
	err = errors.Cause(err)
	return k8serrors.IsNotFound(err)
}

// GetLastTransferredSnapName returns last sucessfull backup's snapshot name
func (b *CStorBackup) GetLastTransferredSnapName() (string, error) {
	if b.client == nil {
		glog.Infof("b client nil ??")
	}
	cb, err := b.client.BackupV1alpha1().
		CStorBackupCompleteds(b.GetNamespace()).
		Get(b.GetCompletedBackupName(), v1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		//TODO wrap error
		return "", err
	}
	return cb.Spec.SnapName, nil
}

// UpdateCompletedBackup updates the last completed backup object for current backup
func (b *CStorBackup) UpdateCompletedBackup() error {
	cb, err := b.client.BackupV1alpha1().
		CStorBackupCompleteds(b.GetNamespace()).
		Get(b.GetCompletedBackupName(), v1.GetOptions{})
	if err != nil {
		//TODO wrap error check not found error
		bk := &apis.CStorBackupCompleted{
			ObjectMeta: v1.ObjectMeta{
				Name:      b.GetCompletedBackupName(),
				Namespace: b.GetNamespace(),
				Labels:    b.GetLabels(),
			},
			Spec: apis.CStorBackupSpec{
				BackupName: b.GetBackupName(),
				VolumeName: b.GetVolumeName(),
				SnapName:   b.GetSnapName(),
			},
		}

		_, err := b.client.BackupV1alpha1().
			CStorBackupCompleteds(b.GetNamespace()).
			Create(bk)
		if err != nil {
			//TODO wrap error
			glog.Errorf("Error creating last-backup resource for backup:%v err:%v", bk.Spec.BackupName, err)
			return err
		}
		//TODO wrap error
		glog.Infof("LastBackup resource created for backup:%s volume:%s", bk.Spec.BackupName, bk.Spec.VolumeName)
		return nil
	}
	cb.Spec.PrevSnapName = cb.Spec.SnapName
	cb.Spec.SnapName = b.GetSnapName()
	_, err = b.client.BackupV1alpha1().
		CStorBackupCompleteds(b.GetNamespace()).
		Update(cb)
	return err
}
