package cas

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sync"
)

const (
	SessionCookieName = "_cas_session"
)

type Client struct {
	casServer *url.URL
	tickets   *MemoryStore
	client    *http.Client

	mu       sync.Mutex
	sessions map[string]string
}

func NewClient(casServer *url.URL) *Client {
	return &Client{
		casServer: casServer,
		tickets:   NewMemoryStore(),
		client:    &http.Client{},
		sessions:  make(map[string]string),
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
	u, err := c.casServer.Parse(path.Join(c.casServer.Path, "login"))
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

func (c *Client) logoutUrlForRequest(r *http.Request) (string, error) {
	u, err := c.casServer.Parse(path.Join(c.casServer.Path, "logout"))
	if err != nil {
		return "", err
	}

	service, err := requestURL(r)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(service))
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

func (c *Client) RedirectToLogout(w http.ResponseWriter, r *http.Request) {
	u, err := c.logoutUrlForRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.clearSession(w, r)
	http.Redirect(w, r, u, http.StatusFound)
}

func (c *Client) RedirectToLogin(w http.ResponseWriter, r *http.Request, service string) {
	u, err := c.loginUrlForRequest(r, service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("---> redirect to cas with service %s\n", u)
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
	cookie := getCookie(w, r)
	ticket, ok := c.sessions[cookie.Value]
	if ok == false {
		ticket = r.URL.Query().Get("ticket")
		if ticket == "" {
			return nil, nil
		}

		if err := c.validateTicket(ticket, r); err != nil {
			return nil, err
		}
		c.setSession(cookie.Value, ticket)
	}

	resp := c.tickets.Read(ticket)
	if resp == nil {
		clearCookie(w, cookie)
		return nil, fmt.Errorf("invalid ticket")
	} else {
		return resp, nil
	}
}

func getCookie(w http.ResponseWriter, r *http.Request) *http.Cookie {
	c, err := r.Cookie(SessionCookieName)
	if err != nil {
		c = &http.Cookie{
			Name:     SessionCookieName,
			Value:    newSessionId(),
			MaxAge:   86400,
			HttpOnly: false,
		}

		r.AddCookie(c) // so we can find it later if required
		http.SetCookie(w, c)
	}

	return c
}

func newSessionId() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	bytes := make([]byte, 64)
	rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = alphabet[v%byte(len(alphabet))]
	}
	return string(bytes)
}

func clearCookie(w http.ResponseWriter, c *http.Cookie) {
	c.MaxAge = -1
	http.SetCookie(w, c)
}

func (c *Client) setSession(id string, ticket string) {
	c.mu.Lock()
	c.sessions[id] = ticket
	c.mu.Unlock()
}

func (c *Client) clearSession(w http.ResponseWriter, r *http.Request) {
	cookie := getCookie(w, r)
	if s, ok := c.sessions[cookie.Value]; ok {
		c.tickets.Delete(s)
		c.deleteSession(s)
	}
	clearCookie(w, cookie)
}

func (c *Client) deleteSession(id string) {
	c.mu.Lock()
	delete(c.sessions, id)
	c.mu.Unlock()
}

func (c *Client) findAndDeleteSessionWithTicket(ticket string) {
	for s, t := range c.sessions {
		if t == ticket {
			c.deleteSession(s)
			return
		}
	}
}
