package cache

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/zdnscloud/gok8s/cache/internal"
	"github.com/zdnscloud/gok8s/client/apiutil"
)

type Options struct {
	Scheme    *runtime.Scheme
	Mapper    meta.RESTMapper
	Resync    *time.Duration
	Namespace string
}

var defaultResyncTime = 10 * time.Hour

func New(config *rest.Config, opts Options) (Cache, error) {
	opts, err := defaultOpts(config, opts)
	if err != nil {
		return nil, err
	}
	im := internal.NewInformersMap(config, opts.Scheme, opts.Mapper, *opts.Resync, opts.Namespace)
	return &informerCache{InformersMap: im}, nil
}

func defaultOpts(config *rest.Config, opts Options) (Options, error) {
	if opts.Scheme == nil {
		opts.Scheme = scheme.Scheme
	}

	if opts.Mapper == nil {
		var err error
		opts.Mapper, err = apiutil.NewDiscoveryRESTMapper(config)
		if err != nil {
			return opts, fmt.Errorf("could not create RESTMapper from config")
		}
	}

	if opts.Resync == nil {
		opts.Resync = &defaultResyncTime
	}
	return opts, nil
}
