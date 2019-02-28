package handler

import (
	"net/url"

	"github.com/zdnscloud/gorest/httperror"
	"github.com/zdnscloud/gorest/types"
)

func ErrorHandler(request *types.APIContext, err error) {
	var apiError *httperror.APIError
	if apiErr, ok := err.(*httperror.APIError); ok {
		if apiErr.Cause != nil {
			url, _ := url.PathUnescape(request.Request.URL.String())
			if url == "" {
				url = request.Request.URL.String()
			}
		}
		apiError = apiErr
	} else {
		apiError = &httperror.APIError{
			Code:    httperror.ServerError,
			Message: err.Error(),
		}
	}

	data := toError(apiError)
	request.WriteResponse(apiError.Code.Status, data)
}

func toError(apiError *httperror.APIError) map[string]interface{} {
	e := map[string]interface{}{
		"type":    "error",
		"status":  apiError.Code.Status,
		"code":    apiError.Code.Code,
		"message": apiError.Message,
	}
	if apiError.FieldName != "" {
		e["fieldName"] = apiError.FieldName
	}

	return e
}
