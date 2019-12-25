package resourcefield

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/resource/schema/resourcefield/validator"
	"github.com/zdnscloud/gorest/util"
)

type FieldBuilder struct {
	fields []Field
}

func NewBuilder() *FieldBuilder {
	return &FieldBuilder{}
}

func (b *FieldBuilder) Build(typ reflect.Type) (*structField, error) {
	if err := b.buildFields(typ); err != nil {
		return nil, err
	}

	if len(b.fields) == 0 {
		return nil, nil
	}

	fields := make(map[string]Field)
	for _, field := range b.fields {
		fields[field.Name()] = field
	}

	return newStructField(nil, fields), nil
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
	case util.Uint, util.Int, util.String, util.Bool:
		if rest == "" {
			return nil, nil
		}
		if restTags := strings.Split(rest, ","); len(restTags) > 0 {
			return b.buildLeafField(name, typ, json, restTags)
		}
	case util.StringIntMap, util.StringStringMap, util.StringUintMap, util.IntSlice, util.UintSlice, util.StringSlice:
		if rest == "" {
			return nil, nil
		}

		restTags := strings.Split(rest, ",")
		if len(restTags) == 0 {
			return nil, nil
		}

		f, err := b.buildLeafField(name, typ, json, restTags)
		if err != nil {
			return nil, err
		}

		if kind == util.IntSlice || kind == util.UintSlice || kind == util.StringSlice {
			return newSliceLeafField(f), nil
		} else {
			return newMapLeafField(f), nil
		}
	case util.StructPtr:
		return b.createField(name, typ.Elem(), json, rest)
	case util.StringStructMap, util.StringStructPtrMap, util.StructSlice, util.StructPtrSlice:
		var self Field
		var err error
		if rest != "" {
			self, err = b.buildLeafField(name, typ, json, strings.Split(rest, ","))
			if err != nil {
				return nil, err
			}
		}

		nestType := typ.Elem()
		if kind == util.StringStructPtrMap || kind == util.StructPtrSlice {
			nestType = nestType.Elem()
		}

		inner, err := NewBuilder().Build(nestType)
		if err != nil {
			return nil, err
		}

		if inner == nil && self == nil {
			return nil, nil
		}

		if self == nil {
			self, _ = b.buildLeafField(name, typ, json, nil)
		}

		if kind == util.StringStructMap || kind == util.StringStructPtrMap {
			return newMapStructField(self, inner), nil
		} else {
			return newSliceStructField(self, inner), nil
		}
	case util.Struct:
		sf, err := NewBuilder().Build(typ)
		if err != nil {
			return nil, err
		}

		if sf != nil {
			self := newLeafField(name, fieldJsonName(name, json), typ.Kind())
			if err := fieldParseOptional(self, typ.Kind(), strings.Split(rest, ",")); err != nil {
				return nil, err
			}
			sf.Field = self
			return sf, nil
		}
	}
	return nil, nil
}

func (b *FieldBuilder) buildLeafField(name string, typ reflect.Type, json string, restTags []string) (*leafField, error) {
	v, err := validator.Build(typ, restTags)
	if err != nil {
		return nil, err
	}
	field := newLeafField(name, fieldJsonName(name, json), typ.Kind())
	if len(v) > 0 {
		field.SetValidators(v)
	}
	if err := fieldParseOptional(field, typ.Kind(), restTags); err != nil {
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
