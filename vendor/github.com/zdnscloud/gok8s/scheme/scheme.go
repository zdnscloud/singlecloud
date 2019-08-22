package scheme

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Builder struct {
	GroupVersion schema.GroupVersion
	runtime.SchemeBuilder
}

func New(group, version string) *Builder {
	return &Builder{
		GroupVersion: schema.GroupVersion{Group: group, Version: version},
	}
}

func (bld *Builder) Register(object ...runtime.Object) *Builder {
	bld.SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(bld.GroupVersion, object...)
		metav1.AddToGroupVersion(scheme, bld.GroupVersion)
		return nil
	})
	return bld
}

func (bld *Builder) RegisterAll(b *Builder) *Builder {
	bld.SchemeBuilder = append(bld.SchemeBuilder, b.SchemeBuilder...)
	return bld
}

func (bld *Builder) AddToScheme(s *runtime.Scheme) error {
	return bld.SchemeBuilder.AddToScheme(s)
}

func (bld *Builder) Build() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	return s, bld.AddToScheme(s)
}
