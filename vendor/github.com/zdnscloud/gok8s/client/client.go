package client

import (
	"context"
	"errors"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsV1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/zdnscloud/gok8s/client/apiutil"
	"github.com/zdnscloud/gok8s/util"
)

var (
	errMetricsServerIsNotValiable   = errors.New("metrics server isn't available")
	errDiscoveryServerIsNotValiable = errors.New("discovery server isn't available")
)

type Options struct {
	// Scheme, used to map go structs to GroupVersionKinds
	Scheme *runtime.Scheme

	// Mapper, will be used to map GroupVersionKinds to Resources
	Mapper meta.RESTMapper
}

func New(config *rest.Config, options Options) (Client, error) {
	util.Assert(config != nil, "nil rest config is provided")

	// Init a scheme if none provided
	if options.Scheme == nil {
		options.Scheme = GetDefaultScheme()
	}

	// Init a Mapper if none provided
	if options.Mapper == nil {
		mapper, err := apiutil.NewDiscoveryRESTMapper(config)
		if err != nil {
			return nil, err
		} else {
			options.Mapper = mapper
		}
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	originalTimeout := config.Timeout
	config.Timeout = 10 * time.Second
	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		metricsClient = nil
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		discoveryClient = nil
	}

	config.Timeout = originalTimeout
	return &client{
		discoveryClient: discoveryClient,
		metricsClient:   metricsClient,
		typedClient: typedClient{
			cache: clientCache{
				config:         config,
				scheme:         options.Scheme,
				mapper:         options.Mapper,
				codecs:         serializer.NewCodecFactory(options.Scheme),
				resourceByType: make(map[reflect.Type]*resourceMeta),
			},
			paramCodec: runtime.NewParameterCodec(options.Scheme),
		},

		unstructuredClient: unstructuredClient{
			client:     dynamicClient,
			restMapper: options.Mapper,
		},
	}, nil
}

var _ Client = &client{}

type client struct {
	typedClient        typedClient
	unstructuredClient unstructuredClient
	discoveryClient    *discovery.DiscoveryClient
	metricsClient      metricsclientset.Interface
}

func (c *client) ServerVersion() (*version.Info, error) {
	if c.discoveryClient != nil {
		return c.discoveryClient.ServerVersion()
	} else {
		return nil, errDiscoveryServerIsNotValiable
	}
}

func (c *client) GetNodeMetrics(name string, selector labels.Selector) (*metricsapi.NodeMetricsList, error) {
	if c.metricsClient == nil {
		return nil, errMetricsServerIsNotValiable
	}

	var err error
	versionedMetrics := &metricsV1beta1api.NodeMetricsList{}
	nm := c.metricsClient.MetricsV1beta1().NodeMetricses()
	if name != "" {
		m, err := nm.Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		versionedMetrics.Items = []metricsV1beta1api.NodeMetrics{*m}
	} else {
		versionedMetrics, err = nm.List(metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return nil, err
		}
	}
	metrics := &metricsapi.NodeMetricsList{}
	err = metricsV1beta1api.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func (c *client) GetPodMetrics(namespace, name string, selector labels.Selector) (*metricsapi.PodMetricsList, error) {
	if c.metricsClient == nil {
		return nil, errMetricsServerIsNotValiable
	}

	var err error
	versionedMetrics := &metricsV1beta1api.PodMetricsList{}
	nm := c.metricsClient.MetricsV1beta1().PodMetricses(namespace)
	if name != "" {
		m, err := nm.Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		versionedMetrics.Items = []metricsV1beta1api.PodMetrics{*m}
	} else {
		versionedMetrics, err = nm.List(metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return nil, err
		}
	}
	metrics := &metricsapi.PodMetricsList{}
	err = metricsV1beta1api.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func (c *client) RestClientForObject(obj runtime.Object, timeout time.Duration) (rest.Interface, error) {
	return c.typedClient.RestClientForObject(obj, timeout)
}

func (c *client) Create(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return c.typedClient.Create(ctx, obj)
	} else {
		return c.unstructuredClient.Create(ctx, obj)
	}
}

func (c *client) Update(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return c.typedClient.Update(ctx, obj)
	} else {
		return c.unstructuredClient.Update(ctx, obj)
	}
}

func (c *client) Patch(ctx context.Context, obj runtime.Object, typ types.PatchType, data []byte) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return c.typedClient.Patch(ctx, obj, typ, data)
	} else {
		return c.unstructuredClient.Patch(ctx, obj, typ, data)
	}
}

func (c *client) Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return c.typedClient.Delete(ctx, obj, opts...)
	} else {
		return c.unstructuredClient.Delete(ctx, obj, opts...)
	}
}

func (c *client) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return c.typedClient.Get(ctx, key, obj)
	} else {
		return c.unstructuredClient.Get(ctx, key, obj)
	}
}

func (c *client) List(ctx context.Context, opts *ListOptions, obj runtime.Object) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return c.typedClient.List(ctx, opts, obj)
	} else {
		return c.unstructuredClient.List(ctx, opts, obj)
	}
}

func (c *client) Status() StatusWriter {
	return &statusWriter{client: c}
}

type statusWriter struct {
	client *client
}

var _ StatusWriter = &statusWriter{}

func (sw *statusWriter) Update(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok == false {
		return sw.client.typedClient.UpdateStatus(ctx, obj)
	} else {
		return sw.client.unstructuredClient.UpdateStatus(ctx, obj)
	}
}
