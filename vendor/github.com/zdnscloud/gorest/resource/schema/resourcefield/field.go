package resourcefield

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Field interface {
	JsonName() string

	Name() string

	IsRequired() bool
	SetRequired(bool)

	//validate fields of go struct
	Validate(interface{}) error

	//work on json format string
	CheckRequired(json map[string]interface{}) error
}

var _ Field = &leafField{}
var _ Field = &compositeField{}

type leafField struct {
	name       string
	jsonName   string
	required   bool
	validators []Validator
}

func newLeafField(name, jsonName string) *leafField {
	return &leafField{
		name:     name,
		jsonName: jsonName,
		required: false,
	}
}

func (f *leafField) Name() string {
	return f.name
}

func (f *leafField) JsonName() string {
	return f.jsonName
}

func (f *leafField) IsRequired() bool {
	return f.required
}

func (f *leafField) SetRequired(required bool) {
	f.required = required
}

func (f *leafField) CheckRequired(raw map[string]interface{}) error {
	jsonName := f.JsonName()
	if f.IsRequired() {
		val, ok := raw[jsonName]
		if ok == false {
			return fmt.Errorf("field %s is missing", jsonName)
		}

		v := reflect.ValueOf(val)
		kind := v.Kind()
		if kind == reflect.String || kind == reflect.Map || kind == reflect.Slice {
			if v.Len() == 0 {
				return fmt.Errorf("field %s is missing", jsonName)
			}
		}
	}
	return nil
}

func (f *leafField) SetValidators(validators []Validator) {
	f.validators = validators
}

func (f *leafField) Validate(val interface{}) error {
	for _, validator := range f.validators {
		if err := validator.Validate(val); err != nil {
			return err
		}
	}
	return nil
}

//filed like map with struct as value type
//struct slice
//struct ptr or just struct
type OwnerKind string

const (
	OwnerNone      OwnerKind = "none"
	OwnerSlice     OwnerKind = "slice"
	OwnerStringMap OwnerKind = "string_map"
)

type compositeField struct {
	name      string
	jsonName  string
	required  bool
	ownerKind OwnerKind
	field     *resourceField
}

func newCompositeField(name, jsonName string, inner *resourceField) *compositeField {
	return &compositeField{
		name:      name,
		jsonName:  jsonName,
		required:  false,
		ownerKind: OwnerNone,
		field:     inner,
	}
}

func (f *compositeField) Name() string {
	return f.name
}

func (f *compositeField) JsonName() string {
	return f.jsonName
}

func (f *compositeField) IsRequired() bool {
	return f.required
}

func (f *compositeField) SetRequired(required bool) {
	f.required = required
}

func (f *compositeField) SetOwner(kind OwnerKind) {
	f.ownerKind = kind
}

func (f *compositeField) Validate(value interface{}) error {
	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Ptr {
		value := reflect.ValueOf(value)
		if !value.IsNil() {
			return f.Validate(value.Elem().Interface())
		} else {
			return nil
		}
	}

	switch f.ownerKind {
	case OwnerStringMap:
		if kind != reflect.Map {
			return fmt.Errorf("use map field to validate %v", kind)
		}
		iter := reflect.ValueOf(value).MapRange()
		for iter.Next() {
			if err := f.field.Validate(iter.Value().Interface()); err != nil {
				return err
			}
		}
	case OwnerSlice:
		if kind != reflect.Slice {
			return fmt.Errorf("use slice field to validate %v", kind)
		}
		fieldValue := reflect.ValueOf(value)
		for i := 0; i < fieldValue.Len(); i++ {
			//todo, is nil element valid?
			if err := f.field.Validate(fieldValue.Index(i).Interface()); err != nil {
				return err
			}
		}
	case OwnerNone:
		if kind != reflect.Struct {
			return fmt.Errorf("use struct field to validate %v", kind)
		}
		return f.field.Validate(value)
	}

	return nil
}

func (f *compositeField) CheckRequired(jd map[string]interface{}) error {
	if !f.IsRequired() {
		return nil
	}

	jsonName := f.JsonName()
	val, ok := jd[jsonName]
	if ok == false || val == nil {
		return fmt.Errorf("field %s is missing", jsonName)
	}

	//check empty slice and empty map
	v := reflect.ValueOf(val)
	kind := v.Kind()
	switch f.ownerKind {
	case OwnerStringMap:
		if kind != reflect.Map {
			return fmt.Errorf("use map field to validate %v", kind)
		}

		if v.Len() == 0 {
			return fmt.Errorf("field %s is missing", jsonName)
		}

		iter := v.MapRange()
		for iter.Next() {
			d, _ := json.Marshal(iter.Value().Interface())
			m := make(map[string]interface{})
			if err := json.Unmarshal(d, &m); err != nil {
				return fmt.Errorf("value of field %s is not a struct", jsonName)
			}
			if err := f.field.CheckRequired(m); err != nil {
				return err
			}
		}
	case OwnerSlice:
		if kind != reflect.Slice {
			return fmt.Errorf("use slice field to validate %v", kind)
		}

		if v.Len() == 0 {
			return fmt.Errorf("field %s is missing", jsonName)
		}

		for i := 0; i < v.Len(); i++ {
			d, _ := json.Marshal(v.Index(i).Interface())
			m := make(map[string]interface{})
			if err := json.Unmarshal(d, &m); err != nil {
				return fmt.Errorf("elem of field %s is not a struct", jsonName)
			}

			if err := f.field.CheckRequired(m); err != nil {
				return err
			}
		}
	case OwnerNone:
		if kind != reflect.Map {
			return fmt.Errorf("use struct field %s to validate %v", jsonName, kind)
		}
		d, _ := json.Marshal(val)
		m := make(map[string]interface{})
		if err := json.Unmarshal(d, &m); err != nil {
			return fmt.Errorf("field %s is not a struct", jsonName)
		}
		return f.field.CheckRequired(m)
	}
	return nil
}
