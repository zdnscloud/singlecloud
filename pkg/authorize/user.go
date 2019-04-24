package authorize

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	Administrator string = "admin"
	AdminPasswd   string = "6710fc5dd8cd10e010af0083d9573fd327e8e67e" //hex encoding for sha1(zdns)
	AllNamespace  string = "_all"
)

var adminUser = &types.User{
	Name:     Administrator,
	Password: AdminPasswd,
}

type UserManager struct {
	repo  *TokenRepo
	users map[string]*types.User
	lock  sync.RWMutex
}

func NewUserManager(secret []byte, validDuration time.Duration) *UserManager {
	users := make(map[string]*types.User)
	adminUser.SetID(Administrator)
	adminUser.SetType(types.UserType)
	adminUser.SetCreationTimestamp(time.Now())
	users[Administrator] = adminUser
	return &UserManager{
		repo:  NewTokenRepo(secret, validDuration),
		users: users,
	}
}

func (m *UserManager) HandleRequest(ctx *resttypes.Context) *resttypes.APIError {
	if ctx.Object.GetType() == types.UserType {
		if ctx.Action != nil && ctx.Action.Name == types.ActionLogin {
			return nil
		}
	}

	token, ok := getTokenFromRequest(ctx.Request)
	if ok == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "please provide token")
	}

	user, err := m.repo.ParseToken(token)
	if err != nil {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "invalid token:"+err.Error())
	}

	if err := m.authenticateUser(user, ctx); err != nil {
		return resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
	} else {
		return nil
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

func (m *UserManager) authenticateUser(userName string, ctx *resttypes.Context) error {
	if userName == Administrator {
		return nil
	}

	m.lock.RLock()
	user, ok := m.users[userName]
	m.lock.RUnlock()
	if ok == false {
		return fmt.Errorf("user %s doesn't exists", userName)
	}

	ctx.Set(types.CurrentUserKey, user)

	ancestors := resttypes.GetAncestors(ctx.Object)
	if len(ancestors) >= 2 {
		if ancestors[0].GetType() == types.ClusterType && ancestors[1].GetType() == types.NamespaceType {
			cluster := ancestors[0].GetID()
			namespace := ancestors[1].GetID()
			for _, project := range user.Projects {
				if project.Cluster == cluster && (project.Namespace == AllNamespace || project.Namespace == namespace) {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("user %s has no sufficient permission", userName)
}

func (m *UserManager) AddUser(user *types.User) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[user.Name]; ok {
		return fmt.Errorf("user %s already exists", user.Name)
	} else {
		user.SetID(user.Name)
		m.users[user.Name] = user
		return nil
	}
}

func (m *UserManager) DeleteUser(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.users[name]; ok {
		delete(m.users, name)
		return nil
	} else {
		return fmt.Errorf("user %s doesn't exist", name)
	}
}

func (m *UserManager) UpdateUser(user *types.User) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	name := user.GetID()
	if target, ok := m.users[name]; ok == false {
		return fmt.Errorf("user %s doesn't exist", name)
	} else {
		if user.Password != "" {
			target.Password = user.Password
		}
		target.Projects = user.Projects
		return nil
	}
}

func (m *UserManager) ResetPassword(name, oldPassword, newPassword string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if user, ok := m.users[name]; ok == false {
		return fmt.Errorf("user %s doesn't exist", name)
	} else if user.Password != oldPassword {
		return fmt.Errorf("old password isn't correct")
	} else {
		user.Password = newPassword
		m.users[name] = user
		return nil
	}
}

func (m *UserManager) CreateToken(userName, password string) (string, error) {
	m.lock.RLock()
	user, ok := m.users[userName]
	m.lock.RUnlock()
	if ok == false {
		return "", fmt.Errorf("user %s dosen't exist", userName)
	} else if user.Password != password {
		return "", fmt.Errorf("user %s password isn't correct", userName)
	}

	return m.repo.CreateToken(userName)
}

func (m *UserManager) GetUser(userName string) *types.User {
	m.lock.RLock()
	user, ok := m.users[userName]
	m.lock.RUnlock()

	if ok {
		return user
	} else {
		return nil
	}
}

func (m *UserManager) GetUsers() []*types.User {
	users := make([]*types.User, 0, len(m.users))
	m.lock.RLock()
	for _, user := range m.users {
		users = append(users, user)
	}
	m.lock.RUnlock()
	return users
}
