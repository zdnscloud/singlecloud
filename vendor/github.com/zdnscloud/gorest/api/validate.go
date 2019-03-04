package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/zdnscloud/gorest/parse"
	"github.com/zdnscloud/gorest/types"
)

const (
	csrfCookie = "CSRF"
	csrfHeader = "X-API-CSRF"
)

func ValidateAction(request *types.APIContext) (*types.Action, *types.APIError) {
	if request.Action == "" || request.Method != http.MethodPost {
		return nil, nil
	}

	actions := request.Schema.CollectionActions
	if request.ID != "" {
		actions = request.Schema.ResourceActions
	}

	action, ok := actions[request.Action]
	if !ok {
		return nil, types.NewAPIError(types.InvalidAction, fmt.Sprintf("Invalid action: %s", request.Action))
	}

	return &action, nil
}

func CheckCSRF(apiContext *types.APIContext) *types.APIError {
	if !parse.IsBrowser(apiContext.Request, false) {
		return nil
	}

	cookie, err := apiContext.Request.Cookie(csrfCookie)
	if err == http.ErrNoCookie {
		bytes := make([]byte, 5)
		_, err := rand.Read(bytes)
		if err != nil {
			return types.NewAPIError(types.ServerError, fmt.Sprintf("Failed in CSRF processing: %s", err.Error()))
		}

		cookie = &http.Cookie{
			Name:  csrfCookie,
			Value: hex.EncodeToString(bytes),
		}
	} else if err != nil {
		return types.NewAPIError(types.InvalidCSRFToken, "Failed to parse cookies")
	} else if apiContext.Method != http.MethodGet {
		/*
		 * Very important to use apiContext.Method and not apiContext.Request.Method. The client can override the HTTP method with _method
		 */
		if cookie.Value == apiContext.Request.Header.Get(csrfHeader) {
			// Good
		} else if cookie.Value == apiContext.Request.URL.Query().Get(csrfCookie) {
			// Good
		} else {
			return types.NewAPIError(types.InvalidCSRFToken, "Invalid CSRF token")
		}
	}

	cookie.Path = "/"
	http.SetCookie(apiContext.Response, cookie)
	return nil
}
