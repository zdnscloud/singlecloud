package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

type typedClient struct {
	cache      clientCache
	paramCodec runtime.ParameterCodec
}

func (c *typedClient) RestClientForObject(obj runtime.Object, timeout time.Duration) (rest.Interface, error) {
	return c.cache.newRestClient(obj, timeout)
}

func (c *typedClient) Create(ctx context.Context, obj runtime.Object) error {
	o, err := c.cache.getObjMeta(obj)
	if err != nil {
		return err
	}
	return o.Post().
		NamespaceIfScoped(o.GetNamespace(), o.isNamespaced()).
		Resource(o.resource()).
		Body(obj).
		Context(ctx).
		Do().
		Into(obj)
}

func (c *typedClient) Update(ctx context.Context, obj runtime.Object) error {
	o, err := c.cache.getObjMeta(obj)
	if err != nil {
		return err
	}
	return o.Put().
		NamespaceIfScoped(o.GetNamespace(), o.isNamespaced()).
		Resource(o.resource()).
		Name(o.GetName()).
		Body(obj).
		Context(ctx).
		Do().
		Into(obj)
}

func (c *typedClient) Patch(ctx context.Context, obj runtime.Object, typ types.PatchType, data []byte) error {
	o, err := c.cache.getObjMeta(obj)
	if err != nil {
		return err
	}
	return o.Patch(typ).
		NamespaceIfScoped(o.GetNamespace(), o.isNamespaced()).
		Resource(o.resource()).
		Name(o.GetName()).
		Body(data).
		Context(ctx).
		Do().
		Into(obj)
}

func (c *typedClient) Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error {
	o, err := c.cache.getObjMeta(obj)
	if err != nil {
		return err
	}

	deleteOpts := DeleteOptions{}
	return o.Delete().
		NamespaceIfScoped(o.GetNamespace(), o.isNamespaced()).
		Resource(o.resource()).
		Name(o.GetName()).
		Body(deleteOpts.ApplyOptions(opts).AsDeleteOptions()).
		Context(ctx).
		Do().
		Error()
}

func (c *typedClient) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error {
	r, err := c.cache.getResource(obj)
	if err != nil {
		return err
	}
	return r.Get().
		NamespaceIfScoped(key.Namespace, r.isNamespaced()).
		Resource(r.resource()).
		Context(ctx).
		Name(key.Name).Do().Into(obj)
}

func (c *typedClient) List(ctx context.Context, opts *ListOptions, obj runtime.Object) error {
	r, err := c.cache.getResource(obj)
	if err != nil {
		return err
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	return r.Get().
		NamespaceIfScoped(namespace, r.isNamespaced()).
		Resource(r.resource()).
		Body(obj).
		VersionedParams(opts.AsListOptions(), c.paramCodec).
		Context(ctx).
		Do().
		Into(obj)
}

func (c *typedClient) UpdateStatus(ctx context.Context, obj runtime.Object) error {
	o, err := c.cache.getObjMeta(obj)
	if err != nil {
		return err
	}

	return o.Put().
		NamespaceIfScoped(o.GetNamespace(), o.isNamespaced()).
		Resource(o.resource()).
		Name(o.GetName()).
		SubResource("status").
		Body(obj).
		Context(ctx).
		Do().
		Into(obj)
}
