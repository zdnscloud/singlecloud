package client

import (
	"reflect"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/zdnscloud/gok8s/client/apiutil"
)

type clientCache struct {
	config *rest.Config
	scheme *runtime.Scheme
	mapper meta.RESTMapper
	codecs serializer.CodecFactory
	// resourceByType caches type metadata
	resourceByType map[reflect.Type]*resourceMeta
	mu             sync.RWMutex
}

func (c *clientCache) newResource(obj runtime.Object) (*resourceMeta, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(gvk.Kind, "List") && meta.IsListType(obj) {
		gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
	}

	client, err := apiutil.RESTClientForGVK(gvk, c.config, c.codecs)
	if err != nil {
		return nil, err
	}
	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	return &resourceMeta{Interface: client, mapping: mapping, gvk: gvk}, nil
}

func (c *clientCache) getResource(obj runtime.Object) (*resourceMeta, error) {
	typ := reflect.TypeOf(obj)
	c.mu.RLock()
	r, known := c.resourceByType[typ]
	c.mu.RUnlock()
	if known {
		return r, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	r, err := c.newResource(obj)
	if err != nil {
		return nil, err
	}
	c.resourceByType[typ] = r
	return r, err
}

func (c *clientCache) getObjMeta(obj runtime.Object) (*objMeta, error) {
	r, err := c.getResource(obj)
	if err != nil {
		return nil, err
	}
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	return &objMeta{resourceMeta: r, Object: m}, err
}

// resourceMeta caches state for a Kubernetes type.
type resourceMeta struct {
	rest.Interface
	gvk     schema.GroupVersionKind
	mapping *meta.RESTMapping
}

func (r *resourceMeta) isNamespaced() bool {
	return r.mapping.Scope.Name() != meta.RESTScopeNameRoot
}

func (r *resourceMeta) resource() string {
	return r.mapping.Resource.Resource
}

type objMeta struct {
	*resourceMeta
	v1.Object
}
