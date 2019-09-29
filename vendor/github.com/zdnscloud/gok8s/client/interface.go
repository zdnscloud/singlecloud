package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
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
	Patch(ctx context.Context, obj runtime.Object, typ types.PatchType, data []byte) error
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

type Metrics interface {
	GetNodeMetrics(name string, selector labels.Selector) (*metricsapi.NodeMetricsList, error)
	GetPodMetrics(namespace, name string, selector labels.Selector) (*metricsapi.PodMetricsList, error)
}

type Client interface {
	Reader
	Writer
	Discovery
	StatusClient
	Metrics

	RestClientForObject(obj runtime.Object, timeout time.Duration) (rest.Interface, error)
}
