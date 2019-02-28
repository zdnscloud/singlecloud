package types

import (
	"net/http"

	"github.com/zdnscloud/gorest/httperror"
	"github.com/zdnscloud/gorest/types/slice"
)

func (s *Schema) CanList(context *APIContext) error {
	if context == nil {
		if slice.ContainsString(s.CollectionMethods, http.MethodGet) {
			return nil
		}
		return httperror.NewAPIError(httperror.PermissionDenied, "can not list "+s.ID)
	}
	return context.AccessControl.CanList(context, s)
}

func (s *Schema) CanGet(context *APIContext) error {
	if context == nil {
		if slice.ContainsString(s.ResourceMethods, http.MethodGet) {
			return nil
		}
		return httperror.NewAPIError(httperror.PermissionDenied, "can not get "+s.ID)
	}
	return context.AccessControl.CanGet(context, s)
}

func (s *Schema) CanCreate(context *APIContext) error {
	if context == nil {
		if slice.ContainsString(s.CollectionMethods, http.MethodPost) {
			return nil
		}
		return httperror.NewAPIError(httperror.PermissionDenied, "can not create "+s.ID)
	}
	return context.AccessControl.CanCreate(context, s)
}

func (s *Schema) CanUpdate(context *APIContext) error {
	if context == nil {
		if slice.ContainsString(s.ResourceMethods, http.MethodPut) {
			return nil
		}
		return httperror.NewAPIError(httperror.PermissionDenied, "can not update "+s.ID)
	}
	return context.AccessControl.CanUpdate(context, s)
}

func (s *Schema) CanDelete(context *APIContext) error {
	if context == nil {
		if slice.ContainsString(s.ResourceMethods, http.MethodDelete) {
			return nil
		}
		return httperror.NewAPIError(httperror.PermissionDenied, "can not delete "+s.ID)
	}
	return context.AccessControl.CanDelete(context, s)
}
