package parse

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/zdnscloud/gorest/types"
)

var (
	multiSlashRegexp = regexp.MustCompile("//+")
	allowedFormats   = map[string]bool{
		"json": true,
		"yaml": true,
	}
)

func Parse(rw http.ResponseWriter, req *http.Request, schemas *types.Schemas) (*types.APIContext, *types.APIError) {
	result := types.NewAPIContext(req, rw, schemas)
	result.Method = parseMethod(req)
	result.ResponseFormat = parseResponseFormat(req)
	path := req.URL.EscapedPath()
	path = multiSlashRegexp.ReplaceAllString(path, "/")
	version, obj, schema, err := parseVersionAndResource(schemas, path)
	if err != nil {
		return result, err
	}

	result.Type = obj.GetType()
	result.ID = obj.GetID()
	result.Action = parseAction(req.URL)
	result.Query = req.URL.Query()
	result.Version = version
	result.Parent = obj.GetParent()
	result.Schema = schema

	if err := ValidateMethod(result); err != nil {
		return result, err
	}

	return result, nil
}

func versionsForPath(schemas *types.Schemas, escapedPath string) *types.APIVersion {
	for _, version := range schemas.Versions() {
		if strings.HasPrefix(escapedPath, path.Join("/", version.Group, version.Path)) {
			return &version
		}
	}
	return nil
}

func parseVersionAndResource(schemas *types.Schemas, escapedPath string) (*types.APIVersion, types.Object, *types.Schema, *types.APIError) {
	version := versionsForPath(schemas, escapedPath)
	if version == nil {
		return nil, nil, nil, types.NewAPIError(types.NotFound, "no found version with "+escapedPath)
	}

	if strings.HasSuffix(escapedPath, "/") {
		escapedPath = escapedPath[:len(escapedPath)-1]
	}

	versionParts := strings.Split(version.Path, "/")
	versionGroups := strings.Split(version.Group, "/")
	pp := strings.Split(escapedPath, "/")
	var pathParts []string
	for _, p := range pp {
		part, err := url.PathUnescape(p)
		if err == nil {
			pathParts = append(pathParts, part)
		} else {
			pathParts = append(pathParts, p)
		}
	}

	paths := pathParts[len(versionParts)+len(versionGroups):]

	var obj *types.Resource
	var schema *types.Schema
	for i := 0; i < len(paths); i += 2 {
		schema = schemas.Schema(version, paths[i])
		if schema == nil {
			return version, nil, nil, types.NewAPIError(types.NotFound, "no found schema for "+paths[i])
		}

		if i == 0 {
			obj = &types.Resource{
				ID:   safeIndex(paths, i+1),
				Type: schema.ID,
			}
			continue
		}

		if schema.Parent != obj.Type {
			return version, nil, nil, types.NewAPIError(types.InvalidType,
				fmt.Sprintf("schema %v parent should not be %s", schema.ID, obj.Type))
		}

		obj = &types.Resource{
			ID:     safeIndex(paths, i+1),
			Type:   schema.ID,
			Parent: obj,
		}
	}

	return version, obj, schema, nil
}

func safeIndex(slice []string, index int) string {
	if index >= len(slice) {
		return ""
	}
	return slice[index]
}

func parseResponseFormat(req *http.Request) string {
	format := req.URL.Query().Get("_format")

	if format != "" {
		format = strings.TrimSpace(strings.ToLower(format))
	}

	/* Format specified */
	if allowedFormats[format] {
		return format
	}

	if isYaml(req) {
		return "yaml"
	}

	return "json"
}

func isYaml(req *http.Request) bool {
	return strings.Contains(req.Header.Get("Accept"), "application/yaml")
}

func parseMethod(req *http.Request) string {
	method := req.URL.Query().Get("_method")
	if method == "" {
		method = req.Method
	}
	return method
}

func parseAction(url *url.URL) string {
	return url.Query().Get("action")
}
