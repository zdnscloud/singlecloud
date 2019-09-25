package resourcefield

import (
	"fmt"
	"reflect"
)

type ResourceField interface {
	Validate(interface{}) error
	CheckRequired(raw map[string]interface{}) error
}

func New(typ reflect.Type) (ResourceField, error) {
	builder := NewBuilder()
	if rf, err := builder.Build(typ); err != nil {
		return nil, err
	} else if rf == nil {
		return nil, nil
	} else {
		return rf, nil
	}
}

type resourceField struct {
	fields map[string]Field
}

//validate the resource go struct
func (f *resourceField) Validate(value interface{}) error {
	fieldValue := reflect.ValueOf(value)
	switch fieldValue.Kind() {
	case reflect.Ptr:
		if fieldValue.IsNil() {
			return nil
		}

		if fieldValue.Elem().Kind() == reflect.Struct {
			return f.validateStruct(fieldValue.Elem())
		}
	case reflect.Struct:
		return f.validateStruct(fieldValue)
	}
	return fmt.Errorf("struct field doesn't support type %v", fieldValue.Kind())
}

func (f *resourceField) validateStruct(value reflect.Value) error {
	st := value.Type()
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		if sf.Anonymous {
			if err := f.validateStruct(value.Field(i)); err != nil {
				return err
			}
			continue
		}

		if field, ok := f.fields[sf.Name]; ok {
			if err := field.Validate(value.Field(i).Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

//check the json string whether the required field is specified
func (f *resourceField) CheckRequired(raw map[string]interface{}) error {
	for _, field := range f.fields {
		if err := field.CheckRequired(raw); err != nil {
			return err
		}
	}
	return nil
}

func newResourceField(fields []Field) *resourceField {
	fields_ := make(map[string]Field)
	for _, field := range fields {
		fields_[field.Name()] = field
	}

	return &resourceField{
		fields: fields_,
	}
}
