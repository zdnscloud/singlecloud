package authentication

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	CASRedirectPath = "/cas/redirect"
	CASRolePath     = "/cas/role"
	CASLogoutPath   = "/cas/logout"
)

func indexPath(r *http.Request, index string) string {
	u, _ := url.Parse(r.URL.String())
	u.Host = r.Host
	u.Scheme = "http"
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		u.Scheme = scheme
	} else if r.TLS != nil {
		u.Scheme = "https"
	}
	u.Path = index

	q := u.Query()
	for _, cleanParam := range []string{"ticket", "service"} {
		q.Del(cleanParam)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func (a *Authenticator) RegisterHandler(router gin.IRoutes) error {
	router.GET(CASRedirectPath, func(c *gin.Context) {
		if a.CasAuth != nil {
			user, err := a.CasAuth.Authenticate(c.Writer, c.Request)
			if err == nil && user != "" {
				http.Redirect(c.Writer, c.Request, indexPath(c.Request, "/"), http.StatusFound)
			}
		}
	})

	router.GET(CASRolePath, func(c *gin.Context) {
		if a.CasAuth != nil {
			user, _ := a.CasAuth.Authenticate(c.Writer, c.Request)
			body, _ := json.Marshal(map[string]string{
				"user": user,
			})
			c.Writer.WriteHeader(http.StatusOK)
			c.Writer.Write(body)
		}
	})

	router.GET(CASLogoutPath, func(c *gin.Context) {
		if a.CasAuth != nil {
			a.CasAuth.RedirectToLogout(c.Writer, c.Request)
		}
	})
	return nil
}

func (a *Authenticator) MiddlewareFunc() gin.HandlerFunc {
	var exceptionalPath = []string{
		"/apis",
		"/login",
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		doRedirect := true
		for _, ep := range exceptionalPath {
			if strings.HasPrefix(path, ep) {
				doRedirect = false
				break
			}
		}

		userName, err := a.Authenticate(c.Writer, c.Request)
		if err != nil {
			log.Errorf("cas auth failed:%v", err)
			return
		}

		if err == nil && userName == "" {
			if doRedirect && a.CasAuth != nil {
				log.Debugf("cas redirect")
				a.CasAuth.RedirectToLogin(c.Writer, c.Request, CASRedirectPath)
				c.Abort()
			}
		} else {
			ctx := context.WithValue(c.Request.Context(), types.CurrentUserKey, userName)
			c.Request = c.Request.WithContext(ctx)
		}
	}
}
