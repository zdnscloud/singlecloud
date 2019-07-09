package util

import (
	"reflect"
)

func IsValueNil(i interface{}) bool {
	if i == nil {
		return true
	}

	value := reflect.ValueOf(i)
	valueKind := value.Kind()
	return (valueKind == reflect.Interface || valueKind == reflect.Ptr || valueKind == reflect.Map) && value.IsNil()
}
