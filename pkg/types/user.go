package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetUserSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
}

type User struct {
	resttypes.Resource `json:",inline"`
	Name               string    `json:"name"`
	Password           string    `json:"password"`
	Projects           []Project `json:"projects"`
}

type Project struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
}

var UserType = resttypes.GetResourceType(User{})
