package types

import (
	"time"
)

type Object interface {
	ObjectID
	ObjectType
	ObjectLinks
	ObjectTimestamp
	ObjectParent
	ObjectSchema
}

type ObjectParent interface {
	GetParent() Object
	SetParent(Object)
}

type ObjectID interface {
	GetID() string
	SetID(string)
}

type ObjectType interface {
	GetType() string
	SetType(string)
}

type ObjectLinks interface {
	GetLinks() map[string]string
	SetLinks(map[string]string)
}

type ObjectTimestamp interface {
	GetCreationTimestamp() time.Time
	SetCreationTimestamp(time.Time)
}

type ObjectSchema interface {
	GetSchema() *Schema
	SetSchema(*Schema)
}

type Handler interface {
	Create(*Context, []byte) (interface{}, *APIError)
	Delete(*Context) *APIError
	Update(*Context) (interface{}, *APIError)
	List(*Context) interface{}
	Get(*Context) interface{}
	Action(*Context) (interface{}, *APIError)
}
