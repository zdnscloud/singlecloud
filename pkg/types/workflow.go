package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const WorkFlowEmptyTaskAction = "emptytask"

type WorkFlow struct {
	resource.ResourceBase `json:",inline"`
	Name                  string             `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Git                   GitInfo            `json:"git" rest:"required=true"`
	Image                 ImageInfo          `json:"image" rest:"required=true"`
	AutoDeploy            bool               `json:"autoDeploy" rest:"description=immutable"`
	Deploy                Deployment         `json:"deploy"`
	SubTasks              []WorkFlowSubTask  `json:"subTasks" rest:"description=readonly"`
	Status                WorkFlowTaskStatus `json:"status" rest:"description=readonly"`
}

type GitInfo struct {
	RepositoryURL string `json:"repositoryUrl" rest:"required=true"`
	Revision      string `json:"revision" rest:"required=true"`
	User          string `json:"user"`
	Password      string `json:"password"`
}

type ImageInfo struct {
	Name             string `json:"name" rest:"required=true"`
	RegistryUser     string `json:"registryUser" rest:"required=true"`
	RegistryPassword string `json:"registryPassword" rest:"required=true"`
}

type WorkFlows []*WorkFlow

func (w WorkFlows) Len() int {
	return len(w)
}

func (w WorkFlows) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w WorkFlows) Less(i, j int) bool {
	return w[i].Name < w[j].Name
}

func (w WorkFlow) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (w WorkFlow) SupportAsyncDelete() bool {
	return true
}

var WorkFlowActions = []resource.Action{
	resource.Action{
		Name: WorkFlowEmptyTaskAction,
	},
}

func (w WorkFlow) GetActions() []resource.Action {
	return WorkFlowActions
}
