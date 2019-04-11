package handler

import (
	"time"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authorize"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type UserManager struct {
	DefaultHandler
	impl *authorize.UserManager
}

func newUserManager(secret []byte, tokenValidDuration time.Duration) *UserManager {
	return &UserManager{
		impl: authorize.NewUserManager(secret, tokenValidDuration),
	}
}

func (m *UserManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	user := ctx.Object.(*types.User)
	if err := m.impl.AddUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate user name")
	}
	user.SetID(user.Name)
	user.SetType(types.UserType)
	user.SetCreationTimestamp(time.Now())
	return user, nil
}

func (m *UserManager) Get(ctx *resttypes.Context) interface{} {
	var ret types.User
	user := m.impl.GetUser(ctx.Object.GetID())
	if user != nil {
		ret = *user
		ret.Password = ""
		return &ret
	} else {
		return nil
	}
}

func (m *UserManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if err := m.impl.DeleteUser(ctx.Object.GetID()); err != nil {
		return resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	return nil
}

func (m *UserManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	user := ctx.Object.(*types.User)
	if err := m.impl.UpdateUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	return user, nil
}

func (m *UserManager) List(ctx *resttypes.Context) interface{} {
	users := m.impl.GetUsers()
	var ret []*types.User
	for _, user := range users {
		tmp := *user
		tmp.Password = ""
		ret = append(ret, &tmp)
	}
	return ret
}

func (m *UserManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if ctx.Action.Name != "login" {
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, "only login is supported now")
	}

	up, ok := ctx.Action.Input.(*types.UserPassword)
	if ok == false {
		return nil, resttypes.NewAPIError(resttypes.InvalidFormat, "login param not valid")
	}

	token, err := m.impl.CreateToken(up.User, up.Password)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
	} else {
		return map[string]string{
			"token": token,
		}, nil
	}
}

func (m *UserManager) createAuthenticationHandler() api.HandlerFunc {
	return func(ctx *resttypes.Context) *resttypes.APIError {
		return m.impl.HandleRequest(ctx)
	}
}
