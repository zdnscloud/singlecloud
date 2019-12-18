package resourcefield

import (
	"fmt"
	"reflect"

	"github.com/zdnscloud/gorest/resource/schema/resourcefield/validator"
)

type Field interface {
	JsonName() string
	Name() string

	IsRequired() bool
	SetRequired(bool)

	//validate fields of go struct
	//resource should be unmarshalled from raw
	Validate(resource interface{}, raw map[string]interface{}) error
}

var _ Field = &leafField{}
var _ Field = &structField{}
var _ Field = &sliceLeafField{}
var _ Field = &sliceStructField{}
var _ Field = &mapLeafField{}
var _ Field = &mapStructField{}

//field with type, int, string, boolean
type leafField struct {
	name       string
	jsonName   string
	kind       reflect.Kind
	required   bool
	validators []validator.Validator
}

func newLeafField(name, jsonName string, kind reflect.Kind) *leafField {
	return &leafField{
		name:     name,
		jsonName: jsonName,
		kind:     kind,
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

func (f *leafField) SetValidators(validators []validator.Validator) {
	f.validators = validators
}

func (f *leafField) Validate(val interface{}, raw map[string]interface{}) error {
	if _, ok := raw[f.JsonName()]; !ok {
		if f.IsRequired() {
			return fmt.Errorf("field %s is missing", f.jsonName)
		} else {
			return nil
		}
	}

	if reflect.ValueOf(val).Kind() != f.kind {
		return fmt.Errorf("field %s has invalid invalid kind", f.jsonName)
	}

	return f.doValidate(val)
}

func (f *leafField) doValidate(val interface{}) error {
	for _, validator := range f.validators {
		if err := validator.Validate(val); err != nil {
			return err
		}
	}
	return nil
}

type sliceLeafField struct {
	*leafField
}

func newSliceLeafField(inner *leafField) *sliceLeafField {
	return &sliceLeafField{
		leafField: inner,
	}
}

func (f *sliceLeafField) Validate(val interface{}, raw map[string]interface{}) error {
	specified, _, err := fieldIsSpecifiedWithKind(f.leafField, raw, reflect.Slice)
	if err != nil {
		return err
	}

	value := reflect.ValueOf(val)
	if value.Kind() != reflect.Slice {
		return fmt.Errorf("runtime value of %s isn't synchronize with json data", f.leafField.JsonName())
	}
	if specified {
		for i := 0; i < value.Len(); i++ {
			if err := f.leafField.doValidate(value.Index(i).Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func fieldIsSpecifiedWithKind(f Field, raw map[string]interface{}, kind reflect.Kind) (bool, interface{}, error) {
	jsonVal, specified := raw[f.JsonName()]
	//handle set direct name to nil, which is same with not speicified
	if jsonVal == nil {
		specified = false
	}

	if f.IsRequired() {
		if !specified {
			return specified, nil, fmt.Errorf("field %s is missing", f.JsonName())
		}
	}

	if specified {
		v := reflect.ValueOf(jsonVal)
		if !v.IsValid() {
			return specified, nil, fmt.Errorf("field %s has invalid value", f.JsonName())
		}

		if v.Kind() != kind {
			return specified, nil, fmt.Errorf("field %s isn't %v", f.JsonName(), kind)
		}

		if v.Len() == 0 && f.IsRequired() {
			return specified, nil, fmt.Errorf("field %s with empty slice ", f.JsonName())
		}
	}
	return specified, jsonVal, nil
}

type sliceStructField struct {
	Field
	inner Field
}

func newSliceStructField(self, inner Field) *sliceStructField {
	return &sliceStructField{
		Field: self,
		inner: inner,
	}
}

func (f *sliceStructField) Validate(val interface{}, raw map[string]interface{}) error {
	specified, jsonVal, err := fieldIsSpecifiedWithKind(f.Field, raw, reflect.Slice)
	if err != nil {
		return err
	}

	if !specified || f.inner == nil {
		return nil
	}

	value := reflect.ValueOf(val)
	jsonValue := reflect.ValueOf(jsonVal)
	if value.Kind() != reflect.Slice || value.Len() != jsonValue.Len() {
		return fmt.Errorf("runtime value of %s isn't synchronize with json data", f.Field.JsonName())
	}

	for i := 0; i < value.Len(); i++ {
		elemVal := jsonValue.Index(i).Interface()
		elemRaw, ok := elemVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("elem of field %s is not a struct", f.Field.JsonName())
		}
		if err := f.inner.Validate(value.Index(i).Interface(), elemRaw); err != nil {
			return err
		}
	}

	return nil
}

type mapLeafField struct {
	*leafField
}

func newMapLeafField(inner *leafField) *mapLeafField {
	return &mapLeafField{
		leafField: inner,
	}
}

func (f *mapLeafField) Validate(val interface{}, raw map[string]interface{}) error {
	specified, _, err := fieldIsSpecifiedWithKind(f.leafField, raw, reflect.Map)
	if err != nil {
		return err
	}
	if !specified {
		return nil
	}

	value := reflect.ValueOf(val)
	if value.Kind() != reflect.Map {
		return fmt.Errorf("runtime value of %s isn't synchronize with json data", f.leafField.JsonName())
	}
	iter := value.MapRange()
	for iter.Next() {
		if err := f.leafField.doValidate(iter.Value().Interface()); err != nil {
			return err
		}
	}
	return nil
}

type mapStructField struct {
	Field
	inner Field
}

func newMapStructField(self, inner Field) *mapStructField {
	return &mapStructField{
		Field: self,
		inner: inner,
	}
}

func (f *mapStructField) Validate(val interface{}, raw map[string]interface{}) error {
	specified, jsonVal, err := fieldIsSpecifiedWithKind(f.Field, raw, reflect.Map)
	if err != nil {
		return err
	}

	if !specified || f.inner == nil {
		return nil
	}

	jsonValue := reflect.ValueOf(jsonVal)
	value := reflect.ValueOf(val)
	if value.Kind() != reflect.Map || jsonValue.Len() != value.Len() {
		return fmt.Errorf("runtime value of %s isn't synchronize with json data", f.Field.JsonName())
	}

	ji := jsonValue.MapRange()
	vi := value.MapRange()
	for ji.Next() && vi.Next() {
		elemRaw, ok := (ji.Value().Interface()).(map[string]interface{})
		if !ok {
			return fmt.Errorf("value of field %s is not a struct", f.Field.JsonName())
		}
		if err := f.inner.Validate(vi.Value().Interface(), elemRaw); err != nil {
			return err
		}
	}
	return nil
}

type structField struct {
	Field
	fields map[string]Field
}

func newStructField(self Field, fields map[string]Field) *structField {
	return &structField{
		Field:  self,
		fields: fields,
	}
}

func (f *structField) Validate(val interface{}, raw map[string]interface{}) error {
	value := reflect.ValueOf(val)
	//only handle one level redirect
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("struct field with non-sturct but %v", value.Kind())
	}

	//this is a nest struct
	if f.Field != nil {
		jsonName := f.Field.JsonName()
		jsonVal, ok := raw[jsonName]
		if f.Field.IsRequired() && !ok {
			return fmt.Errorf("struct field %s is missing", jsonName)
		}
		if nr, ok := jsonVal.(map[string]interface{}); ok {
			raw = nr
		} else {
			return fmt.Errorf("value of field %s in json data is not a struct", jsonName)
		}
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		//un-exported field
		if ft.PkgPath != "" {
			continue
		}

		if ft.Anonymous {
			if err := f.Validate(value.Field(i).Interface(), raw); err != nil {
				return err
			}
			continue
		}

		if field, ok := f.fields[ft.Name]; ok {
			if err := field.Validate(value.Field(i).Interface(), raw); err != nil {
				return err
			}
		}
	}
	return nil
}
