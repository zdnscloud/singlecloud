package resource

import (
	"fmt"
	"time"
)

type ResourceBase struct {
	ID                string                            `json:"id,omitempty"`
	Type              string                            `json:"type,omitempty"`
	Links             map[ResourceLinkType]ResourceLink `json:"links,omitempty"`
	CreationTimestamp ISOTime                           `json:"creationTimestamp,omitempty"`

	action *Action  `json:"-"`
	parent Resource `json:"-"`
	schema Schema   `json:"-"`
}

func (r ResourceBase) GetParents() []ResourceKind {
	return nil
}

func (r ResourceBase) CreateDefaultResource() Resource {
	return nil
}

func (r ResourceBase) CreateAction(name string) *Action {
	return nil
}

var _ ResourceKind = ResourceBase{}

func (r *ResourceBase) GetID() string {
	return r.ID
}

func (r *ResourceBase) SetID(id string) {
	r.ID = id
}

func (r *ResourceBase) GetLinks() map[ResourceLinkType]ResourceLink {
	return r.Links
}

func (r *ResourceBase) SetLinks(links map[ResourceLinkType]ResourceLink) {
	r.Links = links
}

func (r *ResourceBase) GetCreationTimestamp() time.Time {
	return time.Time(r.CreationTimestamp)
}

func (r *ResourceBase) SetCreationTimestamp(timestamp time.Time) {
	r.CreationTimestamp = ISOTime(timestamp)
}

func (r *ResourceBase) GetParent() Resource {
	return r.parent
}

func (r *ResourceBase) SetParent(parent Resource) {
	r.parent = parent
}

func (r *ResourceBase) GetSchema() Schema {
	return r.schema
}

func (r *ResourceBase) SetSchema(schema Schema) {
	r.schema = schema
}

func (r *ResourceBase) GetAction() *Action {
	return r.action
}

func (r *ResourceBase) SetAction(action *Action) {
	r.action = action
}

func (r *ResourceBase) SetType(typ string) {
	r.Type = typ
}

func (r *ResourceBase) GetType() string {
	return r.Type
}

var _ Resource = &ResourceBase{}

func GetAncestors(r Resource) []Resource {
	var ancestors []Resource
	for r := r.GetParent(); r != nil; r = r.GetParent() {
		ancestors = append(ancestors, r)
	}
	//reverse
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}
	return ancestors
}

type ISOTime time.Time

func (t ISOTime) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}

	return []byte(fmt.Sprintf("\"%s\"", time.Time(t).Format(time.RFC3339))), nil
}

func (t *ISOTime) UnmarshalJSON(data []byte) (err error) {
	if len(data) == 4 && string(data) == "null" {
		*t = ISOTime(time.Time{})
		return
	}

	now, err := time.Parse(`"`+time.RFC3339+`"`, string(data))
	*t = ISOTime(now)
	return
}
