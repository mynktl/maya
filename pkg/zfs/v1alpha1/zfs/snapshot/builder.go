/*
Copyright 2019 The OpenEBS Authors.

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

package zfs

import (
	"fmt"
	"os/exec"
	"strings"

	zfs "github.com/openebs/maya/pkg/zfs/v1alpha1"
	"github.com/pkg/errors"
)

// SnapOp defines snapshot operation type, this can be either snapshot, send or receive
type SnapOp string

const (
	// SnapOperation defines zfs snapshot operation
	SnapOperation = "snapshot"

	// SendOperation defines zfs send operation
	SendOperation = "send"

	// RecvOperation defines zfs receive operation
	RecvOperation = "receive"
)

//VolumeSnapshot defines structure for volume 'Snapshot' operation
type VolumeSnapshot struct {
	//list of property
	Property []string

	//name of snapshot
	Snapshot string

	//name of dataset on which snapshot should be taken
	Dataset string

	//operation type
	OpType SnapOp

	//remote destination for snapshot send/recv using nc
	Target string

	// to send incremental snapshot
	LastSnapshot string

	//Recursively create snapshots of all descendent datasets
	Recursive bool

	// Generate a deduplicated stream
	Dedup bool

	// dry-run
	DryRun bool

	// use compression for zfs send
	EnableCompression bool

	// command string
	Command string

	// predicatelist is list of predicate function used for validating object
	predicatelist []PredicateFunc

	// error
	err error
}

// NewVolumeSnapshot returns new instance of object VolumeSnapshot
func NewVolumeSnapshot() *VolumeSnapshot {
	return &VolumeSnapshot{}
}

// WithCheck add given predicate to predicate list
func (v *VolumeSnapshot) WithCheck(pred ...PredicateFunc) *VolumeSnapshot {
	v.predicatelist = append(v.predicatelist, pred...)
	return v
}

// WithProperty method fills the Property field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithProperty(key, value string) *VolumeSnapshot {
	v.Property = append(v.Property, fmt.Sprintf("%s=%s", key, value))
	return v
}

// WithRecursive method fills the Recursive field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithRecursive(Recursive bool) *VolumeSnapshot {
	v.Recursive = Recursive
	return v
}

// WithSnapshot method fills the Snapshot field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithSnapshot(Snapshot string) *VolumeSnapshot {
	v.Snapshot = Snapshot
	return v
}

// WithDataset method fills the Dataset field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithDataset(Dataset string) *VolumeSnapshot {
	v.Dataset = Dataset
	return v
}

// WithOpType method fills the OpType field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithOpType(OpType SnapOp) *VolumeSnapshot {
	v.OpType = OpType
	return v
}

// WithTarget method fills the Target field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithTarget(Target string) *VolumeSnapshot {
	v.Target = Target
	return v
}

// WithDedup method fills the Dedup field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithDedup(Dedup bool) *VolumeSnapshot {
	v.Dedup = Dedup
	return v
}

// WithLastSnapshot method fills the LastSnapshot field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithLastSnapshot(LastSnapshot string) *VolumeSnapshot {
	v.LastSnapshot = LastSnapshot
	return v
}

// WithDryRun method fills the DryRun field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithDryRun(DryRun bool) *VolumeSnapshot {
	v.DryRun = DryRun
	return v
}

// WithEnableCompression method fills the EnableCompression field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithEnableCompression(EnableCompression bool) *VolumeSnapshot {
	v.EnableCompression = EnableCompression
	return v
}

// WithCommand method fills the Command field of VolumeSnapshot object.
func (v *VolumeSnapshot) WithCommand(Command string) *VolumeSnapshot {
	v.Command = Command
	return v
}

// Validate is to validate generated VolumeSnapshot object by builder
func (v *VolumeSnapshot) Validate() *VolumeSnapshot {
	if len(v.predicatelist) != 0 {
		for _, pred := range v.predicatelist {
			if !pred(v) {
				v.err = errors.Wrapf(v.err, "Failed to run predicate {%v}", pred)
			}
		}
	}
	return v
}

// Execute is to execute generated VolumeSnapshot object
func (v *VolumeSnapshot) Execute() ([]byte, error) {
	v, err := v.Build()
	if err != nil {
		return nil, err
	}
	// execute command here
	return exec.Command(zfs.ZFS, v.Command).CombinedOutput()
}

// Build returns the VolumeSnapshot object generated by builder
func (v *VolumeSnapshot) Build() (*VolumeSnapshot, error) {
	var c strings.Builder
	v = v.Validate()

	switch v.GetOpType() {
	case SendOperation:
		v.appendCommand(c, fmt.Sprintf(" %s ", SendOperation))
		if IsDedupSet()(v) {
			v.appendCommand(c, fmt.Sprintf(" -D "))
		}

		if IsLastSnapshotSet()(v) {
			v.appendCommand(c, fmt.Sprintf(" -i @%s ", v.LastSnapshot))
		}

		v.appendCommand(c, fmt.Sprintf(" %s@%s ", v.Dataset, v.Snapshot))
		v.appendCommand(c, fmt.Sprintf(" | nc %s", v.Target))

	case RecvOperation:
		v.appendCommand(c, fmt.Sprintf(" %s ", RecvOperation))
		v.appendCommand(c, fmt.Sprintf(" %s@%s ", v.Dataset, v.Snapshot))
		v.appendCommand(c, fmt.Sprintf(" | nc %s", v.Target))

	case SnapOperation:
		v.appendCommand(c, fmt.Sprintf(" %s ", SnapOperation))
		if IsRecursiveSet()(v) {
			v.appendCommand(c, fmt.Sprintf(" -r "))
		}

		if IsPropertySet()(v) {
			for _, p := range v.Property {
				v.appendCommand(c, fmt.Sprintf(" -o %s", p))
			}
		}

		v.appendCommand(c, fmt.Sprintf(" %s@%s ", v.Dataset, v.Snapshot))
	}

	v.Command = c.String()
	return v, v.err
}

// appendCommand append string to given string builder
func (v *VolumeSnapshot) appendCommand(c strings.Builder, cmd string) {
	_, err := c.WriteString(cmd)
	if err != nil {
		v.err = errors.Wrapf(v.err, "Failed to append cmd{%s} : %s", cmd, err.Error())
	}
}
