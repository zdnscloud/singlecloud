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
	WebLogoutPath      = "/web/logout"
	WebLoginPath       = "/web/login"
	WebRolePath        = "/web/role"
	WebCASRedirectPath = "/web/casredirect"
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
	router.GET(WebRolePath, func(c *gin.Context) {
		var user, authBy string
		if a.CasAuth != nil {
			user, _ = a.CasAuth.Authenticate(c.Writer, c.Request)
			authBy = "CAS"
		}

		if user == "" {
			user, _ = a.JwtAuth.Authenticate(c.Writer, c.Request)
			if user != "" {
				authBy = "JWT"
			}
		}

		body, _ := json.Marshal(map[string]string{
			"user":   user,
			"authBy": authBy,
		})
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Write(body)
	})

	router.GET(WebCASRedirectPath, func(c *gin.Context) {
		if a.CasAuth != nil {
			if err := a.CasAuth.SaveTicket(c.Writer, c.Request); err != nil {
				body, _ := json.Marshal(map[string]string{
					"err": err.Error(),
				})
				c.Writer.Header().Set("Content-Type", "application/json")
				c.Writer.WriteHeader(http.StatusUnauthorized)
				c.Writer.Write(body)
				return
			}
		}
		http.Redirect(c.Writer, c.Request, indexPath(c.Request, "/index"), http.StatusFound)
	})

	router.POST(WebLoginPath, func(c *gin.Context) {
		a.JwtAuth.Login(c.Writer, c.Request)
	})

	router.GET(WebLogoutPath, func(c *gin.Context) {
		user, _ := a.JwtAuth.Authenticate(c.Writer, c.Request)
		if user != "" {
			a.JwtAuth.Logout(c.Writer, c.Request)
		} else if a.CasAuth != nil {
			a.CasAuth.Logout(c.Writer, c.Request)
		}
	})

	return nil
}

func (a *Authenticator) MiddlewareFunc() gin.HandlerFunc {
	var authExceptionPaths = []string{
		"/assets",
		"/apis/ws.zcloud.cn",
		WebRolePath,
		WebCASRedirectPath,
	}

	var jumpExceptionalPaths = []string{
		"/apis",
		"/login",
		"/web",
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, p := range authExceptionPaths {
			if strings.HasPrefix(path, p) {
				return
			}
		}

		userName, err := a.Authenticate(c.Writer, c.Request)
		if err != nil {
			log.Errorf("auth failed:%v", err)
			return
		}

		if userName != "" {
			ctx := context.WithValue(c.Request.Context(), types.CurrentUserKey, userName)
			c.Request = c.Request.WithContext(ctx)
		} else {
			log.Errorf("auth get empty user for request %s", c.Request.URL.String())
			doRedirect := true
			for _, p := range jumpExceptionalPaths {
				if strings.HasPrefix(path, p) {
					doRedirect = false
					break
				}
			}

			if doRedirect == false {
				return
			}

			if a.CasAuth != nil {
				log.Debugf("redirect path %v to cas", path)
				a.CasAuth.RedirectToLogin(c.Writer, c.Request, WebCASRedirectPath)
			} else {
				log.Debugf("redirect path %v to /login", path)
				http.Redirect(c.Writer, c.Request, indexPath(c.Request, "/login"), http.StatusFound)
			}
		}
	}
}
