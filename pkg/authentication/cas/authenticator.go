package cas

import (
	"net/http"
	"net/url"
	"strings"
)

type Authenticator struct {
	client *Client
}

func NewAuthenticator(casServer string) (*Authenticator, error) {
	url, err := url.Parse(casServer)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		client: NewClient(url),
	}, nil
}

func (a *Authenticator) Authenticate(w http.ResponseWriter, r *http.Request) (string, error) {
	resp, err := a.client.GetAuthResponse(w, r)
	if err != nil {
		return "", err
	} else if resp == nil {
		a.client.RedirectToLogin(w, r)
		return "", nil
	} else {
		return resp.User, nil
	}
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
