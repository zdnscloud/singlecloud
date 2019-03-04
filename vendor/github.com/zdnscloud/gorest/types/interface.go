package types

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
	Create(Object, []byte) (interface{}, *APIError)
	Delete(Object) *APIError
	Update(Object) (interface{}, *APIError)
	List(Object) interface{}
	Get(Object) interface{}
	Action(Object, string, map[string]interface{}) (interface{}, *APIError)
}
