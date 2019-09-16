package authentication

import (
	"net/http"

	resttypes "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/authentication/cas"
	"github.com/zdnscloud/singlecloud/pkg/authentication/jwt"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
)

type Authenticator struct {
	JwtAuth *jwt.Authenticator
	CasAuth *cas.Authenticator
}

func New(casServer string, db storage.DB) (*Authenticator, error) {
	jwtAuth, err := jwt.NewAuthenticator(db)
	if err != nil {
		return nil, err
	}

	auth := &Authenticator{
		JwtAuth: jwtAuth,
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

func (a *Authenticator) Authenticate(w http.ResponseWriter, req *http.Request) (string, *resttypes.APIError) {
	user, err := a.JwtAuth.Authenticate(w, req)
	if err != nil {
		return "", err
	} else if user != "" {
		return user, nil
	}

	if a.CasAuth == nil {
		return "", nil
	} else {
		user, err := a.CasAuth.Authenticate(w, req)
		if err == nil && user != "" {
			if !a.JwtAuth.HasUser(user) {
				newUser := &types.User{Name: user}
				newUser.SetID(user)
				a.JwtAuth.AddUser(newUser)
			}
		}
		return user, err
	}
}
