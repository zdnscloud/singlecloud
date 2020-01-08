package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	ActionSearchPod = "searchPod"
)

type Namespace struct {
	resource.ResourceBase `json:",inline"`
	Name                  string           `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Cpu                   int64            `json:"cpu" rest:"description=readonly"`
	CpuUsed               int64            `json:"cpuUsed" rest:"description=readonly"`
	CpuUsedRatio          string           `json:"cpuUsedRatio" rest:"description=readonly"`
	Memory                int64            `json:"memory" rest:"description=readonly"`
	MemoryUsed            int64            `json:"memoryUsed" rest:"description=readonly"`
	MemoryUsedRatio       string           `json:"memoryUsedRatio" rest:"description=readonly"`
	Pod                   int64            `json:"pod" rest:"description=readonly"`
	PodUsed               int64            `json:"podUsed" rest:"description=readonly"`
	PodUsedRatio          string           `json:"podUsedRatio" rest:"description=readonly"`
	PodsUseMostCPU        []*PodCpuInfo    `json:"podsUseMostCPU,omitempty" rest:"description=readonly"`
	PodsUseMostMemory     []*PodMemoryInfo `json:"podsUseMostMemory,omitempty" rest:"description=readonly"`
}

type PodCpuInfo struct {
	Name    string `json:"name"`
	CpuUsed int64  `json:"cpuUsed"`
}

type PodMemoryInfo struct {
	Name       string `json:"name"`
	MemoryUsed int64  `json:"memoryUsed"`
}

func (n Namespace) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

type PodToSearch struct {
	Name string `json:"name"`
}

type PodInfo struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (n Namespace) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:   ActionSearchPod,
			Input:  &PodToSearch{},
			Output: &PodInfo{},
		},
	}
}

type PodByCpuUsage []*PodCpuInfo

func (a PodByCpuUsage) Len() int           { return len(a) }
func (a PodByCpuUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PodByCpuUsage) Less(i, j int) bool { return a[i].CpuUsed > a[j].CpuUsed }

type PodByMemoryUsage []*PodMemoryInfo

func (a PodByMemoryUsage) Len() int           { return len(a) }
func (a PodByMemoryUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PodByMemoryUsage) Less(i, j int) bool { return a[i].MemoryUsed > a[j].MemoryUsed }
