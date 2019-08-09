package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/zdnscloud/gok8s/scheme"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: "storage.zcloud.cn", Version: "v1"}
)

func AddToScheme(s *runtime.Scheme) {
	builder := &scheme.Builder{GroupVersion: SchemeGroupVersion}
	builder.Register(&Cluster{}, &ClusterList{})
	builder.AddToScheme(s)
}
