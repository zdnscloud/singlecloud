package rstore

import (
	"reflect"

	"cement/reflector"
	"cement/stringtool"
)

type ResourceType string

type Resource interface {
	Validate() error
}

func GetResourceType(r Resource) ResourceType {
	n, _ := reflector.StructName(r)
	return ResourceType(stringtool.ToSnake(n))
}

func ResourceID(r Resource) string {
	v := reflect.ValueOf(r).Elem().FieldByName("Id")
	if v.IsValid() {
		return v.String()
	} else {
		return ""
	}
}

func SetResourceID(r Resource, id string) {
	v := reflect.ValueOf(r).Elem().FieldByName("Id")
	if v.IsValid() {
		v.SetString(id)
	}
}
