package parse

import (
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/gorest/util/name"
)

var (
	multiSlashRegexp = regexp.MustCompile("//+")
	allowedFormats   = map[string]bool{
		"json": true,
		"yaml": true,
	}
)

type ParsedURL struct {
	Version *types.APIVersion
	Type    string
	ID      string
	Action  string
	Parent  types.Parent
	Query   url.Values
}

func defaultURLParser(schemas *types.Schemas, url *url.URL) (ParsedURL, *types.APIError) {
	result := ParsedURL{}

	path := url.EscapedPath()
	path = multiSlashRegexp.ReplaceAllString(path, "/")
	version, parent, parts := parseVersionAndParent(schemas, path)

	if version == nil {
		return result, types.NewAPIError(types.NotFound, "no found version with url "+path)
	}

	result.Version = version
	result.Action = parseAction(url)
	result.Query = url.Query()
	result.Parent = parent

	result.Type = safeIndex(parts, 0)
	result.ID = safeIndex(parts, 1)

	return result, nil
}

func Parse(rw http.ResponseWriter, req *http.Request, schemas *types.Schemas) (*types.APIContext, *types.APIError) {
	result := types.NewAPIContext(req, rw, schemas)
	result.Method = parseMethod(req)
	result.ResponseFormat = parseResponseFormat(req)

	// The response format is guarenteed to be set even in the event of an error
	parsedURL, err := defaultURLParser(schemas, req.URL)
	// wait to check error, want to set as much as possible

	result.Type = parsedURL.Type
	result.ID = parsedURL.ID
	result.Action = parsedURL.Action
	result.Query = parsedURL.Query
	result.Version = parsedURL.Version
	result.Parent = parsedURL.Parent

	if err != nil {
		return result, err
	}

	if err := defaultResolver(result.Type, result); err != nil {
		return result, err
	}

	result.Type = result.Schema.ID
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

func parseVersionAndParent(schemas *types.Schemas, escapedPath string) (*types.APIVersion, types.Parent, []string) {
	parent := types.Parent{}
	version := versionsForPath(schemas, escapedPath)
	if version == nil {
		return nil, parent, nil
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

	if len(paths) <= 2 {
		return version, parent, paths
	} else {
		schema := schemas.Schema(version, paths[2])
		if schema != nil && name.GuessPluralName(schema.Parent) == paths[0] {
			return version, types.Parent{ID: paths[1], Name: paths[0]}, paths[2:]
		}

		return version, parent, nil
	}
}

func defaultResolver(typeName string, apiContext *types.APIContext) *types.APIError {
	if typeName == "" {
		return types.NewAPIError(types.NotFound, "no found schema name from url")
	}

	schema := apiContext.Schemas.Schema(apiContext.Version, typeName)
	if schema == nil {
		return types.NewAPIError(types.NotFound, "no found schema "+typeName)
	}

	apiContext.Schema = schema
	return nil
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
