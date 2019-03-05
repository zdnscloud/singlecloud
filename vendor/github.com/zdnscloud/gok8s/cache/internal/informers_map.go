package internal

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/zdnscloud/gok8s/client/apiutil"
)

func NewInformersMap(config *rest.Config,
	scheme *runtime.Scheme,
	mapper meta.RESTMapper,
	resync time.Duration,
	namespace string) *InformersMap {
	m := &InformersMap{
		config:         config,
		Scheme:         scheme,
		mapper:         mapper,
		informersByGVK: make(map[schema.GroupVersionKind]*ResourceInformer),
		codecs:         serializer.NewCodecFactory(scheme),
		paramCodec:     runtime.NewParameterCodec(scheme),
		resync:         resync,
		namespace:      namespace,
	}
	return m
}

type InformersMap struct {
	Scheme         *runtime.Scheme
	config         *rest.Config
	mapper         meta.RESTMapper
	informersByGVK map[schema.GroupVersionKind]*ResourceInformer
	codecs         serializer.CodecFactory
	paramCodec     runtime.ParameterCodec
	stop           <-chan struct{}
	resync         time.Duration
	mu             sync.RWMutex
	started        bool
	namespace      string
}

func (m *InformersMap) Start(stop <-chan struct{}) error {
	go func() {
		m.mu.Lock()
		m.stop = stop
		for _, informer := range m.informersByGVK {
			go informer.Run(stop)
		}
		m.started = true
		m.mu.Unlock()
	}()
	<-stop
	return nil
}

func (m *InformersMap) WaitForCacheSync(stop <-chan struct{}) bool {
	syncedFuncs := append([]cache.InformerSynced(nil), m.hasSyncedFuncs()...)
	return cache.WaitForCacheSync(stop, syncedFuncs...)
}

func (m *InformersMap) hasSyncedFuncs() []cache.InformerSynced {
	m.mu.RLock()
	defer m.mu.RUnlock()
	syncedFuncs := make([]cache.InformerSynced, 0, len(m.informersByGVK))
	for _, informer := range m.informersByGVK {
		syncedFuncs = append(syncedFuncs, informer.HasSynced)
	}
	return syncedFuncs
}

func (m *InformersMap) GetInformer(gvk schema.GroupVersionKind) (*ResourceInformer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.informersByGVK[gvk]; ok {
		return c, nil
	} else {
		return m.createResourceCache(gvk)
	}
}

func (m *InformersMap) createResourceCache(gvk schema.GroupVersionKind) (*ResourceInformer, error) {
	lw, err := m.createListWatcher(gvk)
	if err != nil {
		return nil, err
	}
	obj, err := m.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	c := newResourceCache(
		cache.NewSharedIndexInformer(lw, obj, m.resync, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		}), gvk)

	m.informersByGVK[gvk] = c
	if m.started {
		go c.Run(m.stop)
		if !cache.WaitForCacheSync(m.stop, c.HasSynced) {
			return nil, fmt.Errorf("failed waiting for %T Informer to sync", gvk)
		}
	}
	return c, nil
}

func (m *InformersMap) createListWatcher(gvk schema.GroupVersionKind) (*cache.ListWatch, error) {
	mapping, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	client, err := apiutil.RESTClientForGVK(gvk, m.config, m.codecs)
	if err != nil {
		return nil, err
	}
	listGVK := gvk.GroupVersion().WithKind(gvk.Kind + "List")
	listObj, err := m.Scheme.New(listGVK)
	if err != nil {
		return nil, err
	}

	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			res := listObj.DeepCopyObject()
			isNamespaceScoped := m.namespace != "" && mapping.Scope.Name() != meta.RESTScopeNameRoot
			err := client.Get().
				NamespaceIfScoped(m.namespace, isNamespaceScoped).
				Resource(mapping.Resource.Resource).
				VersionedParams(&opts, m.paramCodec).
				Do().Into(res)
			return res, err
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.Watch = true
			isNamespaceScoped := m.namespace != "" && mapping.Scope.Name() != meta.RESTScopeNameRoot
			return client.Get().
				NamespaceIfScoped(m.namespace, isNamespaceScoped).
				Resource(mapping.Resource.Resource).
				VersionedParams(&opts, m.paramCodec).
				Watch()
		},
	}, nil
}
