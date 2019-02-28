package authorization

import (
	"net/http"

	"github.com/zdnscloud/gorest/httperror"
	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/gorest/types/slice"
)

type AllAccess struct {
}

func (*AllAccess) CanCreate(apiContext *types.APIContext, schema *types.Schema) error {
	if slice.ContainsString(schema.CollectionMethods, http.MethodPost) {
		return nil
	}
	return httperror.NewAPIError(httperror.PermissionDenied, "can not create "+schema.ID)
}

func (*AllAccess) CanGet(apiContext *types.APIContext, schema *types.Schema) error {
	if slice.ContainsString(schema.ResourceMethods, http.MethodGet) {
		return nil
	}
	return httperror.NewAPIError(httperror.PermissionDenied, "can not get "+schema.ID)
}

func (*AllAccess) CanList(apiContext *types.APIContext, schema *types.Schema) error {
	if slice.ContainsString(schema.CollectionMethods, http.MethodGet) {
		return nil
	}
	return httperror.NewAPIError(httperror.PermissionDenied, "can not list "+schema.ID)
}

func (*AllAccess) CanUpdate(apiContext *types.APIContext, schema *types.Schema) error {
	if slice.ContainsString(schema.ResourceMethods, http.MethodPut) {
		return nil
	}
	return httperror.NewAPIError(httperror.PermissionDenied, "can not update "+schema.ID)
}

func (*AllAccess) CanDelete(apiContext *types.APIContext, schema *types.Schema) error {
	if slice.ContainsString(schema.ResourceMethods, http.MethodDelete) {
		return nil
	}
	return httperror.NewAPIError(httperror.PermissionDenied, "can not delete "+schema.ID)
}
