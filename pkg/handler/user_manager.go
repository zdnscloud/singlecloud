package handler

import (
	"fmt"
	"time"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authentication/jwt"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type UserManager struct {
	api.DefaultHandler

	authorizer    *authorization.Authorizer
	authenticator *jwt.Authenticator
}

func newUserManager(authenticator *jwt.Authenticator, authorizer *authorization.Authorizer) *UserManager {
	return &UserManager{
		authenticator: authenticator,
		authorizer:    authorizer,
	}
}

func (m *UserManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create user")
	}

	user := ctx.Object.(*types.User)
	user.SetID(user.Name)
	user.SetType(types.UserType)
	user.SetCreationTimestamp(time.Now())
	if err := m.authenticator.AddUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate user name")
	}
	if err := m.authorizer.AddUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate user name")
	}
	return user, nil
}

func (m *UserManager) Get(ctx *resttypes.Context) interface{} {
	currentUser := getCurrentUser(ctx)
	if isAdmin(currentUser) == false && currentUser != ctx.Object.GetID() {
		return nil
	}

	if user := m.authorizer.GetUser(currentUser); user != nil {
		return user
	} else {
		return nil
	}
}

func (m *UserManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can delete user")
	}

	userName := ctx.Object.GetID()
	if err := m.authenticator.DeleteUser(userName); err != nil {
		return resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	if err := m.authorizer.DeleteUser(userName); err != nil {
		return resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	return nil
}

func (m *UserManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin could update user")
	}

	user := ctx.Object.(*types.User)
	if err := m.authorizer.UpdateUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, err.Error())
	} else {
		return user, nil
	}
}

func (m *UserManager) List(ctx *resttypes.Context) interface{} {
	currentUser := getCurrentUser(ctx)
	var users []*types.User
	if isAdmin(currentUser) {
		users = m.authorizer.ListUser()
	} else {
		users = []*types.User{m.authorizer.GetUser(currentUser)}
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
	token, err := m.authenticator.CreateToken(userName, up.Password)
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

	userName := getCurrentUser(ctx)
	if userName != ctx.Object.GetID() {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only user himself could reset his password")
	}

	if err := m.authenticator.ResetPassword(userName, param.OldPassword, param.NewPassword); err != nil {
		return resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
	}
	return nil
}

func getCurrentUser(ctx *resttypes.Context) string {
	currentUser := ctx.Request.Context().Value(types.CurrentUserKey)
	if currentUser == nil {
		return ""
	}
	return currentUser.(string)
}

func isAdmin(user string) bool {
	return user == types.Administrator
}
