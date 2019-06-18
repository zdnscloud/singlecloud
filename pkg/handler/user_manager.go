package handler

import (
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AdminPasswd string = "6710fc5dd8cd10e010af0083d9573fd327e8e67e" //hex encoding for sha1(zdns)
)

var adminUser = &types.User{
	Name:     authorization.Administrator,
	Password: AdminPasswd,
}

type UserManager struct {
	api.DefaultHandler

	clusters *ClusterManager
	lock     sync.Mutex
	users    map[string]*types.User
}

func newUserManager(clusters *ClusterManager) *UserManager {
	users := make(map[string]*types.User)
	adminUser.SetID(authorization.Administrator)
	adminUser.SetType(types.UserType)
	adminUser.SetCreationTimestamp(time.Now())
	users[authorization.Administrator] = adminUser
	return &UserManager{
		clusters: clusters,
		users:    users,
	}
}

func (m *UserManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create user")
	}

	user := ctx.Object.(*types.User)
	if err := m.clusters.authorizer.AddUser(user.Name, user.Projects); err != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate user name")
	}

	user.SetID(user.Name)
	user.SetType(types.UserType)
	user.SetCreationTimestamp(time.Now())
	m.lock.Lock()
	m.users[user.Name] = user
	m.lock.Unlock()
	return hideUserPassword(user), nil
}

func (m *UserManager) Get(ctx *resttypes.Context) interface{} {
	currentUser := getCurrentUser(ctx)
	if isAdmin(currentUser) == false && currentUser.Name != ctx.Object.GetID() {
		return nil
	}

	userName := ctx.Object.GetID()
	m.lock.Lock()
	defer m.lock.Unlock()
	if user, ok := m.users[userName]; ok {
		return hideUserPassword(user)
	} else {
		return nil
	}
}

func (m *UserManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can delete user")
	}

	userName := ctx.Object.GetID()
	m.lock.Lock()
	delete(m.users, userName)
	m.lock.Unlock()

	if err := m.clusters.authorizer.DeleteUser(userName); err != nil {
		return resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	return nil
}

func (m *UserManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin could update user")
	}

	user := ctx.Object.(*types.User)
	if err := m.clusters.authorizer.UpdateUser(user.Name, user.Projects); err != nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, err.Error())
	} else {
		m.lock.Lock()
		m.users[user.Name] = user
		m.lock.Unlock()
		return hideUserPassword(user), nil
	}
}

func (m *UserManager) List(ctx *resttypes.Context) interface{} {
	currentUser := getCurrentUser(ctx)
	var users []*types.User
	if isAdmin(currentUser) {
		m.lock.Lock()
		for _, user := range m.users {
			users = append(users, hideUserPassword(user))
		}
		m.lock.Unlock()
	} else {
		m.lock.Lock()
		user, ok := m.users[currentUser.Name]
		if ok {
			users = []*types.User{hideUserPassword(user)}
		}
		m.lock.Unlock()
	}
	return users
}

func (m *UserManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	switch ctx.Action.Name {
	case types.ActionLogin:
		return m.login(ctx)
	case types.ActionResetPassword:
		return nil, m.resetPassword(ctx)
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
	}
}

func (m *UserManager) login(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	up, ok := ctx.Action.Input.(*types.UserPassword)
	if ok == false {
		return nil, resttypes.NewAPIError(resttypes.InvalidFormat, "login param not valid")
	}

	userName := ctx.Object.GetID()
	m.lock.Lock()
	defer m.lock.Unlock()
	user, ok := m.users[userName]
	if ok == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "user name doesn't exists")
	} else if up.Password != user.Password {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "password isn't correct")
	}

	token, err := m.clusters.authenticator.JwtAuth.CreateToken(userName)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
	} else {
		return map[string]string{
			"token": token,
		}, nil
	}
}

func (m *UserManager) resetPassword(ctx *resttypes.Context) *resttypes.APIError {
	param, ok := ctx.Action.Input.(*types.ResetPassword)
	if ok == false {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "reset password param not valid")
	}

	user := getCurrentUser(ctx)
	if user.Name != ctx.Object.GetID() {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only user himself could reset his password")
	}

	if param.OldPassword != user.Password {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "password isn't correct")
	}
	user.Password = param.NewPassword
	return nil
}

func (m *UserManager) createAuthenticationHandler() api.HandlerFunc {
	return func(ctx *resttypes.Context) *resttypes.APIError {
		if ctx.Object.GetType() == types.UserType {
			if ctx.Action != nil && ctx.Action.Name == types.ActionLogin {
				return nil
			}
		}

		userName, err := m.clusters.authenticator.Authenticate(ctx.Response, ctx.Request)
		if err != nil {
			return resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
		}

		m.lock.Lock()
		user, ok := m.users[userName]
		m.lock.Unlock()
		if ok == false {
			return resttypes.NewAPIError(resttypes.PermissionDenied, "user doesn't exists")
		}
		ctx.Set(types.CurrentUserKey, user)

		ancestors := resttypes.GetAncestors(ctx.Object)
		if len(ancestors) < 2 {
			return nil
		}

		if ancestors[0].GetType() == types.ClusterType && ancestors[1].GetType() == types.NamespaceType {
			cluster := ancestors[0].GetID()
			namespace := ancestors[1].GetID()
			if m.clusters.authorizer.Authorize(user.Name, cluster, namespace) == false {
				return resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("user %s has no sufficient permission to work on cluster %s namespace %s", userName, cluster, namespace))
			}
		}
		return nil
	}
}

func hideUserPassword(user *types.User) *types.User {
	ret := *user
	ret.Password = ""
	return &ret
}

func getCurrentUser(ctx *resttypes.Context) *types.User {
	currentUser_, ok := ctx.Get(types.CurrentUserKey)
	if ok == false {
		log.Fatalf("current user isn't set")
	}
	return currentUser_.(*types.User)
}

func isAdmin(user *types.User) bool {
	return user.Name == authorization.Administrator
}
