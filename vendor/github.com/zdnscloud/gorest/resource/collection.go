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
	var resources []Resource
	typ := collection.GetType()
	if i != nil {
		v := reflect.ValueOf(i)
		if v.Kind() != reflect.Slice {
			return nil, fmt.Errorf("list handler doesn't return slice but %v", v.Kind())
		}

		l := v.Len()
		if l > 0 {
			resources := make([]Resource, 0, l)
			e := v.Index(0)
			if r, ok := e.Interface().(Resource); ok == false {
				return nil, fmt.Errorf("list handler doesn't return slice of resource but %v", e.Kind())
			} else if r.GetID() == "" {
				return nil, fmt.Errorf("list handler get resource doesn't have id")
			} else {
				r.SetType(typ)
				resources = append(resources, r)
			}
		}
	}

	return &ResourceCollection{
		Type:         "collection",
		ResourceType: typ,
		Resources:    resources,
		collection:   collection,
	}, nil
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
