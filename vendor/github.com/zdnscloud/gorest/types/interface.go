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

type Handler interface {
	Create(Object, []byte) (interface{}, *APIError)
	Delete(Object) *APIError
	Update(Object) (interface{}, *APIError)
	List(Object) interface{}
	Get(Object) interface{}
	Action(Object, string, map[string]interface{}) (interface{}, *APIError)
}
