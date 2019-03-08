package types

import (
	"bytes"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/util/name"
)

type Schemas struct {
	typeNames     map[reflect.Type]string
	schemasByPath map[string]map[string]*Schema
	versions      []APIVersion
	schemas       []*Schema
	errors        []error
}

func NewSchemas() *Schemas {
	return &Schemas{
		typeNames:     map[reflect.Type]string{},
		schemasByPath: map[string]map[string]*Schema{},
	}
}

func (s *Schemas) Err() error {
	return NewErrors(s.errors...)
}

func (s *Schemas) AddSchemas(schema *Schemas) *Schemas {
	for _, schema := range schema.Schemas() {
		s.AddSchema(*schema)
	}
	return s
}

func (s *Schemas) AddSchema(schema Schema) *Schemas {
	s.setupDefaults(&schema)

	schemas, ok := s.schemasByPath[schema.Version.Path]
	if !ok {
		schemas = map[string]*Schema{}
		s.schemasByPath[schema.Version.Path] = schemas
		s.versions = append(s.versions, schema.Version)
	}

	if _, ok := schemas[schema.ID]; !ok {
		schemas[schema.ID] = &schema
		s.schemas = append(s.schemas, &schema)
	}

	return s
}

func (s *Schemas) setupDefaults(schema *Schema) {
	if schema.ID == "" {
		s.errors = append(s.errors, fmt.Errorf("ID is not set on schema: %v", schema))
		return
	}
	if schema.Version.Path == "" || schema.Version.Version == "" {
		s.errors = append(s.errors, fmt.Errorf("version is not set on schema: %s", schema.ID))
		return
	}
	if schema.PluralName == "" {
		schema.PluralName = name.GuessPluralName(schema.ID)
	}
}

func (s *Schemas) Versions() []APIVersion {
	return s.versions
}

func (s *Schemas) Schemas() []*Schema {
	return s.schemas
}

func (s *Schemas) Schema(version *APIVersion, name string) *Schema {
	schemas, ok := s.schemasByPath[version.Path]
	if !ok {
		return nil
	}

	schema := schemas[name]
	if schema != nil {
		return schema
	}

	for _, check := range schemas {
		if strings.EqualFold(check.ID, name) || strings.EqualFold(check.PluralName, name) {
			return check
		}
	}

	return nil
}

func (s *Schemas) UrlMethods() map[string][]string {
	urlMethods := make(map[string][]string)
	for _, schema := range s.schemas {
		url := path.Join("/"+schema.Version.Group, schema.Version.Path)
		var parents []string
		for parent := schema.Parent; parent != ""; {
			if parentSchema := s.Schema(&schema.Version, parent); parentSchema != nil {
				parents = append(parents, parent)
				parent = parentSchema.Parent
			} else {
				panic(fmt.Sprintf("schema %v is non-exists", parent))
			}
		}

		buffer := bytes.Buffer{}
		for i := len(parents) - 1; i >= 0; i-- {
			buffer.WriteString("/")
			buffer.WriteString(name.GuessPluralName(parents[i]))
			buffer.WriteString("/:")
			buffer.WriteString(parents[i])
			buffer.WriteString("_id")
		}

		parentUrl := buffer.String()
		url = path.Join("/"+schema.Version.Group, schema.Version.Path, parentUrl, schema.PluralName)
		if len(schema.CollectionMethods) != 0 {
			urlMethods[url] = schema.CollectionMethods
		}

		if len(schema.ResourceMethods) != 0 {
			urlMethods[path.Join(url, ":"+schema.ID+"_id")] = schema.ResourceMethods
		}
	}

	return urlMethods
}

type MultiErrors struct {
	Errors []error
}

func NewErrors(inErrors ...error) error {
	var errors []error
	for _, err := range inErrors {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) == 0 {
		return nil
	} else if len(errors) == 1 {
		return errors[0]
	}
	return &MultiErrors{
		Errors: errors,
	}
}

func (m *MultiErrors) Error() string {
	buf := bytes.NewBuffer(nil)
	for _, err := range m.Errors {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(err.Error())
	}

	return buf.String()
}
