package restore

// PredicateFunc defines data-type for validation function
type PredicateFunc func(*CStorRestore) bool

// IsRestoreNameSet checks if backup-name is set or not
func IsRestoreNameSet() PredicateFunc {
	return func(c *CStorRestore) bool {
		return len(c.GetRestoreName()) != 0
	}
}

// IsVolumeNameSet checks if volume-name is set or not
func IsVolumeNameSet() PredicateFunc {
	return func(c *CStorRestore) bool {
		return len(c.GetVolumeName()) != 0
	}
}

// IsNamespaceSet checks if namespace is set or not
func IsNamespaceSet() PredicateFunc {
	return func(c *CStorRestore) bool {
		return len(c.GetNamespace()) != 0
	}
}

// IsRestoreSrcSet checks if restore source is set or not
func IsRestoreSrcSet() PredicateFunc {
	return func(c *CStorRestore) bool {
		return len(c.GetRestoreSrc()) != 0
	}
}

// IsPendingStatus is to check if the restore is in a pending state.
func IsPendingStatus() PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsPendingStatus()
	}
}

// IsInProgressStatus is to check if the restore is in in-progress state.
func IsInProgressStatus() PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsInProgressStatus()
	}
}

// IsInitStatus is to check if the restore is in init state.
func IsInitStatus() PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsInitStatus()
	}
}

// IsDoneStatus is to check if the restore is completed or not
func IsDoneStatus() PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsDoneStatus()
	}
}

// IsFailedStatus is to check if the restore is failed or not
func IsFailedStatus() PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsFailedStatus()
	}
}

// IsDestroyEvent is to check if the call is for restore destroy.
func IsDestroyEvent() PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsDestroyEvent()
	}
}

// IsOnlyStatusChange is to check the only status change of CStorRestore object.
func IsOnlyStatusChange(newbkp *CStorRestore) PredicateFunc {
	return func(c *CStorRestore) bool {
		return c.IsOnlyStatusChange(newbkp)
	}
}
