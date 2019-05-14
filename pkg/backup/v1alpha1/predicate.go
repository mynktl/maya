package backup

// PredicateFunc defines data-type for validation function
type PredicateFunc func(*CStorBackup) bool

// IsBackupNameSet checks if backup-name is set or not
func IsBackupNameSet() PredicateFunc {
	return func(c *CStorBackup) bool {
		return len(c.GetBackupName()) != 0
	}
}

// IsVolumeNameSet checks if volume-name is set or not
func IsVolumeNameSet() PredicateFunc {
	return func(c *CStorBackup) bool {
		return len(c.GetVolumeName()) != 0
	}
}

// IsSnapNameSet checks if snapshot name is set or not
func IsSnapNameSet() PredicateFunc {
	return func(c *CStorBackup) bool {
		return len(c.GetSnapName()) != 0
	}
}

// IsPrevSnapNameSet checks if previous snapshot name is set or not
func IsPrevSnapNameSet() PredicateFunc {
	return func(c *CStorBackup) bool {
		return len(c.GetPrevSnapName()) != 0
	}
}

// IsNamespaceSet checks if namespace is set or not
func IsNamespaceSet() PredicateFunc {
	return func(c *CStorBackup) bool {
		return len(c.GetNamespace()) != 0
	}
}

// IsBackupDestSet checks if backup destination is set or not
func IsBackupDestSet() PredicateFunc {
	return func(c *CStorBackup) bool {
		return len(c.GetBackupDest()) != 0
	}
}

// IsPendingStatus is to check if the backup is in a pending state.
func IsPendingStatus() PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsPendingStatus()
	}
}

// IsInProgressStatus is to check if the backup is in in-progress state.
func IsInProgressStatus() PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsInProgressStatus()
	}
}

// IsInitStatus is to check if the backup is in init state.
func IsInitStatus() PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsInitStatus()
	}
}

// IsDoneStatus is to check if the backup is completed or not
func IsDoneStatus() PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsDoneStatus()
	}
}

// IsFailedStatus is to check if the backup is failed or not
func IsFailedStatus() PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsFailedStatus()
	}
}

// IsDestroyEvent is to check if the call is for backup destroy.
func IsDestroyEvent() PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsDestroyEvent()
	}
}

// IsOnlyStatusChange is to check the only status change of CStorBackup object.
func IsOnlyStatusChange(newbkp *CStorBackup) PredicateFunc {
	return func(c *CStorBackup) bool {
		return c.IsOnlyStatusChange(newbkp)
	}
}
