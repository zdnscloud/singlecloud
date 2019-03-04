package types

import (
	"reflect"
)

type Collection struct {
	Type         string      `json:"type,omitempty"`
	ResourceType string      `json:"resourceType,omitempty"`
	Data         interface{} `json:"data"`
}

type APIVersion struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
}

type Schema struct {
	ID                string            `json:"id,omitempty"`
	Type              string            `json:"type,omitempty"`
	Version           APIVersion        `json:"version"`
	PluralName        string            `json:"pluralName,omitempty"`
	ResourceMethods   []string          `json:"resourceMethods,omitempty"`
	ResourceFields    map[string]Field  `json:"resourceFields"`
	ResourceActions   map[string]Action `json:"resourceActions,omitempty"`
	CollectionMethods []string          `json:"collectionMethods,omitempty"`
	CollectionFields  map[string]Field  `json:"collectionFields,omitempty"`
	CollectionActions map[string]Action `json:"collectionActions,omitempty"`

	StructVal reflect.Value `json:"-"`
	Handler   Handler       `json:"-"`
	Parent    string        `json:"-"`
}

type Parent struct {
	ID   string `json:"-"`
	Name string `json:"-"`
}

type Field struct {
	Type         string      `json:"type,omitempty"`
	Default      interface{} `json:"default,omitempty"`
	Nullable     bool        `json:"nullable,omitempty"`
	Create       bool        `json:"create"`
	WriteOnly    bool        `json:"writeOnly,omitempty"`
	Required     bool        `json:"required,omitempty"`
	Update       bool        `json:"update"`
	MinLength    *int64      `json:"minLength,omitempty"`
	MaxLength    *int64      `json:"maxLength,omitempty"`
	Min          *int64      `json:"min,omitempty"`
	Max          *int64      `json:"max,omitempty"`
	Options      []string    `json:"options,omitempty"`
	ValidChars   string      `json:"validChars,omitempty"`
	InvalidChars string      `json:"invalidChars,omitempty"`
	Description  string      `json:"description,omitempty"`
	CodeName     string      `json:"-"`
	DynamicField bool        `json:"dynamicField,omitempty"`
}

type Action struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

type ActionHandler func(request *APIContext, action *Action) *APIError

type RequestHandler func(request *APIContext) *APIError

type Resource struct {
	ID     string `json:"id,omitempty"`
	Type   string `json:"type,omitempty"`
	Parent Parent `json:"-"`
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

func (r *Resource) GetParent() Parent {
	return r.Parent
}

func (r *Resource) SetParent(parent Parent) {
	r.Parent = parent
}
