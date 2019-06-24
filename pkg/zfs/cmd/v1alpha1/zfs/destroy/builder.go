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

package vdestroy

import (
	"fmt"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	"github.com/openebs/maya/pkg/zfs/cmd/v1alpha1/bin"
	"github.com/pkg/errors"
)

const (
	// Operation defines type of zfs operation
	Operation = "destroy"
)

//VolumeDestroy defines structure for volume 'Destroy' operation
type VolumeDestroy struct {
	//Do a dry-run
	DryRun bool

	//recursively destroy all  the clones, snapshots, and children
	Recursive bool

	//name of the dataset or snapshot
	Name string

	// command for this structure
	Command string

	// checks is list of predicate function used for validating object
	checks []PredicateFunc

	// error
	err error
}

// NewVolumeDestroy returns new instance of object VolumeDestroy
func NewVolumeDestroy() *VolumeDestroy {
	return &VolumeDestroy{}
}

// WithCheck add given check to checks list
func (v *VolumeDestroy) WithCheck(check ...PredicateFunc) *VolumeDestroy {
	v.checks = append(v.checks, check...)
	return v
}

// WithDryRun method fills the DryRun field of VolumeDestroy object.
func (v *VolumeDestroy) WithDryRun(DryRun bool) *VolumeDestroy {
	v.DryRun = DryRun
	return v
}

// WithRecursive method fills the Recursive field of VolumeDestroy object.
func (v *VolumeDestroy) WithRecursive(Recursive bool) *VolumeDestroy {
	v.Recursive = Recursive
	return v
}

// WithName method fills the Name field of VolumeDestroy object.
func (v *VolumeDestroy) WithName(Name string) *VolumeDestroy {
	v.Name = Name
	return v
}

// WithCommand method fills the Command field of VolumeDestroy object.
func (v *VolumeDestroy) WithCommand(Command string) *VolumeDestroy {
	v.Command = Command
	return v
}

// Validate is to validate generated VolumeDestroy object by builder
func (v *VolumeDestroy) Validate() *VolumeDestroy {
	for _, check := range v.checks {
		if !check(v) {
			v.err = errors.Wrapf(v.err, "validation failed {%v}", runtime.FuncForPC(reflect.ValueOf(check).Pointer()).Name())
		}
	}
	return v
}

// Execute is to execute generated VolumeDestroy object
func (v *VolumeDestroy) Execute() ([]byte, error) {
	v, err := v.Build()
	if err != nil {
		return nil, err
	}
	// execute command here
	return exec.Command(bin.ZFS, v.Command).CombinedOutput()
}

// Build returns the VolumeDestroy object generated by builder
func (v *VolumeDestroy) Build() (*VolumeDestroy, error) {
	var c strings.Builder

	v = v.Validate()
	v.appendCommand(c, fmt.Sprintf(" %s ", Operation))

	if IsDryRunSet()(v) {
		v.appendCommand(c, fmt.Sprintf(" -n"))
	}

	if IsRecursiveSet()(v) {
		v.appendCommand(c, fmt.Sprintf(" -R "))
	}

	v.appendCommand(c, v.Name)
	v.Command = c.String()

	return v, v.err
}

// appendCommand append string to given string builder
func (v *VolumeDestroy) appendCommand(c strings.Builder, cmd string) {
	_, err := c.WriteString(cmd)
	if err != nil {
		v.err = errors.Wrapf(v.err, "Failed to append cmd{%s} : %s", cmd, err.Error())
	}
}
