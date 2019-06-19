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
	Operation = "add"
)

//PoolExpansion defines structure for pool 'Expansion' operation
type PoolExpansion struct {
	// list of vdev to add
	VdevList []string

	// property list
	Property []string

	// name of pool
	Pool string

	// command string
	Command string

	// predicatelist is list of predicate function used for validating object
	predicatelist []PredicateFunc

	// error
	err error
}

// NewPoolExpansion returns new instance of object PoolExpansion
func NewPoolExpansion() *PoolExpansion {
	return &PoolExpansion{}
}

// WithCheck add given predicate to predicate list
func (p *PoolExpansion) WithCheck(pred ...PredicateFunc) *PoolExpansion {
	p.predicatelist = append(p.predicatelist, pred...)
	return p
}

// WithVdevList method fills the VdevList field of PoolExpansion object.
func (p *PoolExpansion) WithVdevList(vdev string) *PoolExpansion {
	p.VdevList = append(p.VdevList, vdev)
	return p
}

// WithProperty method fills the Property field of PoolExpansion object.
func (p *PoolExpansion) WithProperty(key, value string) *PoolExpansion {
	p.Property = append(p.Property, fmt.Sprintf("%s=%s", key, value))
	return p
}

// WithPool method fills the Pool field of PoolExpansion object.
func (p *PoolExpansion) WithPool(Pool string) *PoolExpansion {
	p.Pool = Pool
	return p
}

// WithCommand method fills the Command field of PoolExpansion object.
func (p *PoolExpansion) WithCommand(Command string) *PoolExpansion {
	p.Command = Command
	return p
}

// Validate is to validate generated PoolExpansion object by builder
func (p *PoolExpansion) Validate() *PoolExpansion {
	if len(p.predicatelist) != 0 {
		for _, pred := range p.predicatelist {
			if !pred(p) {
				p.err = errors.Wrapf(p.err, "Failed to run predicate {%v}", pred)
			}
		}
	}
	return p
}

// Execute is to execute generated PoolExpansion object
func (p *PoolExpansion) Execute() ([]byte, error) {
	p, err := p.Build()
	if err != nil {
		return nil, err
	}
	// execute command here
	return exec.Command(zfs.ZPOOL, p.Command).CombinedOutput()
}

// Build returns the PoolExpansion object generated by builder
func (p *PoolExpansion) Build() (*PoolExpansion, error) {
	var c strings.Builder
	p = p.Validate()
	p.appendCommand(c, fmt.Sprintf(" %s ", Operation))

	if IsPropertySet()(p) {
		for _, v := range p.Property {
			p.appendCommand(c, fmt.Sprintf(" -o %s ", v))
		}
	}

	p.appendCommand(c, p.Pool)

	for _, v := range p.VdevList {
		p.appendCommand(c, fmt.Sprintf(" %s ", v))
	}

	p.Command = c.String()
	return p, p.err
}

// appendCommand append string to given string builder
func (p *PoolExpansion) appendCommand(c strings.Builder, cmd string) {
	_, err := c.WriteString(cmd)
	if err != nil {
		p.err = errors.Wrapf(p.err, "Failed to append cmd{%s} : %s", cmd, err.Error())
	}
}
