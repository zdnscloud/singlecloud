package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetUserSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE", "POST"}
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  "login",
		Input: UserPassword{},
	})
}

type UserPassword struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type User struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	Password           string `json:"password"`
	Roles              []Role `json:"roles"`
}

type Role struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
}

var UserType = resttypes.GetResourceType(User{})
