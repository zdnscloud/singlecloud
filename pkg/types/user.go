package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	CurrentUserKey      string = "_zlcoud_current_user"
	ActionLogin         string = "login"
	ActionResetPassword string = "resetPassword"
)

func SetUserSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE", "POST"}
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionLogin,
		Input: UserPassword{},
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionResetPassword,
		Input: ResetPassword{},
	})
}

type UserPassword struct {
	Password string `json:"password"`
}

type ResetPassword struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type User struct {
	resttypes.Resource `json:",inline"`
	Name               string    `json:"name"`
	Password           string    `json:"password,omitempty"`
	Projects           []Project `json:"projects"`
}

type Project struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
}

var UserType = resttypes.GetResourceType(User{})
