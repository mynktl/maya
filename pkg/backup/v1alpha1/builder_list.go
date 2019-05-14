package backup

import (
	apis "github.com/openebs/maya/pkg/apis/openebs.io/backup/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CStorBackupList is a list of CStorBackup resources
type CStorBackupList struct {
	// KubeClient to perform operation on CR
	*KubeClient

	// List of CStorBackup object
	Item []*CStorBackup
}

// CStorBackupListBuilder defines builder for CStorBackupList
type CStorBackupListBuilder struct {
	// CStorBackup object list
	olist *CStorBackupList

	// CStorBackup API object list
	alist *apis.CStorBackupList

	// List of filters or checks to perform before building a list
	filters []PredicateFunc

	// namespace for which list should be built
	namespace string

	// clientfn is custom function to fetch KubeClient needed for CR operations
	clientfn LoadBackupClient

	// err stores error generated during build operation
	err error

	// list options
	opts v1.ListOptions
}

// NewCStorBackupListBuilder returns new builder
func NewCStorBackupListBuilder() *CStorBackupListBuilder {
	return &CStorBackupListBuilder{
		olist: &CStorBackupList{
			KubeClient: &KubeClient{},
		},
	}
}

// WithNamespace set namespace for builder
func (cbl *CStorBackupListBuilder) WithNamespace(ns string) *CStorBackupListBuilder {
	cbl.namespace = ns
	return cbl
}

// WithCheckList updates the filter for current builder
func (cbl *CStorBackupListBuilder) WithCheckList(pred []PredicateFunc) *CStorBackupListBuilder {
	cbl.filters = append(cbl.filters, pred...)
	return cbl
}

// WithClientSet loads the kubeclient either by given function or default function
func (cbl *CStorBackupListBuilder) WithClientSet(fn LoadBackupClient) *CStorBackupListBuilder {
	var err error

	if fn != nil {
		cbl.olist.client, err = fn()
		cbl.clientfn = fn
	} else {
		cbl.olist.client, err = LoadClientSet()
	}

	if err != nil {
		errors.Wrapf(cbl.err, "Failed to load clientset")
	}
	return cbl
}

// WithAPIObjList sets CStorBackup API objset list for current builder
func (cbl *CStorBackupListBuilder) WithAPIObjList(al *apis.CStorBackupList) *CStorBackupListBuilder {
	if al != nil {
		cbl.alist = al
	}
	return cbl
}

// Build creates the list of CStorBackup object
func (cbl *CStorBackupListBuilder) Build() (*CStorBackupList, error) {
	if cbl.alist == nil {
		cbl.alist, cbl.err = cbl.olist.List(cbl.namespace, v1.ListOptions{})
	}

	for _, aobj := range cbl.alist.Items {
		bobj, err := NewCStorBackupBuilder().
			WithCheckList(cbl.filters).
			WithClientSet(cbl.clientfn).
			BuildFromAPIObject(&aobj)

		if err != nil {
			errors.Wrapf(cbl.err, "Failed to build object for {%s}", aobj.Name)
			continue
		}
		cbl.olist.Item = append(cbl.olist.Item, bobj)
	}

	return cbl.olist, cbl.err
}
