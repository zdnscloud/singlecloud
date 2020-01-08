package handler

import (
	"time"

	"github.com/zdnscloud/cement/log"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/authentication/jwt"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type UserManager struct {
	authorizer    *authorization.Authorizer
	authenticator *jwt.Authenticator
}

func newUserManager(authenticator *jwt.Authenticator, authorizer *authorization.Authorizer) *UserManager {
	return &UserManager{
		authenticator: authenticator,
		authorizer:    authorizer,
	}
}

func (m *UserManager) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create user")
	}

	user := ctx.Resource.(*types.User)
	if user.Password == "" {
		return nil, resterr.NewAPIError(resterr.NotNullable, "empty password")
	}

	user.SetID(user.Name)
	user.SetCreationTimestamp(time.Now())
	if err := m.authenticator.AddUser(user); err != nil {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, "duplicate user name")
	}
	if err := m.authorizer.AddUser(user); err != nil {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, "duplicate user name")
	}
	return user, nil
}

func (m *UserManager) Get(ctx *restresource.Context) restresource.Resource {
	currentUser := getCurrentUser(ctx)
	target := ctx.Resource.GetID()
	if isAdmin(currentUser) == false && currentUser != target {
		return nil
	}

	if user := m.authorizer.GetUser(target); user != nil {
		return user
	} else {
		return nil
	}
}

func (m *UserManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterr.NewAPIError(resterr.PermissionDenied, "only admin can delete user")
	}

	userName := ctx.Resource.GetID()
	if err := m.authenticator.DeleteUser(userName); err != nil {
		return resterr.NewAPIError(resterr.NotFound, err.Error())
	}
	if err := m.authorizer.DeleteUser(userName); err != nil {
		return resterr.NewAPIError(resterr.NotFound, err.Error())
	}
	return nil
}

func (m *UserManager) Update(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin could update user")
	}

	user := ctx.Resource.(*types.User)
	//reset user password
	if user.Password != "" {
		if err := m.authenticator.ResetPassword(user.GetID(), "", user.Password, true); err != nil {
			return nil, resterr.NewAPIError(resterr.NotFound, err.Error())
		}
	}
	//update user priviledge
	if err := m.authorizer.UpdateUser(user); err != nil {
		return nil, resterr.NewAPIError(resterr.NotFound, err.Error())
	}

	return user, nil
}

func (m *UserManager) List(ctx *restresource.Context) interface{} {
	currentUser := getCurrentUser(ctx)
	var users []*types.User
	if isAdmin(currentUser) {
		users = m.authorizer.ListUser()
	} else {
		user := m.authorizer.GetUser(currentUser)
		if user != nil {
			users = []*types.User{user}
		} else {
			log.Errorf("user %s is deleted during request", currentUser)
		}
	}
	return users
}

func (m *UserManager) Action(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	action := ctx.Resource.GetAction()
	switch action.Name {
	case types.ActionLogin:
		return m.login(ctx)
	case types.ActionResetPassword:
		return nil, m.resetPassword(ctx)
	default:
		return nil, nil
	}
}

func (m *UserManager) login(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	action := ctx.Resource.GetAction()
	up, ok := action.Input.(*types.UserPassword)
	if ok == false {
		return nil, resterr.NewAPIError(resterr.InvalidFormat, "login param not valid")
	}

	if up.Password == "" {
		return nil, resterr.NewAPIError(resterr.NotNullable, "empty password")
	}

	userName := ctx.Resource.GetID()
	token, err := m.authenticator.CreateToken(userName, up.Password)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidBodyContent, err.Error())
	} else {
		return types.LoginInfo{token}, nil
	}
}

func (m *UserManager) resetPassword(ctx *restresource.Context) *resterr.APIError {
	action := ctx.Resource.GetAction()
	param, ok := action.Input.(*types.ResetPassword)
	if ok == false {
		return resterr.NewAPIError(resterr.InvalidFormat, "reset password param not valid")
	}

	userName := getCurrentUser(ctx)
	if userName != ctx.Resource.GetID() {
		return resterr.NewAPIError(resterr.PermissionDenied, "only user himself could reset his password")
	}

	if err := m.authenticator.ResetPassword(userName, param.OldPassword, param.NewPassword, false); err != nil {
		return resterr.NewAPIError(resterr.PermissionDenied, err.Error())
	}
	return nil
}

func getCurrentUser(ctx *restresource.Context) string {
	currentUser := ctx.Request.Context().Value(types.CurrentUserKey)
	if currentUser == nil {
		return ""
	}
	return currentUser.(string)
}

func isAdmin(user string) bool {
	return user == types.Administrator
}
