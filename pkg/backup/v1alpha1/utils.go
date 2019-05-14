package backup

import (
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

// GetLastTransferredSnapName returns last successfull backup's snapshot name
func (b *CStorBackup) GetLastTransferredSnapName() (string, error) {
	if b.client == nil {
		return "", errors.New("Missing clientset")
	}

	cb, err := b.client.OpenebsV1alpha1().
		CStorCompletedBackups(b.GetNamespace()).
		Get(b.GetCompletedBackupName(), v1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", errors.Wrapf(err, "Failed to fetch completedbackup{%s}", b.GetCompletedBackupName())
	}
	return cb.Spec.SnapName, nil
}

// UpdateCompletedBackup updates the last completed backup object for current backup
func (b *CStorBackup) UpdateCompletedBackup() error {
	cb, err := b.client.OpenebsV1alpha1().
		CStorCompletedBackups(b.GetNamespace()).
		Get(b.GetCompletedBackupName(), v1.GetOptions{})
	if err != nil {
		bk := &apis.CStorCompletedBackup{
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

		_, err := b.client.OpenebsV1alpha1().
			CStorCompletedBackups(b.GetNamespace()).
			Create(bk)
		return err
	}
	cb.Spec.PrevSnapName = cb.Spec.SnapName
	cb.Spec.SnapName = b.GetSnapName()
	_, err = b.client.OpenebsV1alpha1().
		CStorCompletedBackups(b.GetNamespace()).
		Update(cb)
	return err
}
