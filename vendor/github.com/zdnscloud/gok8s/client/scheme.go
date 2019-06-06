package client

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

func GetDefaultScheme() *runtime.Scheme {
	sm := scheme.Scheme
	apiregistrationv1beta1.AddToScheme(sm)
	apiregistrationv1.AddToScheme(sm)
	apiextensionsv1beta1.AddToScheme(sm)
	return sm
}
