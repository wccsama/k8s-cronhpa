// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	cronhpav1 "github.com/k8s-cronhpa/pkg/api/cronhpa/v1"
	versioned "github.com/k8s-cronhpa/pkg/client/clientset/versioned"
	internalinterfaces "github.com/k8s-cronhpa/pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/k8s-cronhpa/pkg/client/listers/cronhpa/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// CronHPAInformer provides access to a shared informer and lister for
// CronHPAs.
type CronHPAInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.CronHPALister
}

type cronHPAInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewCronHPAInformer constructs a new informer for CronHPA type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCronHPAInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCronHPAInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredCronHPAInformer constructs a new informer for CronHPA type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCronHPAInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.WccV1().CronHPAs(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.WccV1().CronHPAs(namespace).Watch(context.TODO(), options)
			},
		},
		&cronhpav1.CronHPA{},
		resyncPeriod,
		indexers,
	)
}

func (f *cronHPAInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCronHPAInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *cronHPAInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&cronhpav1.CronHPA{}, f.defaultInformer)
}

func (f *cronHPAInformer) Lister() v1.CronHPALister {
	return v1.NewCronHPALister(f.Informer().GetIndexer())
}
