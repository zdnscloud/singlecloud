package cache

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/zdnscloud/gok8s/client"
)

// IndexerFunc knows how to take an object and turn it into a series
// of (non-namespaced) keys for that object.
type IndexerFunc func(runtime.Object) []string

// FieldIndexer knows how to index over a particular "field" such that it
// can later be used by a field selector.
type FieldIndexer interface {
	IndexField(obj runtime.Object, field string, extractValue IndexerFunc) error
}

type Informers interface {
	GetInformer(obj runtime.Object) (toolscache.SharedIndexInformer, error)
	GetInformerForKind(gvk schema.GroupVersionKind) (toolscache.SharedIndexInformer, error)
	Start(stopCh <-chan struct{}) error
	WaitForCacheSync(stop <-chan struct{}) bool
	IndexField(obj runtime.Object, field string, extractValue IndexerFunc) error
}

type Cache interface {
	client.Reader
	Informers
}
