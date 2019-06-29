package cas

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/zdnscloud/singlecloud/pkg/authentication/session"
)

const (
	SessionCookieName = "_cas_session"
)

type Client struct {
	casServer *url.URL
	tickets   *MemoryStore
	client    *http.Client
	sessions  *session.SessionMgr
}

func NewClient(casServer *url.URL) *Client {
	return &Client{
		casServer: casServer,
		tickets:   NewMemoryStore(),
		client:    &http.Client{},
		sessions:  session.New(SessionCookieName),
	}
}

func requestURL(r *http.Request) (*url.URL, error) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return nil, err
	}

	u.Host = r.Host
	u.Scheme = "http"

	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		u.Scheme = scheme
	} else if r.TLS != nil {
		u.Scheme = "https"
	}

	return u, nil
}

func (c *Client) loginUrlForRequest(r *http.Request, service string) (string, error) {
	return c.casUrlForRequest(r, "login", service)
}

func (c *Client) logoutUrlForRequest(r *http.Request, service string) (string, error) {
	return c.casUrlForRequest(r, "logout", service)
}

func (c *Client) casUrlForRequest(r *http.Request, target, service string) (string, error) {
	u, err := c.casServer.Parse(path.Join(c.casServer.Path, target))
	if err != nil {
		return "", err
	}

	serviceUrl, err := requestURL(r)
	if err != nil {
		return "", err
	}
	serviceUrl.Path = service

	q := u.Query()
	q.Add("service", sanitisedURLString(serviceUrl))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (c *Client) serviceValidateUrlForRequest(ticket string, r *http.Request) (string, error) {
	u, err := c.casServer.Parse(path.Join(c.casServer.Path, "serviceValidate"))
	if err != nil {
		return "", err
	}

	service, err := requestURL(r)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(service))
	q.Add("ticket", ticket)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) validateUrlForRequest(ticket string, r *http.Request) (string, error) {
	u, err := c.casServer.Parse(path.Join(c.casServer.Path, "validate"))
	if err != nil {
		return "", err
	}

	service, err := requestURL(r)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(service))
	q.Add("ticket", ticket)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (c *Client) RedirectToLogout(w http.ResponseWriter, r *http.Request, service string) {
	u, err := c.logoutUrlForRequest(r, service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.sessions.ClearSession(w, r)
	http.Redirect(w, r, u, http.StatusFound)
}

func (c *Client) RedirectToLogin(w http.ResponseWriter, r *http.Request, service string) {
	u, err := c.loginUrlForRequest(r, service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, u, http.StatusFound)
}

func (c *Client) validateTicket(ticket string, service *http.Request) error {
	u, err := c.serviceValidateUrlForRequest(ticket, service)
	if err != nil {
		return err
	}

	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	r.Header.Add("User-Agent", "Golang CAS client gopkg.in/cas")

	resp, err := c.client.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return c.validateTicketCas1(ticket, service)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cas: validate ticket: %v", string(body))
	}

	success, err := ParseServiceResponse(body)
	if err != nil {
		return err
	}

	if err := c.tickets.Write(ticket, success); err != nil {
		return err
	}

	return nil
}

func (c *Client) validateTicketCas1(ticket string, service *http.Request) error {
	u, err := c.validateUrlForRequest(ticket, service)
	if err != nil {
		return err
	}

	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	r.Header.Add("User-Agent", "Golang CAS client gopkg.in/cas")
	resp, err := c.client.Do(r)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	body := string(data)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cas: validate ticket: %v", body)
	}
	if body == "no\n\n" {
		return nil // not logged in
	}
	success := &AuthenticationResponse{
		User: body[4 : len(body)-1],
	}

	return c.tickets.Write(ticket, success)
}

func (c *Client) GetAuthResponse(w http.ResponseWriter, r *http.Request) (*AuthenticationResponse, error) {
	ticket, err := c.sessions.GetSession(r)
	if err != nil {
		return nil, err
	}

	resp := c.tickets.Read(ticket)
	if resp == nil {
		c.sessions.ClearSession(w, r)
		return nil, fmt.Errorf("invalid ticket")
	} else {
		return resp, nil
	}
}

func (c *Client) SaveTicket(w http.ResponseWriter, r *http.Request) error {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		return fmt.Errorf("url has no ticket")
	}
	if err := c.validateTicket(ticket, r); err != nil {
		return err
	}
	c.sessions.AddSession(w, r, ticket)
	return nil
}
