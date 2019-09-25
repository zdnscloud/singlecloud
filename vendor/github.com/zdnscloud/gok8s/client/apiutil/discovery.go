package apiutil

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// APIGroupResources is an API group with a mapping of versions to
// resources.
type APIGroupResources struct {
	Group metav1.APIGroup
	// A mapping of version string to a slice of APIResources for
	// that version.
	VersionedResources map[string][]metav1.APIResource
}

// NewDiscoveryRESTMapper returns a PriorityRESTMapper based on the discovered
// groups and resources passed in.
func newDiscoveryRESTMapper(groupResources []*APIGroupResources) meta.RESTMapper {
	unionMapper := meta.MultiRESTMapper{}

	var groupPriority []string
	// /v1 is special.  It should always come first
	resourcePriority := []schema.GroupVersionResource{{Group: "", Version: "v1", Resource: meta.AnyResource}}
	kindPriority := []schema.GroupVersionKind{{Group: "", Version: "v1", Kind: meta.AnyKind}}

	for _, group := range groupResources {
		groupPriority = append(groupPriority, group.Group.Name)

		// Make sure the preferred version comes first
		if len(group.Group.PreferredVersion.Version) != 0 {
			preferred := group.Group.PreferredVersion.Version
			if _, ok := group.VersionedResources[preferred]; ok {
				resourcePriority = append(resourcePriority, schema.GroupVersionResource{
					Group:    group.Group.Name,
					Version:  group.Group.PreferredVersion.Version,
					Resource: meta.AnyResource,
				})

				kindPriority = append(kindPriority, schema.GroupVersionKind{
					Group:   group.Group.Name,
					Version: group.Group.PreferredVersion.Version,
					Kind:    meta.AnyKind,
				})
			}
		}

		for _, discoveryVersion := range group.Group.Versions {
			resources, ok := group.VersionedResources[discoveryVersion.Version]
			if !ok {
				continue
			}

			// Add non-preferred versions after the preferred version, in case there are resources that only exist in those versions
			if discoveryVersion.Version != group.Group.PreferredVersion.Version {
				resourcePriority = append(resourcePriority, schema.GroupVersionResource{
					Group:    group.Group.Name,
					Version:  discoveryVersion.Version,
					Resource: meta.AnyResource,
				})

				kindPriority = append(kindPriority, schema.GroupVersionKind{
					Group:   group.Group.Name,
					Version: discoveryVersion.Version,
					Kind:    meta.AnyKind,
				})
			}

			gv := schema.GroupVersion{Group: group.Group.Name, Version: discoveryVersion.Version}
			versionMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gv})

			for _, resource := range resources {
				scope := meta.RESTScopeNamespace
				if !resource.Namespaced {
					scope = meta.RESTScopeRoot
				}

				// if we have a slash, then this is a subresource and we shouldn't create mappings for those.
				if strings.Contains(resource.Name, "/") {
					continue
				}

				plural := gv.WithResource(resource.Name)
				singular := gv.WithResource(resource.SingularName)
				// this is for legacy resources and servers which don't list singular forms.  For those we must still guess.
				if len(resource.SingularName) == 0 {
					_, singular = meta.UnsafeGuessKindToResource(gv.WithKind(resource.Kind))
				}

				versionMapper.AddSpecific(gv.WithKind(strings.ToLower(resource.Kind)), plural, singular, scope)
				versionMapper.AddSpecific(gv.WithKind(resource.Kind), plural, singular, scope)
				// TODO this is producing unsafe guesses that don't actually work, but it matches previous behavior
				versionMapper.Add(gv.WithKind(resource.Kind+"List"), scope)
			}
			// TODO why is this type not in discovery (at least for "v1")
			versionMapper.Add(gv.WithKind("List"), meta.RESTScopeRoot)
			unionMapper = append(unionMapper, versionMapper)
		}
	}

	for _, group := range groupPriority {
		resourcePriority = append(resourcePriority, schema.GroupVersionResource{
			Group:    group,
			Version:  meta.AnyVersion,
			Resource: meta.AnyResource,
		})
		kindPriority = append(kindPriority, schema.GroupVersionKind{
			Group:   group,
			Version: meta.AnyVersion,
			Kind:    meta.AnyKind,
		})
	}

	return meta.PriorityRESTMapper{
		Delegate:         unionMapper,
		ResourcePriority: resourcePriority,
		KindPriority:     kindPriority,
	}
}

// GetAPIGroupResources uses the provided discovery client to gather
// discovery information and populate a slice of APIGroupResources.
func GetAPIGroupResources(cl discovery.DiscoveryInterface) ([]*APIGroupResources, error) {
	apiGroups, err := cl.ServerGroups()
	if err != nil {
		if apiGroups == nil || len(apiGroups.Groups) == 0 {
			return nil, err
		}
		// TODO track the errors and update callers to handle partial errors.
	}
	var result []*APIGroupResources
	for _, group := range apiGroups.Groups {
		groupResources := &APIGroupResources{
			Group:              group,
			VersionedResources: make(map[string][]metav1.APIResource),
		}
		for _, version := range group.Versions {
			resources, err := cl.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				// continue as best we can
				// TODO track the errors and update callers to handle partial errors.
				if resources == nil || len(resources.APIResources) == 0 {
					continue
				}
			}
			groupResources.VersionedResources[version.Version] = resources.APIResources
		}
		result = append(result, groupResources)
	}
	return result, nil
}

// DeferredDiscoveryRESTMapper is a RESTMapper that will defer
// initialization of the RESTMapper until the first mapping is
// requested.
type DeferredDiscoveryRESTMapper struct {
	initMu   sync.Mutex
	delegate meta.RESTMapper
	cl       discovery.CachedDiscoveryInterface
}

// NewDeferredDiscoveryRESTMapper returns a
// DeferredDiscoveryRESTMapper that will lazily query the provided
// client for discovery information to do REST mappings.
func NewDeferredDiscoveryRESTMapper(cl discovery.CachedDiscoveryInterface) *DeferredDiscoveryRESTMapper {
	return &DeferredDiscoveryRESTMapper{
		cl: cl,
	}
}

// reset resets the internally cached Discovery information and will
// cause the next mapping request to re-discover.
func (d *DeferredDiscoveryRESTMapper) reset() error {
	d.cl.Invalidate()
	groupResources, err := GetAPIGroupResources(d.cl)
	if err == nil {
		d.delegate = newDiscoveryRESTMapper(groupResources)
	} else {
		d.delegate = nil
	}
	return err
}

// KindFor takes a partial resource and returns back the single match.
// It returns an error if there are multiple matches.
func (d *DeferredDiscoveryRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		gvk, err := d.delegate.KindFor(resource)
		if err == nil {
			return gvk, nil
		}
	}

	if err := d.reset(); err != nil {
		return schema.GroupVersionKind{}, err
	}

	return d.delegate.KindFor(resource)
}

// KindsFor takes a partial resource and returns back the list of
// potential kinds in priority order.
func (d *DeferredDiscoveryRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		gvks, err := d.delegate.KindsFor(resource)
		if err == nil && len(gvks) > 0 {
			return gvks, nil
		}
	}

	if err := d.reset(); err != nil {
		return nil, err
	}

	return d.delegate.KindsFor(resource)
}

// ResourceFor takes a partial resource and returns back the single
// match. It returns an error if there are multiple matches.
func (d *DeferredDiscoveryRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		gvr, err := d.delegate.ResourceFor(input)
		if err == nil {
			return gvr, nil
		}
	}

	if err := d.reset(); err != nil {
		return schema.GroupVersionResource{}, err
	}

	return d.delegate.ResourceFor(input)
}

// ResourcesFor takes a partial resource and returns back the list of
// potential resource in priority order.
func (d *DeferredDiscoveryRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		gvrs, err := d.delegate.ResourcesFor(input)
		if err == nil && len(gvrs) > 0 {
			return gvrs, nil
		}
	}

	if err := d.reset(); err != nil {
		return nil, err
	}

	return d.delegate.ResourcesFor(input)
}

// RESTMapping identifies a preferred resource mapping for the
// provided group kind.
func (d *DeferredDiscoveryRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		m, err := d.delegate.RESTMapping(gk, versions...)
		if err == nil {
			return m, nil
		}
	}

	if err := d.reset(); err != nil {
		return nil, err
	}

	return d.delegate.RESTMapping(gk, versions...)
}

// RESTMappings returns the RESTMappings for the provided group kind
// in a rough internal preferred order. If no kind is found, it will
// return a NoResourceMatchError.
func (d *DeferredDiscoveryRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		ms, err := d.delegate.RESTMappings(gk, versions...)
		if err == nil && len(ms) > 0 {
			return ms, nil
		}
	}

	if err := d.reset(); err != nil {
		return nil, err
	}

	return d.delegate.RESTMappings(gk, versions...)
}

// ResourceSingularizer converts a resource name from plural to
// singular (e.g., from pods to pod).
func (d *DeferredDiscoveryRESTMapper) ResourceSingularizer(resource string) (string, error) {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate != nil {
		singular, err := d.delegate.ResourceSingularizer(resource)
		if err == nil {
			return singular, nil
		}
	}

	if err := d.reset(); err != nil {
		return "", err
	}

	return d.delegate.ResourceSingularizer(resource)
}

func (d *DeferredDiscoveryRESTMapper) String() string {
	d.initMu.Lock()
	defer d.initMu.Unlock()

	if d.delegate == nil {
		if err := d.reset(); err != nil {
			return fmt.Sprintf("DeferredDiscoveryRESTMapper{%v}", err)
		}
	}

	return fmt.Sprintf("DeferredDiscoveryRESTMapper{\n\t%v\n}", d.delegate)
}

// Make sure it satisfies the interface
var _ meta.RESTMapper = &DeferredDiscoveryRESTMapper{}
