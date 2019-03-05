package cache

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"github.com/zdnscloud/gok8s/cache/internal"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/apiutil"
)

var (
	_ Cache = &informerCache{}
)

type informerCache struct {
	*internal.InformersMap
}

func (c *informerCache) Get(ctx context.Context, key client.ObjectKey, out runtime.Object) error {
	gvk, err := apiutil.GVKForObject(out, c.Scheme)
	if err != nil {
		return err
	}

	reader, err := c.InformersMap.GetInformer(gvk)
	if err != nil {
		return err
	}
	return reader.Get(ctx, key, out)
}

func (c *informerCache) List(ctx context.Context, opts *client.ListOptions, out runtime.Object) error {
	gvk, err := apiutil.GVKForObject(out, c.Scheme)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(gvk.Kind, "List") {
		return fmt.Errorf("non-list type %T (kind %q) passed as output", out, gvk)
	}

	gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
	itemsPtr, err := apimeta.GetItemsPtr(out)
	if err != nil {
		return nil
	}

	elemType := reflect.Indirect(reflect.ValueOf(itemsPtr)).Type().Elem()
	cacheTypeValue := reflect.Zero(reflect.PtrTo(elemType))
	if _, ok := cacheTypeValue.Interface().(runtime.Object); ok == false {
		return fmt.Errorf("cannot get cache for %T, its element %T is not a runtime.Object", out, cacheTypeValue.Interface())
	}

	reader, err := c.InformersMap.GetInformer(gvk)
	if err != nil {
		return err
	}

	return reader.List(ctx, opts, out)
}

func (c *informerCache) GetInformerForKind(gvk schema.GroupVersionKind) (cache.SharedIndexInformer, error) {
	return c.InformersMap.GetInformer(gvk)
}

func (c *informerCache) GetInformer(obj runtime.Object) (cache.SharedIndexInformer, error) {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme)
	if err != nil {
		return nil, err
	}
	return c.InformersMap.GetInformer(gvk)
}

func (c *informerCache) IndexField(obj runtime.Object, field string, extractValue IndexerFunc) error {
	informer, err := c.GetInformer(obj)
	if err != nil {
		return err
	}
	return indexByField(informer.GetIndexer(), field, extractValue)
}

func indexByField(indexer cache.Indexer, field string, extractor IndexerFunc) error {
	indexFunc := func(objRaw interface{}) ([]string, error) {
		obj, isObj := objRaw.(runtime.Object)
		if !isObj {
			return nil, fmt.Errorf("object of type %T is not an Object", objRaw)
		}
		meta, err := apimeta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		ns := meta.GetNamespace()

		rawVals := extractor(obj)
		var vals []string
		if ns == "" {
			vals = rawVals
		} else {
			vals = make([]string, len(rawVals)*2)
		}
		for i, rawVal := range rawVals {
			vals[i] = internal.KeyToNamespacedKey(ns, rawVal)
			if ns != "" {
				vals[i+len(rawVals)] = internal.KeyToNamespacedKey("", rawVal)
			}
		}

		return vals, nil
	}

	return indexer.AddIndexers(cache.Indexers{internal.FieldIndexName(field): indexFunc})
}
