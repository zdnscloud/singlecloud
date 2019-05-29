package types

import (
	"fmt"
	"path"
	"reflect"
	"strings"
	"time"
)

type Collection struct {
	Type         string            `json:"type,omitempty"`
	ResourceType string            `json:"resourceType,omitempty"`
	Links        map[string]string `json:"links,omitempty"`
	Data         interface{}       `json:"data"`
}

type APIVersion struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
}

func (v *APIVersion) GetVersionURL() string {
	return path.Join(GroupPrefix, v.Group, v.Version)
}

type Schema struct {
	Version           APIVersion       `json:"version"`
	PluralName        string           `json:"pluralName,omitempty"`
	ResourceMethods   []string         `json:"resourceMethods,omitempty"`
	ResourceFields    map[string]Field `json:"resourceFields"`
	ResourceActions   []Action         `json:"resourceActions,omitempty"`
	CollectionMethods []string         `json:"collectionMethods,omitempty"`
	CollectionFields  map[string]Field `json:"collectionFields,omitempty"`
	CollectionActions []Action         `json:"collectionActions,omitempty"`

	StructVal reflect.Value `json:"-"`
	Handler   Handler       `json:"-"`
	Parents   []string      `json:"-"`
}

func (s *Schema) GetType() string {
	return strings.ToLower(s.StructVal.Type().Name())
}

type Field struct {
	Type         string      `json:"type,omitempty"`
	Default      interface{} `json:"default,omitempty"`
	Nullable     bool        `json:"nullable,omitempty"`
	Create       bool        `json:"create"`
	Required     bool        `json:"required,omitempty"`
	Update       bool        `json:"update"`
	MinLength    *int64      `json:"minLength,omitempty"`
	MaxLength    *int64      `json:"maxLength,omitempty"`
	Min          *int64      `json:"min,omitempty"`
	Max          *int64      `json:"max,omitempty"`
	Options      []string    `json:"options,omitempty"`
	ValidChars   string      `json:"validChars,omitempty"`
	InvalidChars string      `json:"invalidChars,omitempty"`
	CodeName     string      `json:"-"`
}

type Action struct {
	Name   string
	Input  interface{} `json:"input,omitempty"`
	Output string      `json:"output,omitempty"`
}

type ActionHandler func(request *Context, action *Action) *APIError

type RequestHandler func(request *Context) *APIError

type Resource struct {
	ID                string            `json:"id,omitempty"`
	Type              string            `json:"type,omitempty"`
	Links             map[string]string `json:"links,omitempty"`
	CreationTimestamp ISOTime           `json:"creationTimestamp,omitempty"`
	Parent            Object            `json:"-"`
	Schema            *Schema           `json:"-"`
}

func (r *Resource) GetID() string {
	return r.ID
}

func (r *Resource) SetID(id string) {
	r.ID = id
}

func (r *Resource) GetType() string {
	return r.Type
}

func (r *Resource) SetType(typ string) {
	r.Type = typ
}

func (r *Resource) GetLinks() map[string]string {
	return r.Links
}

func (r *Resource) SetLinks(links map[string]string) {
	r.Links = links
}

func (r *Resource) GetCreationTimestamp() time.Time {
	return time.Time(r.CreationTimestamp)
}

func (r *Resource) SetCreationTimestamp(timestamp time.Time) {
	r.CreationTimestamp = ISOTime(timestamp)
}

func (r *Resource) GetParent() Object {
	return r.Parent
}

func (r *Resource) SetParent(parent Object) {
	r.Parent = parent
}

func (r *Resource) GetSchema() *Schema {
	return r.Schema
}

func (r *Resource) SetSchema(schema *Schema) {
	r.Schema = schema
}

func GetAncestors(parent ObjectParent) []Object {
	var antiAncestors []Object
	for obj := parent.GetParent(); obj != nil; obj = obj.GetParent() {
		antiAncestors = append(antiAncestors, obj)
	}

	var ancestors []Object
	for i := len(antiAncestors) - 1; i >= 0; i-- {
		ancestors = append(ancestors, antiAncestors[i])
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
