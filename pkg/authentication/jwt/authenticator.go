package jwt

import (
	"net/http"
	"strings"
	"time"

	"github.com/zdnscloud/gorest/types"
)

var (
	tokenSecret        = []byte("hello single cloud")
	tokenValidDuration = 24 * 3600 * time.Second
)

type Authenticator struct {
	repo *TokenRepo
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{
		repo: NewTokenRepo(tokenSecret, tokenValidDuration),
	}
}

func (a *Authenticator) Authenticate(_ http.ResponseWriter, req *http.Request) (string, *types.APIError) {
	token, ok := getTokenFromRequest(req)
	if ok == false {
		return "", nil
	}

	user, err := a.repo.ParseToken(token)
	if err != nil {
		return "", types.NewAPIError(types.ServerError, err.Error())
	} else {
		return user, nil
	}
}

func (a *Authenticator) CreateToken(user string) (string, error) {
	return a.repo.CreateToken(user)
}

func getTokenFromRequest(req *http.Request) (string, bool) {
	reqToken := req.Header.Get("Authorization")
	if reqToken == "" {
		return "", false
	}

	splitToken := strings.Split(reqToken, "Bearer ")
	if len(splitToken) != 2 {
		return "", false
	}
	token := splitToken[1]
	if len(token) == 0 {
		return "", false
	} else {
		return token, true
	}
}
