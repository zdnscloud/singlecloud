package scheme

import (
	"fmt"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	ut "github.com/zdnscloud/cement/unittest"
)

func kindsAreKnown(t *testing.T, knownTypes map[schema.GroupVersionKind]reflect.Type, gv schema.GroupVersion, kinds []string) {
	for _, kind := range kinds {
		_, ok := knownTypes[gv.WithKind(kind)]
		ut.Assert(t, ok, fmt.Sprintf("should include kind %s", kind))
	}
}

func TestScheme(t *testing.T) {
	gv := schema.GroupVersion{Group: "core", Version: "v1"}
	s, err := New("core", "v1").
		Register(&corev1.Pod{}, &corev1.PodList{}).
		Build()
	ut.Assert(t, err == nil, fmt.Sprintf("build scheme get error: %v", err))

	knownTypes := s.AllKnownTypes()
	ut.Equal(t, len(knownTypes), 13)
	ut.Equal(t, knownTypes[gv.WithKind("Pod")], reflect.TypeOf(corev1.Pod{}))
	ut.Equal(t, knownTypes[gv.WithKind("PodList")], reflect.TypeOf(corev1.PodList{}))

	kindsAreKnown(t, knownTypes, gv, []string{"DeleteOptions", "ExportOptions", "GetOptions", "ListOptions", "WatchEvent"})

	internalGv := schema.GroupVersion{Group: "core", Version: "__internal"}
	kindsAreKnown(t, knownTypes, internalGv, []string{"WatchEvent"})

	emptyGv := schema.GroupVersion{Group: "", Version: "v1"}
	kindsAreKnown(t, knownTypes, emptyGv, []string{"APIGroup", "APIGroupList", "APIResourceList", "APIVersions", "Status"})
}

func TestSchemeWithMultiBuilder(t *testing.T) {
	gv1 := schema.GroupVersion{Group: "core", Version: "v1"}
	b1 := New("core", "v1").Register(&corev1.Pod{}, &corev1.PodList{})

	gv2 := schema.GroupVersion{Group: "apps", Version: "v1"}
	s, err := New("apps", "v1").
		Register(&appsv1.Deployment{}).
		Register(&appsv1.DeploymentList{}).
		RegisterAll(b1).
		Build()

	ut.Assert(t, err == nil, fmt.Sprintf("build scheme get error: %v", err))

	knownTypes := s.AllKnownTypes()
	ut.Equal(t, len(knownTypes), 21)

	kindsAreKnown(t, knownTypes, gv1, []string{"Pod", "PodList"})
	kindsAreKnown(t, knownTypes, gv2, []string{"Deployment", "DeploymentList"})
	kindsAreKnown(t, knownTypes, gv1, []string{"DeleteOptions", "ExportOptions", "GetOptions", "ListOptions", "WatchEvent"})

	internalGv1 := schema.GroupVersion{Group: "apps", Version: "__internal"}
	kindsAreKnown(t, knownTypes, internalGv1, []string{"WatchEvent"})

	emptyGv := schema.GroupVersion{Group: "", Version: "v1"}
	kindsAreKnown(t, knownTypes, emptyGv, []string{"APIGroup", "APIGroupList", "APIResourceList", "APIVersions", "Status"})
}
