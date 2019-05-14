package restore

import (
	apis "github.com/openebs/maya/pkg/apis/openebs.io/restore/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CStorRestoreList is a list of CStorRestore resources
type CStorRestoreList struct {
	// KubeClient to perform operation on CR
	*KubeClient

	// List of CStorRestore object
	Item []*CStorRestore
}

// CStorRestoreListBuilder defines builder for CStorRestoreList
type CStorRestoreListBuilder struct {
	// CStorRestore object list
	olist *CStorRestoreList

	// CStorRestore API object list
	alist *apis.CStorRestoreList

	// List of filters or checks to perform before building a list
	filters []PredicateFunc

	// namespace for which list should be built
	namespace string

	// clientfn is custom function to fetch KubeClient needed for CR operations
	clientfn LoadRestoreClient

	// err stores error generated during build operation
	err error

	// list options
	opts v1.ListOptions
}

// NewCStorRestoreListBuilder returns new builder
func NewCStorRestoreListBuilder() *CStorRestoreListBuilder {
	return &CStorRestoreListBuilder{
		olist: &CStorRestoreList{
			KubeClient: &KubeClient{},
		},
	}
}

// WithNamespace set namespace for builder
func (crl *CStorRestoreListBuilder) WithNamespace(ns string) *CStorRestoreListBuilder {
	crl.namespace = ns
	return crl
}

// WithCheckList updates the filter for current builder
func (crl *CStorRestoreListBuilder) WithCheckList(pred []PredicateFunc) *CStorRestoreListBuilder {
	crl.filters = append(crl.filters, pred...)
	return crl
}

// WithClientSet loads the kubeclient either by given function or default function
func (crl *CStorRestoreListBuilder) WithClientSet(fn LoadRestoreClient) *CStorRestoreListBuilder {
	var err error

	if fn != nil {
		crl.olist.client, err = fn()
		crl.clientfn = fn
	} else {
		crl.olist.client, err = LoadClientSet()
	}

	if err != nil {
		errors.Wrapf(crl.err, "Failed to load clientset")
	}
	return crl
}

// WithAPIObjList sets CStorRestore API objset list for current builder
func (crl *CStorRestoreListBuilder) WithAPIObjList(al *apis.CStorRestoreList) *CStorRestoreListBuilder {
	if al != nil {
		crl.alist = al
	}
	return crl
}

// Build creates the list of CStorRestore object
func (crl *CStorRestoreListBuilder) Build() (*CStorRestoreList, error) {
	if crl.alist == nil {
		crl.alist, crl.err = crl.olist.List(crl.namespace, v1.ListOptions{})
	}

	for _, aobj := range crl.alist.Items {
		bobj, err := NewCStorRestoreBuilder().
			WithCheckList(crl.filters).
			WithClientSet(crl.clientfn).
			BuildFromAPIObject(&aobj)

		if err != nil {
			errors.Wrapf(crl.err, "Failed to build object for {%s}", aobj.Name)
			continue
		}
		crl.olist.Item = append(crl.olist.Item, bobj)
	}

	return crl.olist, crl.err
}
