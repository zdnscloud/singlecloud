package jwt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zdnscloud/gorest"
	resterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/authentication/session"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AdminPasswd string = "0192309fba8c6f0929b5b00867ebccac9a39e34e" //hex encoding for sha1(zcloud)
)

var (
	SessionCookieName  = "_jwt_session"
	tokenSecret        = []byte("hello single cloud")
	tokenValidDuration = 24 * 3600 * time.Second
)

type Authenticator struct {
	repo *TokenRepo

	lock     sync.Mutex
	users    map[string]string
	sessions *session.SessionMgr
	db       kvzoo.Table
}

func NewAuthenticator() (*Authenticator, error) {
	auth := &Authenticator{
		repo:     NewTokenRepo(tokenSecret, tokenValidDuration),
		sessions: session.New(SessionCookieName),
	}

	if err := auth.loadUsers(); err != nil {
		return nil, err
	}

	if _, ok := auth.users[types.Administrator]; ok == false {
		admin := &types.User{
			Name:     types.Administrator,
			Password: AdminPasswd,
		}
		admin.SetID(types.Administrator)
		auth.AddUser(admin)
	}

	return auth, nil
}

func (a *Authenticator) Authenticate(_ http.ResponseWriter, req *http.Request) (string, *resterr.APIError) {
	token, _ := a.sessions.GetSession(req)
	if token == "" {
		token = getFromHeader(req)
		if token == "" {
			return "", nil
		}
	}

	user, err := a.repo.ParseToken(token)
	if err != nil {
		return "", resterr.NewAPIError(resterr.ServerError, err.Error())
	} else {
		return user, nil
	}
}

func (a *Authenticator) AddUser(user *types.User) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	name := user.GetID()
	if _, ok := a.users[name]; ok {
		return fmt.Errorf("user %s already exists", name)
	} else {
		if err := a.addUser(user); err != nil {
			return err
		}
		a.users[name] = user.Password
		return nil
	}
}

func (a *Authenticator) HasUser(userName string) bool {
	a.lock.Lock()
	defer a.lock.Unlock()
	_, ok := a.users[userName]
	return ok
}

func (a *Authenticator) DeleteUser(userName string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if userName == types.Administrator {
		return fmt.Errorf("admin user cann't be deleted")
	}

	if _, ok := a.users[userName]; ok {
		if err := a.deleteUser(userName); err != nil {
			return err
		}
		delete(a.users, userName)
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", userName)
	}
}

func (a *Authenticator) ResetPassword(userName string, old, new string, force bool) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	old_, ok := a.users[userName]
	if ok == false {
		return fmt.Errorf("user %s doesn't exist", userName)
	}

	if !force && old_ != old {
		return fmt.Errorf("password isn't correct")
	}

	if new == "" {
		return fmt.Errorf("new password is empty")
	}

	if err := a.updateUser(userName, new); err != nil {
		return err
	}
	a.users[userName] = new
	return nil
}

func (a *Authenticator) CreateToken(userName, password string) (string, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	password_, ok := a.users[userName]
	if ok == false {
		return "", fmt.Errorf("user %s doesn't exist", userName)
	}
	if password != password_ {
		return "", fmt.Errorf("password isn't correct")
	}

	return a.repo.CreateToken(userName)
}

func (a *Authenticator) Login(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		apiErr := resterr.NewAPIError(resterr.InvalidFormat, err.Error())
		gorest.WriteResponse(w, apiErr.Status, apiErr)
		return
	}

	if err := json.Unmarshal(reqBody, &params); err != nil {
		apiErr := resterr.NewAPIError(resterr.InvalidFormat, "login param not valid")
		gorest.WriteResponse(w, apiErr.Status, apiErr)
		return
	}

	token, err := a.CreateToken(params.Name, params.Password)
	if err != nil {
		apiErr := resterr.NewAPIError(resterr.InvalidFormat, err.Error())
		gorest.WriteResponse(w, apiErr.Status, apiErr)
		return
	}

	a.sessions.AddSession(w, r, token)
}

func (a *Authenticator) Logout(w http.ResponseWriter, r *http.Request) {
	currentUser_ := r.Context().Value(types.CurrentUserKey)
	if currentUser_ == nil {
		return
	}
	a.sessions.ClearSession(w, r)
}

func getFromHeader(req *http.Request) string {
	reqToken := req.Header.Get("Authorization")
	if reqToken == "" {
		return ""
	}

	splitToken := strings.Split(reqToken, "Bearer ")
	if len(splitToken) != 2 {
		return ""
	}
	token := splitToken[1]
	if len(token) == 0 {
		return ""
	} else {
		return token
	}
}
