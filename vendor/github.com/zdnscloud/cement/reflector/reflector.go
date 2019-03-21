package reflector

import (
	"errors"
	"reflect"
)

func GetStructFromPointer(v interface{}) (reflect.Value, bool) {
	structVal := reflect.ValueOf(v)
	if structVal.Kind() != reflect.Ptr {
		return structVal, false
	}

	structVal = reflect.Indirect(structVal)
	if structVal.Kind() != reflect.Struct {
		return structVal, false
	}

	return structVal, true
}

func NewSlicePointer(elemType reflect.Type) interface{} {
	slice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)
	pointer := reflect.New(slice.Type())
	pointer.Elem().Set(slice)
	return pointer.Interface()
}

func CloneStruct(s interface{}) (interface{}, error) {
	structVal, isStructPointer := GetStructFromPointer(s)
	if isStructPointer == false {
		return nil, errors.New("parameter isn't pointer of struct")
	}
	structCpyPointer := reflect.New(structVal.Type())
	structCpyPointer.Elem().Set(structVal)
	return structCpyPointer.Interface(), nil
}

func StructName(sp interface{}) (string, error) {
	v, isPointer := GetStructFromPointer(sp)
	if isPointer == false {
		return "", errors.New("parameter isn't pointer of struct")
	}

	return v.Type().Name(), nil
}

func UnwrapSingleElemSlice(rs interface{}) (interface{}, error) {
	if reflect.ValueOf(rs).Len() != 1 {
		return nil, errors.New("slice isn't single element")
	}

	v := reflect.ValueOf(rs).Index(0)
	p := reflect.New(v.Type())
	p.Elem().Set(v)
	return p.Interface(), nil
}

func GetStructInSlice(ss interface{}) (interface{}, error) {
	slice := reflect.Indirect(reflect.ValueOf(ss))
	if slice.Kind() != reflect.Slice {
		return "", errors.New("out should be a model slice pointer")
	}

	model := slice.Type().Elem()
	if model.Kind() != reflect.Struct {
		return "", errors.New("out should be a slice of structs")
	}
	return reflect.New(model).Interface(), nil
}
