// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CronHPALister helps list CronHPAs.
// All objects returned here must be treated as read-only.
type CronHPALister interface {
	// List lists all CronHPAs in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.CronHPA, err error)
	// CronHPAs returns an object that can list and get CronHPAs.
	CronHPAs(namespace string) CronHPANamespaceLister
	CronHPAListerExpansion
}

// cronHPALister implements the CronHPALister interface.
type cronHPALister struct {
	indexer cache.Indexer
}

// NewCronHPALister returns a new CronHPALister.
func NewCronHPALister(indexer cache.Indexer) CronHPALister {
	return &cronHPALister{indexer: indexer}
}

// List lists all CronHPAs in the indexer.
func (s *cronHPALister) List(selector labels.Selector) (ret []*v1.CronHPA, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.CronHPA))
	})
	return ret, err
}

// CronHPAs returns an object that can list and get CronHPAs.
func (s *cronHPALister) CronHPAs(namespace string) CronHPANamespaceLister {
	return cronHPANamespaceLister{indexer: s.indexer, namespace: namespace}
}

// CronHPANamespaceLister helps list and get CronHPAs.
// All objects returned here must be treated as read-only.
type CronHPANamespaceLister interface {
	// List lists all CronHPAs in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.CronHPA, err error)
	// Get retrieves the CronHPA from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.CronHPA, error)
	CronHPANamespaceListerExpansion
}

// cronHPANamespaceLister implements the CronHPANamespaceLister
// interface.
type cronHPANamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all CronHPAs in the indexer for a given namespace.
func (s cronHPANamespaceLister) List(selector labels.Selector) (ret []*v1.CronHPA, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.CronHPA))
	})
	return ret, err
}

// Get retrieves the CronHPA from the indexer for a given namespace and name.
func (s cronHPANamespaceLister) Get(name string) (*v1.CronHPA, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("cronhpa"), name)
	}
	return obj.(*v1.CronHPA), nil
}
