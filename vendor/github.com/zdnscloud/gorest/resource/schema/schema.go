package schema

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"

	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/gorest/resource/schema/resourcedoc"
	"github.com/zdnscloud/gorest/resource/schema/resourcefield"
)

type Schema struct {
	version          *resource.APIVersion
	fields           resourcefield.ResourceField
	actions          []resource.Action
	handler          resource.Handler
	resourceKind     resource.ResourceKind
	resourceName     string
	resourceKindName string
	children         []*Schema
}

func NewSchema(version *resource.APIVersion, kind resource.ResourceKind, handler resource.Handler) (*Schema, error) {
	if reflect.ValueOf(kind).Kind() == reflect.Ptr {
		return nil, fmt.Errorf("resource kind cannot be a pointer")
	}
	gt := reflect.TypeOf(kind)
	if _, ok := reflect.New(gt).Interface().(resource.Resource); ok == false {
		return nil, fmt.Errorf("resource type doesn't implement resource interface")
	}

	fields, err := resourcefield.New(reflect.TypeOf(kind))
	if err != nil {
		return nil, err
	}

	return &Schema{
		version:          version,
		fields:           fields,
		handler:          handler,
		resourceKind:     kind,
		resourceName:     resource.DefaultResourceName(kind),
		resourceKindName: resource.DefaultKindName(kind),
	}, nil
}

func (s *Schema) Equal(other *Schema) bool {
	return s.resourceName == other.resourceName
}

func (s *Schema) ResourceKindName() string {
	return s.resourceKindName
}

func (s *Schema) ResourceName() string {
	return s.resourceName
}

func (s *Schema) GetChildren() []*Schema {
	return s.children
}

func (s *Schema) CreateResourceFromPathSegments(parent resource.Resource, segments []string, method, action string, body []byte) (resource.Resource, *goresterr.APIError) {
	segmentCount := len(segments)
	if segmentCount == 0 {
		return parent, nil
	}

	if segments[0] != s.resourceName {
		return nil, nil
	}

	r := s.resourceKind.CreateDefaultResource()
	if r == nil {
		r = reflect.New(reflect.TypeOf(s.resourceKind)).Interface().(resource.Resource)
	}

	r.SetSchema(s)
	if parent != nil {
		r.SetParent(parent)
	}

	r.SetType(resource.DefaultKindName(s.resourceKind))
	if segmentCount > 1 {
		r.SetID(segments[1])
	}
	if segmentCount <= 2 {
		if err := s.validateAndFillResource(r, method, action, body); err != nil {
			return nil, err
		} else {
			return r, nil
		}
	}

	for _, child := range s.children {
		if r, err := child.CreateResourceFromPathSegments(r, segments[2:], method, action, body); err != nil {
			return nil, err
		} else if r != nil {
			return r, nil
		}
	}
	return nil, goresterr.NewAPIError(goresterr.NotFound,
		fmt.Sprintf("%s is not a child of %s", segments[2], s.resourceName))
}

func (s *Schema) validateAndFillResource(r resource.Resource, method, action string, body []byte) *goresterr.APIError {
	if method == http.MethodPost && action != "" {
		if action_, err := s.parseAction(action, body); err != nil {
			return err
		} else {
			r.SetAction(action_)
		}
	} else if method == http.MethodPost || method == http.MethodPut {
		//fields could be nil, when no any rest rag is specified in struct field
		if s.fields != nil {
			objMap := make(map[string]interface{})
			if body != nil {
				if err := json.Unmarshal(body, &objMap); err != nil {
					return goresterr.NewAPIError(goresterr.InvalidBodyContent, fmt.Sprintf("request body isn't a string map:%s", err.Error()))
				}
			}
			if err := s.fields.CheckRequired(objMap); err != nil {
				return goresterr.NewAPIError(goresterr.InvalidBodyContent, err.Error())
			}
		}
		if body != nil {
			json.Unmarshal(body, r)
		}
		if s.fields != nil {
			if err := s.fields.Validate(r); err != nil {
				return goresterr.NewAPIError(goresterr.InvalidBodyContent, err.Error())
			}
		}
	}
	return nil
}

func (s *Schema) parseAction(name string, body []byte) (*resource.Action, *goresterr.APIError) {
	if s.handler.GetActionHandler() == nil {
		return nil, goresterr.NewAPIError(goresterr.NotFound,
			fmt.Sprintf("no handler for action %s", name))
	}

	if action := s.resourceKind.CreateAction(name); action != nil {
		if action.Input != nil {
			if err := json.Unmarshal(body, action.Input); err != nil {
				return nil, goresterr.NewAPIError(goresterr.InvalidBodyContent,
					fmt.Sprintf("failed to parse action params: %s", err.Error()))
			}
		}
		return action, nil
	} else {
		return nil, goresterr.NewAPIError(goresterr.NotFound,
			fmt.Sprintf("unknown action %s", name))
	}
}

func (s *Schema) AddChild(child *Schema) error {
	for _, c := range s.children {
		if c.Equal(child) {
			return fmt.Errorf("duplicate import resource kind %s", child.ResourceKindName())
		}
	}
	s.children = append(s.children, child)
	return nil
}

func (s *Schema) GetSchema(kind resource.ResourceKind) *Schema {
	if s.resourceKindName == resource.DefaultKindName(kind) {
		return s
	}

	for _, c := range s.children {
		if target := c.GetSchema(kind); target != nil {
			return target
		}
	}

	return nil
}

func (s *Schema) GetHandler() resource.Handler {
	return s.handler
}

func (s *Schema) GenerateResourceRoute(parents []*Schema) resource.ResourceRoute {
	route := s.generateSelfRoute(parents)
	for _, child := range s.children {
		route = route.Merge(child.GenerateResourceRoute(append(parents, s)))
	}
	return route
}

func (s *Schema) urlIdSegment() string {
	return ":" + s.resourceKindName + "_id"
}

func (s *Schema) generateSelfRoute(parents []*Schema) resource.ResourceRoute {
	collectionPath := s.generateCollectionPath(parents, nil, "")
	resourcePath := path.Join(collectionPath, s.urlIdSegment())
	route := resource.NewResourceRoute()
	for _, method := range resource.GetResourceMethods(s.handler) {
		route.AddPathForMethod(method, resourcePath)
	}
	for _, method := range resource.GetCollectionMethods(s.handler) {
		route.AddPathForMethod(method, collectionPath)
	}
	return route
}

func (s *Schema) generateCollectionPath(parents []*Schema, ids []string, httpSchemeAndHost string) string {
	segments := make([]string, 0, len(parents)*2+3)
	segments = append(segments, httpSchemeAndHost)
	segments = append(segments, s.version.GetUrl())
	for i, parent := range parents {
		segments = append(segments, parent.resourceName)
		if ids != nil {
			segments = append(segments, ids[i])
		} else {
			segments = append(segments, parent.urlIdSegment())
		}
	}
	segments = append(segments, s.resourceName)
	return (path.Join(segments...))
}

func (s *Schema) generateCollectionLink(r resource.Resource, httpSchemeAndHost string) (string, error) {
	var ss []*Schema
	var ids []string
	parent := r.GetParent()
	for parent != nil {
		if ps := parent.GetSchema().(*Schema); ps == nil {
			return "", fmt.Errorf("%s has no schmea", parent.GetType())
		} else {
			ss = append(ss, ps)
		}

		if pid := parent.GetID(); pid == "" {
			return "", fmt.Errorf("%s has no id", parent.GetType())
		} else {
			ids = append(ids, pid)
		}

		parent = parent.GetParent()
	}

	if len(ss) > 1 {
		for i, j := 0, len(ss)-1; i < j; i, j = i+1, j-1 {
			ss[i], ss[j] = ss[j], ss[i]
			ids[i], ids[j] = ids[j], ids[i]
		}
	}
	return s.generateCollectionPath(ss, ids, httpSchemeAndHost), nil
}

func (s *Schema) AddLinksToResource(r resource.Resource, httpSchemeAndHost string) error {
	if r.GetID() == "" {
		return fmt.Errorf("resource has no id")
	}

	cl, err := s.generateCollectionLink(r, httpSchemeAndHost)
	if err != nil {
		return err
	}
	r.SetLinks(s.generateResourceLinks(r, cl))
	return nil
}

func (s *Schema) AddLinksToResourceCollection(rs *resource.ResourceCollection, httpSchemeAndHost string) error {
	cl, err := s.generateCollectionLink(rs.GetCollection(), httpSchemeAndHost)
	if err != nil {
		return err
	}
	for _, r := range rs.GetResources() {
		r.SetLinks(s.generateResourceLinks(r, cl))
	}

	rs.SetLinks(map[resource.ResourceLinkType]resource.ResourceLink{resource.SelfLink: resource.ResourceLink(cl)})
	return nil
}

func (s *Schema) generateResourceLinks(r resource.Resource, parentLink string) map[resource.ResourceLinkType]resource.ResourceLink {
	links := make(map[resource.ResourceLinkType]resource.ResourceLink)
	selfLink := path.Join(parentLink, r.GetID())
	handler := s.GetHandler()
	if handler.GetListHandler() != nil {
		links[resource.CollectionLink] = resource.ResourceLink(parentLink)
	}
	if handler.GetGetHandler() != nil {
		links[resource.SelfLink] = resource.ResourceLink(selfLink)
	}
	if handler.GetUpdateHandler() != nil {
		links[resource.UpdateLink] = resource.ResourceLink(selfLink)
	}
	if handler.GetDeleteHandler() != nil {
		links[resource.RemoveLink] = resource.ResourceLink(selfLink)
	}
	for _, child := range s.GetChildren() {
		childName := child.ResourceName()
		links[resource.ResourceLinkType(childName)] = resource.ResourceLink(path.Join(selfLink, childName))
	}
	return links
}

func (s *Schema) WriteJsonDoc(path string, parents []string) error {
	docMgr := resourcedoc.NewDocumentManager(s.resourceKindName, s.resourceKind, s.handler, parents)
	return docMgr.WriteJsonFile(path)
}
