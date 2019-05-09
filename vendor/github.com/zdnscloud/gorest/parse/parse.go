package parse

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gorest/types"
)

var (
	multiSlashRegexp = regexp.MustCompile("//+")
	allowedFormats   = map[string]bool{
		"json": true,
		"yaml": true,
	}
)

func Parse(rw http.ResponseWriter, req *http.Request, schemas *types.Schemas) (*types.Context, *types.APIError) {
	ctx := types.NewContext(req, rw, schemas)
	err := parseVersionAndResource(ctx)
	if err != nil {
		return ctx, err
	}

	if err := parseMethod(ctx); err != nil {
		return ctx, err
	}

	if err := parseAction(ctx); err != nil {
		return ctx, err
	}

	if err := parseResponseFormat(ctx); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func parseVersionAndResource(ctx *types.Context) *types.APIError {
	path := multiSlashRegexp.ReplaceAllString(ctx.Request.URL.EscapedPath(), "/")
	var version *types.APIVersion
	for _, v := range ctx.Schemas.Versions() {
		if strings.HasPrefix(path, v.GetVersionURL()) {
			version = &v
			break
		}
	}
	if version == nil {
		return types.NewAPIError(types.NotFound, "no found version with "+path)
	}

	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	versionURL := version.GetVersionURL()
	if len(path) <= len(versionURL) {
		return types.NewAPIError(types.InvalidFormat, "no schema name in url "+path)
	}

	path = path[len(versionURL)+1:]
	pp := strings.Split(path, "/")
	var paths []string
	for _, p := range pp {
		part, err := url.PathUnescape(p)
		if err == nil {
			paths = append(paths, part)
		} else {
			paths = append(paths, p)
		}
	}

	if len(paths) == 0 {
		return types.NewAPIError(types.NotFound, "no found schema with url "+path)
	}

	var obj *types.Resource
	var schema *types.Schema
	for i := 0; i < len(paths); i += 2 {
		schema = ctx.Schemas.Schema(version, paths[i])
		if schema == nil {
			return types.NewAPIError(types.NotFound, "no found schema for "+paths[i])
		}

		if i == 0 {
			obj = &types.Resource{
				ID:     safeIndex(paths, i+1),
				Type:   schema.GetType(),
				Schema: schema,
			}
			continue
		}

		if types.IsElemInArray(obj.Type, schema.Parents) == false {
			return types.NewAPIError(types.InvalidType,
				fmt.Sprintf("schema %v parent should not be %s", schema.GetType(), obj.Type))
		}

		obj = &types.Resource{
			ID:     safeIndex(paths, i+1),
			Type:   schema.GetType(),
			Parent: obj,
			Schema: schema,
		}
	}

	ctx.Object = obj
	return nil
}

func safeIndex(slice []string, index int) string {
	if index >= len(slice) {
		return ""
	}
	return slice[index]
}

func parseResponseFormat(ctx *types.Context) *types.APIError {
	format_ := ctx.Request.URL.Query().Get("_format")

	if format_ != "" {
		format := types.ResponseFormat(format_)
		if format != types.ResponseJSON && format != types.ResponseYAML {
			return types.NewAPIError(types.NotFound, "unsupported format"+format_)
		}
		ctx.ResponseFormat = format
	} else if strings.Contains(ctx.Request.Header.Get("Accept"), "application/yaml") {
		ctx.ResponseFormat = types.ResponseYAML
	} else {
		ctx.ResponseFormat = types.ResponseJSON
	}
	return nil
}

func parseMethod(ctx *types.Context) *types.APIError {
	method := ctx.Request.Method
	schema := ctx.Object.GetSchema()
	allowed := schema.ResourceMethods
	if ctx.Object.GetID() == "" {
		allowed = schema.CollectionMethods
	}

	if slice.SliceIndex(allowed, method) == -1 {
		return types.NewAPIError(types.MethodNotAllowed, fmt.Sprintf("Method %s not supported", method))
	}

	ctx.Method = method
	return nil
}

func parseAction(ctx *types.Context) *types.APIError {
	action := ctx.Request.URL.Query().Get("action")
	if action == "" || ctx.Method != http.MethodPost {
		return nil
	}

	actions := ctx.Object.GetSchema().CollectionActions
	if ctx.Object.GetID() != "" {
		actions = ctx.Object.GetSchema().ResourceActions
	}

	for _, action_ := range actions {
		if action_.Name == action {
			ctx.Action = &action_
			return nil
		}
	}

	return types.NewAPIError(types.InvalidAction, fmt.Sprintf("Invalid action: %s", action))
}
