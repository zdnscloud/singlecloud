package resourcefield

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/util"
)

type FieldBuilder struct {
	fields []Field
}

func NewBuilder() *FieldBuilder {
	return &FieldBuilder{}
}

func (b *FieldBuilder) Build(typ reflect.Type) (*resourceField, error) {
	if err := b.buildFields(typ); err != nil {
		return nil, err
	}

	if len(b.fields) == 0 {
		return nil, nil
	}

	return newResourceField(b.fields), nil
}

func (b *FieldBuilder) buildFields(typ reflect.Type) error {
	if typ.Kind() == reflect.Ptr {
		return b.buildFields(typ.Elem())
	}

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("build fields on non-struct type")
	}

	for i := 0; i < typ.NumField(); i++ {
		if err := b.buildField(typ.Field(i)); err != nil {
			return err
		}
	}

	return nil
}

func (b *FieldBuilder) buildField(sf reflect.StructField) error {
	if sf.PkgPath != "" {
		return nil
	}

	//embed struct
	if sf.Anonymous {
		return b.buildFields(sf.Type)
	}

	field, err := b.createField(sf.Name, sf.Type, sf.Tag.Get("json"), sf.Tag.Get("rest"))
	if err != nil {
		return err
	}

	if field != nil {
		return b.addField(field)
	}
	return nil
}

func (b *FieldBuilder) createField(name string, typ reflect.Type, json, rest string) (Field, error) {
	kind := util.Inspect(typ)
	switch kind {
	case util.Uint, util.Int, util.String, util.Bool, util.StringIntMap, util.StringStringMap, util.StringUintMap, util.IntSlice, util.UintSlice, util.StringSlice:
		if rest == "" {
			return nil, nil
		}
		if restTags := strings.Split(rest, ","); len(restTags) > 0 {
			return b.buildLeafField(name, typ, json, restTags)
		}
	case util.StructPtr:
		return b.createField(name, typ.Elem(), json, rest)
	case util.StringStructMap, util.StringStructPtrMap, util.StructSlice, util.StructPtrSlice:
		nestType := typ.Elem()
		if kind == util.StringStructPtrMap || kind == util.StructPtrSlice {
			nestType = nestType.Elem()
		}
		field, err := b.createField(name, nestType, json, rest)
		if err == nil && field != nil {
			if kind == util.StringStructMap || kind == util.StringStructPtrMap {
				field.(*compositeField).SetOwner(OwnerStringMap)
				return field, nil
			} else {
				field.(*compositeField).SetOwner(OwnerSlice)
				return field, nil
			}
		}
		return nil, err
	case util.Struct:
		nb := NewBuilder()
		sf, err := nb.Build(typ)
		if err != nil {
			return nil, err
		}
		if sf != nil {
			return b.buildCompositeField(name, typ.Kind(), json, sf, strings.Split(rest, ","))
		}
	}
	return nil, nil
}

func (b *FieldBuilder) buildLeafField(name string, typ reflect.Type, json string, restTags []string) (Field, error) {
	v, err := buildValidator(typ, restTags)
	if err != nil {
		return nil, err
	}
	field := newLeafField(name, fieldJsonName(name, json))
	if v != nil {
		field.SetValidators([]Validator{v})
	}
	if err := fieldParseOptional(field, typ.Kind(), restTags); err != nil {
		return nil, err
	}
	return field, nil
}

func (b *FieldBuilder) buildCompositeField(name string, kind reflect.Kind, json string, sf *resourceField, restTags []string) (Field, error) {
	field := newCompositeField(name, fieldJsonName(name, json), sf)
	if err := fieldParseOptional(field, kind, restTags); err != nil {
		return nil, err
	}
	return field, nil
}

func (b *FieldBuilder) addField(field Field) error {
	for _, old := range b.fields {
		if old.Name() == field.Name() {
			return fmt.Errorf("duplicate field name %s", field.Name())
		}
	}
	b.fields = append(b.fields, field)
	return nil
}

func fieldJsonName(name, jsonTag string) string {
	if jsonTag != "" {
		tags := strings.Split(jsonTag, ",")
		for _, tag := range tags {
			if tag != "omitempty" {
				return tag
			}
		}
	}

	return name
}
