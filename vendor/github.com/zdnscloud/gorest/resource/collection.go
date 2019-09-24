package resource

import (
	"fmt"
	"reflect"
)

type ResourceCollection struct {
	Type         string                            `json:"type,omitempty"`
	ResourceType string                            `json:"resourceType,omitempty"`
	Links        map[ResourceLinkType]ResourceLink `json:"links,omitempty"`
	Resources    []Resource                        `json:"data"`

	collection Resource `json:"-"`
}

func NewResourceCollection(collection Resource, i interface{}) (*ResourceCollection, error) {
	typ := collection.GetType()
	rs, err := interfaceToResourceCollection(typ, i)
	if err != nil {
		return nil, err
	} else {
		return &ResourceCollection{
			Type:         "collection",
			ResourceType: typ,
			Resources:    rs,
			collection:   collection,
		}, nil
	}
}

func interfaceToResourceCollection(typ string, i interface{}) ([]Resource, error) {
	if i == nil {
		return []Resource{}, nil
	}

	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("list handler doesn't return slice but %v", v.Kind())
	}
	l := v.Len()
	if l == 0 {
		return []Resource{}, nil
	}

	resources := make([]Resource, 0, l)
	for i := 0; i < l; i++ {
		if r, err := valueToResource(typ, v.Index(i)); err != nil {
			return nil, err
		} else {
			r.SetType(typ)
			resources = append(resources, r)
		}
	}

	return resources, nil
}

func valueToResource(typ string, e reflect.Value) (Resource, error) {
	if e.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("resource isn't pointer but %v", e.Kind())
	}

	if e.IsNil() {
		return nil, fmt.Errorf("resource is nil")
	}

	if rk, ok := e.Elem().Interface().(ResourceKind); ok == false {
		return nil, fmt.Errorf("resource isn't a pointer to ResourceKind but %v", e)
	} else if DefaultKindName(rk) != typ {
		return nil, fmt.Errorf("resource with kind %v isn't same with the collection %v ", DefaultKindName(rk), typ)
	}

	r, ok := e.Interface().(Resource)
	if ok == false {
		return nil, fmt.Errorf("resource %v doesn't implement Resource interface", e.Kind())
	}

	return r, nil
}

func (rc *ResourceCollection) SetLinks(links map[ResourceLinkType]ResourceLink) {
	rc.Links = links
}

func (rc *ResourceCollection) GetLinks() map[ResourceLinkType]ResourceLink {
	return rc.Links
}

func (rc *ResourceCollection) GetCollection() Resource {
	return rc.collection
}

func (rc *ResourceCollection) GetResources() []Resource {
	return rc.Resources
}
