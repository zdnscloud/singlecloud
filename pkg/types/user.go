package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	Administrator       string = "admin"
	CurrentUserKey      string = "_zlcoud_current_user"
	ActionLogin         string = "login"
	ActionResetPassword string = "resetPassword"
)

type UserPassword struct {
	Password string `json:"password"`
}

type ResetPassword struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type User struct {
	resource.ResourceBase `json:",inline"`
	Name                  string    `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Password              string    `json:"password,omitempty" rest:"required=true"`
	Projects              []Project `json:"projects"`
}

type Project struct {
	Cluster   string `json:"cluster" rest:"isDomain=true"`
	Namespace string `json:"namespace" rest:"isDomain=true"`
}

type LoginInfo struct {
	Token string `json:"token"`
}

var UserActions = []resource.Action{
	resource.Action{
		Name:   ActionLogin,
		Input:  &UserPassword{},
		Output: &LoginInfo{},
	},
	resource.Action{
		Name:  ActionResetPassword,
		Input: &ResetPassword{},
	},
}

func (u User) GetActions() []resource.Action {
	return UserActions
}
