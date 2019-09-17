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
	Name                  string    `json:"name"`
	Password              string    `json:"password,omitempty"`
	Projects              []Project `json:"projects"`
}

type Project struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
}

func (u User) CreateAction(name string) *resource.Action {
	switch name {
	case ActionLogin:
		return &resource.Action{
			Name:  ActionLogin,
			Input: &UserPassword{},
		}
	case ActionResetPassword:
		return &resource.Action{
			Name:  ActionResetPassword,
			Input: &ResetPassword{},
		}
	default:
		return nil
	}
}
