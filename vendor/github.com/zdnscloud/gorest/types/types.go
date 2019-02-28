package types

import (
	"reflect"
)

const (
	ResourceFieldID = "id"
)

type Collection struct {
	Type         string            `json:"type,omitempty"`
	CreateTypes  map[string]string `json:"createTypes,omitempty"`
	Actions      map[string]string `json:"actions"`
	ResourceType string            `json:"resourceType"`
}

type GenericCollection struct {
	Collection
	Data []interface{} `json:"data"`
}

type Resource struct {
	ID      string            `json:"id,omitempty"`
	Type    string            `json:"type,omitempty"`
	Actions map[string]string `json:"actions"`
}

type APIVersion struct {
	Group            string `json:"group,omitempty"`
	Version          string `json:"version,omitempty"`
	Path             string `json:"path,omitempty"`
	SubContext       bool   `json:"subContext,omitempty"`
	SubContextSchema string `json:"filterField,omitempty"`
}

type Namespaced struct{}

var NamespaceScope TypeScope = "namespace"

type TypeScope string

type Schema struct {
	ID                string            `json:"id,omitempty"`
	Embed             bool              `json:"embed,omitempty"`
	EmbedType         string            `json:"embedType,omitempty"`
	CodeName          string            `json:"-"`
	CodeNamePlural    string            `json:"-"`
	PkgName           string            `json:"-"`
	Type              string            `json:"type,omitempty"`
	BaseType          string            `json:"baseType,omitempty"`
	Version           APIVersion        `json:"version"`
	PluralName        string            `json:"pluralName,omitempty"`
	ResourceMethods   []string          `json:"resourceMethods,omitempty"`
	ResourceFields    map[string]Field  `json:"resourceFields"`
	ResourceActions   map[string]Action `json:"resourceActions,omitempty"`
	CollectionMethods []string          `json:"collectionMethods,omitempty"`
	CollectionFields  map[string]Field  `json:"collectionFields,omitempty"`
	CollectionActions map[string]Action `json:"collectionActions,omitempty"`
	Scope             TypeScope         `json:"-"`

	InternalSchema      *Schema             `json:"-"`
	Mapper              Mapper              `json:"-"`
	ActionHandler       ActionHandler       `json:"-"`
	ListHandler         RequestHandler      `json:"-"`
	CreateHandler       RequestHandler      `json:"-"`
	DeleteHandler       RequestHandler      `json:"-"`
	UpdateHandler       RequestHandler      `json:"-"`
	CollectionFormatter CollectionFormatter `json:"-"`
	ErrorHandler        ErrorHandler        `json:"-"`
	StructVal           reflect.Value       `json:"-"`
	Handler             Handler             `json:"-"`
	Parent              Parent              `json:"-"`
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

func (c *Collection) AddAction(apiContext *APIContext, name string) {
	c.Actions[name] = apiContext.URLBuilder.CollectionAction(apiContext.Schema, nil, name)
}

type Object interface {
	ObjectID
	ObjectType
	ObjectParent
}

type ObjectParent interface {
	GetParent() Parent
	SetParent(Parent)
}

type ObjectID interface {
	GetID() string
	SetID(string)
}

type ObjectType interface {
	GetType() string
	SetType(string)
}

type Handler interface {
	Create(Object) (interface{}, error)
	Delete(Object) error
	BatchDelete(Object) error
	Update(ObjectType, ObjectID, Object) (interface{}, error)
	List(Object) interface{}
	Get(Object) interface{}
	Action(Object, string, map[string]interface{}) (interface{}, error)
}
