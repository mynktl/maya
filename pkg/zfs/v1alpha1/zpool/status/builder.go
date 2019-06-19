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

const (
	// Operation defines type of zfs operation
	Operation = "status"
)

//PoolStatus defines structure for pool 'Status' operation
type PoolStatus struct {
	//pool name
	Pool string

	// command string
	Command string

	// predicatelist is list of predicate function used for validating object
	predicatelist []PredicateFunc

	// error
	err error
}

// NewPoolStatus returns new instance of object PoolStatus
func NewPoolStatus() *PoolStatus {
	return &PoolStatus{}
}

// WithCheck add given predicate to predicate list
func (p *PoolStatus) WithCheck(pred ...PredicateFunc) *PoolStatus {
	p.predicatelist = append(p.predicatelist, pred...)
	return p
}

// WithPool method fills the Pool field of PoolStatus object.
func (p *PoolStatus) WithPool(Pool string) *PoolStatus {
	p.Pool = Pool
	return p
}

// WithCommand method fills the Command field of PoolStatus object.
func (p *PoolStatus) WithCommand(Command string) *PoolStatus {
	p.Command = Command
	return p
}

// Validate is to validate generated PoolStatus object by builder
func (p *PoolStatus) Validate() *PoolStatus {
	if len(p.predicatelist) != 0 {
		for _, pred := range p.predicatelist {
			if !pred(p) {
				p.err = errors.Wrapf(p.err, "Failed to run predicate {%v}", pred)
			}
		}
	}
	return p
}

// Execute is to execute generated PoolStatus object
func (p *PoolStatus) Execute() ([]byte, error) {
	p, err := p.Build()
	if err != nil {
		return nil, err
	}
	// execute command here
	return exec.Command(zfs.ZPOOL, p.Command).CombinedOutput()
}

// Build returns the PoolStatus object generated by builder
func (p *PoolStatus) Build() (*PoolStatus, error) {
	var c strings.Builder
	p = p.Validate()
	p.appendCommand(c, fmt.Sprintf(" %s ", Operation))

	p.appendCommand(c, p.Pool)

	p.Command = c.String()
	return p, p.err
}

// appendCommand append string to given string builder
func (p *PoolStatus) appendCommand(c strings.Builder, cmd string) {
	_, err := c.WriteString(cmd)
	if err != nil {
		p.err = errors.Wrapf(p.err, "Failed to append cmd{%s} : %s", cmd, err.Error())
	}
}
