package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
)

type ObjectKey = types.NamespacedName //"<namespace>/<name>"

func ObjectKeyFromObject(obj runtime.Object) (ObjectKey, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return ObjectKey{}, err
	}
	return ObjectKey{Namespace: accessor.GetNamespace(), Name: accessor.GetName()}, nil
}

type Reader interface {
	Get(ctx context.Context, key ObjectKey, obj runtime.Object) error
	List(ctx context.Context, opts *ListOptions, list runtime.Object) error
}

type Writer interface {
	Create(ctx context.Context, obj runtime.Object) error
	Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error
	Update(ctx context.Context, obj runtime.Object) error
}

type StatusClient interface {
	Status() StatusWriter
}

type StatusWriter interface {
	Update(ctx context.Context, obj runtime.Object) error
}

type Discovery interface {
	ServerVersion() (*version.Info, error)
}

type Client interface {
	Reader
	Writer
	Discovery
	StatusClient
}
