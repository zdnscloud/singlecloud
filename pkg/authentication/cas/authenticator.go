package cas

import (
	"net/http"
	"net/url"

	"github.com/zdnscloud/gorest/resource"
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

func (a *Authenticator) Authenticate(w http.ResponseWriter, r *http.Request) (string, *types.APIError) {
	resp, err := a.client.GetAuthResponse(w, r)
	if err != nil || resp == nil {
		return "", nil
	} else {
		return resp.User, nil
	}
}

func (a *Authenticator) RedirectToLogin(w http.ResponseWriter, r *http.Request, service string) {
	a.client.RedirectToLogin(w, r, service)
}

func (a *Authenticator) Logout(w http.ResponseWriter, r *http.Request) {
	a.client.RemoveTicket(w, r)
	a.client.RedirectToLogout(w, r, "")
}

func (a *Authenticator) SaveTicket(w http.ResponseWriter, r *http.Request) error {
	return a.client.SaveTicket(w, r)
}
