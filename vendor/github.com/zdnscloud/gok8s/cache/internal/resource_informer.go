package internal

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/cache"

	"github.com/zdnscloud/gok8s/client"
)

var _ client.Reader = &ResourceInformer{}

type ResourceInformer struct {
	cache.SharedIndexInformer
	groupVersionKind schema.GroupVersionKind //this field only used to generate error :(
}

func newResourceCache(informer cache.SharedIndexInformer, groupVersionKind schema.GroupVersionKind) *ResourceInformer {
	return &ResourceInformer{
		SharedIndexInformer: informer,
		groupVersionKind:    groupVersionKind,
	}
}

func (c *ResourceInformer) Get(_ context.Context, key client.ObjectKey, out runtime.Object) error {
	storeKey := objectKeyToStoreKey(key)

	obj, exists, err := c.GetIndexer().GetByKey(storeKey)
	if err != nil {
		return err
	}

	if !exists {
		return errors.NewNotFound(schema.GroupResource{
			Group:    c.groupVersionKind.Group,
			Resource: c.groupVersionKind.Kind,
		}, key.Name)
	}

	if _, isObj := obj.(runtime.Object); !isObj {
		return fmt.Errorf("cache contained %T, which is not an Object", obj)
	}

	obj = obj.(runtime.Object).DeepCopyObject()
	outVal := reflect.ValueOf(out)
	objVal := reflect.ValueOf(obj)
	if !objVal.Type().AssignableTo(outVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", objVal.Type(), outVal.Type())
	}
	reflect.Indirect(outVal).Set(reflect.Indirect(objVal))
	return nil
}

func (c *ResourceInformer) List(ctx context.Context, opts *client.ListOptions, out runtime.Object) error {
	var objs []interface{}
	var err error

	if opts != nil && opts.FieldSelector != nil {
		field, val, requiresExact := requiresExactMatch(opts.FieldSelector)
		if !requiresExact {
			return fmt.Errorf("non-exact field matches are not supported by the cache")
		}
		objs, err = c.GetIndexer().ByIndex(FieldIndexName(field), KeyToNamespacedKey(opts.Namespace, val))
	} else if opts != nil && opts.Namespace != "" {
		objs, err = c.GetIndexer().ByIndex(cache.NamespaceIndex, opts.Namespace)
	} else {
		objs = c.GetIndexer().List()
	}
	if err != nil {
		return err
	}
	var labelSel labels.Selector
	if opts != nil && opts.LabelSelector != nil {
		labelSel = opts.LabelSelector
	}

	outItems, err := c.getListItems(objs, labelSel)
	if err != nil {
		return err
	}
	return apimeta.SetList(out, outItems)
}

func (c *ResourceInformer) getListItems(objs []interface{}, labelSel labels.Selector) ([]runtime.Object, error) {
	outItems := make([]runtime.Object, 0, len(objs))
	for _, item := range objs {
		obj, isObj := item.(runtime.Object)
		if !isObj {
			return nil, fmt.Errorf("cache contained %T, which is not an Object", obj)
		}
		meta, err := apimeta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		if labelSel != nil {
			lbls := labels.Set(meta.GetLabels())
			if !labelSel.Matches(lbls) {
				continue
			}
		}
		outItems = append(outItems, obj.DeepCopyObject())
	}
	return outItems, nil
}

func objectKeyToStoreKey(k client.ObjectKey) string {
	if k.Namespace == "" {
		return k.Name
	}
	return k.Namespace + "/" + k.Name
}

func requiresExactMatch(sel fields.Selector) (field, val string, required bool) {
	reqs := sel.Requirements()
	if len(reqs) != 1 {
		return "", "", false
	}
	req := reqs[0]
	if req.Operator != selection.Equals && req.Operator != selection.DoubleEquals {
		return "", "", false
	}
	return req.Field, req.Value, true
}

func FieldIndexName(field string) string {
	return "field:" + field
}

const allNamespacesNamespace = "__all_namespaces"

func KeyToNamespacedKey(ns string, baseKey string) string {
	if ns != "" {
		return ns + "/" + baseKey
	}
	return allNamespacesNamespace + "/" + baseKey
}
