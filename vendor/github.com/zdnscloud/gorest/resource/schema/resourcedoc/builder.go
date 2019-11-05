package resourcedoc

import (
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/util"
)

type Resource struct {
	Top map[string][]Field
	Sub []map[string][]Field
}

type Field struct {
	Name    string
	Type    reflect.Type
	Kind    util.Kind
	Tag     reflect.StructTag
	Special string
}

type Builder struct {
	resource *Resource
}

func NewBuilder() *Builder {
	return &Builder{
		resource: &Resource{
			Top: make(map[string][]Field),
			Sub: make([]map[string][]Field, 0),
		},
	}
}

func (b *Builder) GetTop() map[string][]Field {
	return b.resource.Top
}

func (b *Builder) GetSub() []map[string][]Field {
	return b.resource.Sub
}

func (b *Builder) BuildResource(name string, t reflect.Type) {
	b.resource.Top[name] = b.buildFields(name, t)
	return
}

func (b *Builder) buildFields(name string, t reflect.Type) []Field {
	var fields []Field
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		typ := t.Field(i).Type
		tag := t.Field(i).Tag
		if strings.Contains(name, "ResourceBase") || fieldJsonName(name, tag.Get("json")) == "-" {
			continue
		}

		field := Field{
			Name:    name,
			Type:    typ,
			Kind:    util.Inspect(typ),
			Tag:     tag,
			Special: specialType(typ.Name()),
		}
		fields = append(fields, field)

		if field.Special == "" {
			b.handleField(name, typ)
		}
	}
	return fields
}

func (b *Builder) handleField(name string, t reflect.Type) {
	kind := util.Inspect(t)
	switch kind {
	case util.StringStructPtrMap, util.StructPtrSlice:
		b.handleField(name, t.Elem().Elem())
	case util.StringStructMap, util.StructPtr, util.StructSlice:
		b.handleField(name, t.Elem())
	case util.Struct:
		b.structHandle(t.Name(), t)
	}
}

func (b *Builder) structHandle(name string, t reflect.Type) {
	resource := make(map[string][]Field)
	resource[name] = b.buildFields(name, t)
	if !b.isExist(name) {
		b.resource.Sub = append(b.resource.Sub, resource)
	}
}

func (b *Builder) isExist(name string) bool {
	for _, s := range b.resource.Sub {
		for n, _ := range s {
			if n == name {
				return true
			}
		}
	}
	return false
}

func specialType(typ string) string {
	switch typ {
	case "RawMessage":
		return "json"
	case "ISOTime":
		return "date"
	default:
		return ""
	}
}
