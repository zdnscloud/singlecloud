package schema

import (
	"fmt"
	"io/ioutil"
	"net/http"

	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
)

type SchemaManager struct {
	schemas []*VersionedSchemas
}

var _ resource.SchemaManager = &SchemaManager{}

func NewSchemaManager() *SchemaManager {
	return &SchemaManager{}
}

func (m *SchemaManager) MustImport(v *resource.APIVersion, kind resource.ResourceKind, handler interface{}) {
	if err := m.Import(v, kind, handler); err != nil {
		panic("!!! import get err " + err.Error())
	}
}

func (m *SchemaManager) Import(v *resource.APIVersion, kind resource.ResourceKind, handler interface{}) error {
	handler_, err := resource.HandlerAdaptor(handler)
	if err != nil {
		return err
	}

	vs := m.getVersionedSchemas(v)
	if vs == nil {
		vs = NewVersionedSchemas(v)
		m.schemas = append(m.schemas, vs)
	}
	return vs.Import(kind, handler_)
}

func (m *SchemaManager) getVersionedSchemas(v *resource.APIVersion) *VersionedSchemas {
	for _, vs := range m.schemas {
		if vs.VersionEquals(v) {
			return vs
		}
	}
	return nil
}

func (m *SchemaManager) CreateResourceFromRequest(req *http.Request) (resource.Resource, *goresterr.APIError) {
	path := multiSlashRegexp.ReplaceAllString(req.URL.EscapedPath(), "/")
	var action string
	if req.Method == http.MethodPost {
		action = req.URL.Query().Get("action")
	}

	var body []byte
	if (req.Method == http.MethodPost || req.Method == http.MethodPut) && req.Body != nil {
		var err error
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, goresterr.NewAPIError(goresterr.InvalidBodyContent,
				fmt.Sprintf("failed to read request body: %s", err.Error()))
		}
		defer req.Body.Close()
	}

	for _, vs := range m.schemas {
		if r, err := vs.CreateResourceFromRequest(req.Method, path, body, action); err != nil {
			return nil, err
		} else if r != nil {
			return r, err
		}
	}
	return nil, goresterr.NewAPIError(goresterr.NotFound, fmt.Sprintf("%s has unknown api version", req.URL.Path))
}

func (m *SchemaManager) GetSchema(v *resource.APIVersion, kind resource.ResourceKind) resource.Schema {
	if vs := m.getVersionedSchemas(v); vs != nil {
		return vs.GetSchema(kind)
	}
	return nil
}

func (m *SchemaManager) GenerateResourceRoute() resource.ResourceRoute {
	route := resource.NewResourceRoute()
	for _, vs := range m.schemas {
		route = route.Merge(vs.GenerateResourceRoute())
	}
	return route
}
