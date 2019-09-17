package schema

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
)

type VersionedSchemas struct {
	version *resource.APIVersion
	//instead generate from version very time
	//to optimize search performance
	versionUrl      string
	toplevelSchemas []*Schema
}

func NewVersionedSchemas(v *resource.APIVersion) *VersionedSchemas {
	return &VersionedSchemas{
		version:    v,
		versionUrl: v.GetUrl(),
	}
}

func (s *VersionedSchemas) VersionEquals(v *resource.APIVersion) bool {
	return s.version.Equal(v)
}

func (s *VersionedSchemas) Import(kind resource.ResourceKind, handler resource.Handler) error {
	schema, err := NewSchema(s.version, kind, handler)
	if err != nil {
		return err
	}

	parents := kind.GetParents()
	for _, parent := range parents {
		parentSchema := s.GetSchema(parent)
		if parentSchema == nil {
			return fmt.Errorf("%s who is parent of %s hasn't been imported", resource.DefaultKindName(parent), resource.DefaultKindName(kind))
		} else {
			if err := parentSchema.AddChild(schema); err != nil {
				return err
			}
		}
	}

	if len(parents) == 0 {
		return s.addTopleveSchema(schema)
	}

	return nil
}

var multiSlashRegexp = regexp.MustCompile("//+")

func (s *VersionedSchemas) CreateResourceFromRequest(method, path string, body []byte, action string) (resource.Resource, *goresterr.APIError) {
	if strings.HasPrefix(path, s.versionUrl) == false {
		return nil, nil
	}

	path = strings.TrimPrefix(path, s.versionUrl)

	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1] //get rid of last '/'
	}

	if len(path) == 0 {
		return nil, goresterr.NewAPIError(goresterr.InvalidFormat, "no schema name in url")
	} else {
		path = path[1:] //get rid of first '/'
	}

	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if seg, err := url.PathUnescape(segment); err == nil {
			segments[i] = seg
		}
	}

	segmentCount := len(segments)
	if segmentCount == 0 {
		return nil, goresterr.NewAPIError(goresterr.NotFound, "only api version without any resource")
	}

	for _, schema := range s.toplevelSchemas {
		if r, err := schema.CreateResourceFromPathSegments(nil, segments, method, action, body); err != nil {
			return nil, err
		} else if r != nil {
			return r, nil
		}
	}
	return nil, goresterr.NewAPIError(goresterr.NotFound, fmt.Sprintf("no resource with kind %s", segments[0]))
}

func (s *VersionedSchemas) addTopleveSchema(schema *Schema) error {
	for _, old := range s.toplevelSchemas {
		if old.Equal(schema) {
			return fmt.Errorf("duplicate import kind %s", schema.ResourceKindName())
		}
	}
	s.toplevelSchemas = append(s.toplevelSchemas, schema)
	return nil
}

func (s *VersionedSchemas) GetSchema(kind resource.ResourceKind) *Schema {
	for _, schema := range s.toplevelSchemas {
		if target := schema.GetSchema(kind); target != nil {
			return target
		}
	}
	return nil
}

func (s *VersionedSchemas) GenerateResourceRoute() resource.ResourceRoute {
	route := resource.NewResourceRoute()
	for _, schema := range s.toplevelSchemas {
		route.Merge(schema.GenerateResourceRoute(nil))
	}
	return route
}
