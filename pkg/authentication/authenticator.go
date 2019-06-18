package authentication

import (
	"net/http"

	"github.com/zdnscloud/singlecloud/pkg/authentication/cas"
	"github.com/zdnscloud/singlecloud/pkg/authentication/jwt"
)

type Authenticator struct {
	JwtAuth *jwt.Authenticator
	CasAuth *cas.Authenticator
}

func New(casServer string) (*Authenticator, error) {
	auth := &Authenticator{
		JwtAuth: jwt.NewAuthenticator(),
	}

	if casServer != "" {
		casAuth, err := cas.NewAuthenticator(casServer)
		if err != nil {
			return nil, err
		}
		auth.CasAuth = casAuth
	}
	return auth, nil
}

func (a *Authenticator) Authenticate(w http.ResponseWriter, req *http.Request) (string, error) {
	user, err := a.JwtAuth.Authenticate(w, req)
	if err != nil {
		return "", err
	} else if user != "" {
		return user, nil
	}

	if a.CasAuth == nil {
		return "", nil
	} else {
		return a.CasAuth.Authenticate(w, req)
	}
}
