package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type WorkFlow struct {
	resource.ResourceBase `json:",inline"`
	Name                  string             `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Git                   GitInfo            `json:"git" rest:"required=true"`
	Image                 ImageInfo          `json:"image" rest:"required=true"`
	AutoDeploy            bool               `json:"autoDeploy"`
	Deploy                Deployment         `json:"deploy"`
	Status                WorkFlowTaskStatus `json:"status" rest:"description=readonly,options=succeed|failed|running"`
}

type GitInfo struct {
	Repository string `json:"repository" rest:"required=true"`
	User       string `json:"user"`
	Password   string `json:"password"`
}

type ImageInfo struct {
	Name             string `json:"name" rest:"required=true"`
	RegistryUser     string `json:"registryUser" rest:"required=true"`
	RegistryPassword string `json:"registryPassword" rest:"required=true"`
}

func (w WorkFlow) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
