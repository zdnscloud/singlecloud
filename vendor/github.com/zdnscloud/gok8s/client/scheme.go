package client

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

var defaultSmInitOnce sync.Once
var defaultScheme *runtime.Scheme

func GetDefaultScheme() *runtime.Scheme {
	defaultSmInitOnce.Do(func() {
		defaultScheme = scheme.Scheme
		apiregistrationv1beta1.AddToScheme(defaultScheme)
		apiregistrationv1.AddToScheme(defaultScheme)
		apiextensionsv1beta1.AddToScheme(defaultScheme)
	})

	return defaultScheme
}
