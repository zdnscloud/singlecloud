package types

import (
	"bytes"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/util"
)

const GroupPrefix = "/apis"

type Schemas struct {
	typeNames        map[reflect.Type]string
	schemasByVersion map[string]map[string]*Schema
	versions         []APIVersion
	schemas          []*Schema
	errors           []error
}

func NewSchemas() *Schemas {
	return &Schemas{
		typeNames:        map[reflect.Type]string{},
		schemasByVersion: map[string]map[string]*Schema{},
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

	schemas, ok := s.schemasByVersion[schema.Version.Version]
	if !ok {
		schemas = map[string]*Schema{}
		s.schemasByVersion[schema.Version.Version] = schemas
		s.versions = append(s.versions, schema.Version)
	}

	if _, ok := schemas[schema.PluralName]; !ok {
		schemas[schema.PluralName] = &schema
		s.schemas = append(s.schemas, &schema)
	}

	return s
}

func (s *Schemas) setupDefaults(schema *Schema) {
	if schema.GetType() == "" {
		s.errors = append(s.errors, fmt.Errorf("get type from schema failed: %v", schema))
		return
	}
	if schema.Version.Version == "" {
		s.errors = append(s.errors, fmt.Errorf("version is not set on schema: %s", schema.GetType()))
		return
	}
	if schema.PluralName == "" {
		schema.PluralName = util.GuessPluralName(schema.GetType())
	}
}

func (s *Schemas) Versions() []APIVersion {
	return s.versions
}

func (s *Schemas) Schemas() []*Schema {
	return s.schemas
}

func (s *Schemas) Schema(version *APIVersion, name string) *Schema {
	schemas, ok := s.schemasByVersion[version.Version]
	if !ok {
		return nil
	}

	schema := schemas[name]
	if schema != nil {
		return schema
	}

	for _, check := range schemas {
		if strings.EqualFold(check.GetType(), name) || strings.EqualFold(check.PluralName, name) {
			return check
		}
	}

	return nil
}

func (s *Schemas) UrlMethods() map[string][]string {
	urlMethods := make(map[string][]string)
	for _, schema := range s.schemas {
		var parents []string
		for parent := schema.Parent; parent != ""; {
			if parentSchema := s.Schema(&schema.Version, util.GuessPluralName(parent)); parentSchema != nil {
				parents = append(parents, parent)
				parent = parentSchema.Parent
			} else {
				panic(fmt.Sprintf("schema %v is non-exists", parent))
			}
		}

		buffer := bytes.Buffer{}
		for i := len(parents) - 1; i >= 0; i-- {
			buffer.WriteString("/")
			buffer.WriteString(util.GuessPluralName(parents[i]))
			buffer.WriteString("/:")
			buffer.WriteString(parents[i])
			buffer.WriteString("_id")
		}

		parentUrl := buffer.String()
		url := path.Join(schema.Version.GetVersionURL(), parentUrl, schema.PluralName)
		if len(schema.CollectionMethods) != 0 {
			urlMethods[url] = schema.CollectionMethods
		}

		if len(schema.ResourceMethods) != 0 {
			urlMethods[path.Join(url, ":"+schema.GetType()+"_id")] = schema.ResourceMethods
		}
	}

	return urlMethods
}

func (s *Schemas) GetChildren(parent string) []string {
	if parent == "" {
		return nil
	}

	var children []string
	for _, schema := range s.schemas {
		if schema.Parent == parent {
			children = append(children, schema.PluralName)
		}
	}

	return children
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
