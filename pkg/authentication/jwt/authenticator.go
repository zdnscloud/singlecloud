package jwt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authentication/session"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AdminPasswd string = "6710fc5dd8cd10e010af0083d9573fd327e8e67e" //hex encoding for sha1(zdns)
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
}

func NewAuthenticator() *Authenticator {
	users := make(map[string]string)
	users[types.Administrator] = AdminPasswd
	return &Authenticator{
		repo:     NewTokenRepo(tokenSecret, tokenValidDuration),
		users:    users,
		sessions: session.New(SessionCookieName),
	}
}

func (a *Authenticator) Authenticate(_ http.ResponseWriter, req *http.Request) (string, *resttypes.APIError) {
	token, _ := a.sessions.GetSession(req)
	if token == "" {
		token = getFromHeader(req)
		if token == "" {
			return "", nil
		}
	}

	user, err := a.repo.ParseToken(token)
	if err != nil {
		return "", resttypes.NewAPIError(resttypes.ServerError, err.Error())
	} else {
		return user, nil
	}
}

func (a *Authenticator) AddUser(user *types.User) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if _, ok := a.users[user.Name]; ok {
		return fmt.Errorf("user %s already exists", user)
	} else {
		a.users[user.Name] = user.Password
		return nil
	}
}

func (a *Authenticator) DeleteUser(userName string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if userName == types.Administrator {
		return fmt.Errorf("admin user cann't be deleted")
	}

	if _, ok := a.users[userName]; ok {
		delete(a.users, userName)
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", userName)
	}
}

func (a *Authenticator) ResetPassword(userName string, old, new string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if old_, ok := a.users[userName]; ok {
		if old_ != old {
			return fmt.Errorf("password isn't correct")
		}
		a.users[userName] = new
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", userName)
	}
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
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := json.Unmarshal(reqBody, &params); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := a.CreateToken(params.Name, params.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
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
