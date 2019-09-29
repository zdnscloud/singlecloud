package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Namespace struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`

	Cpu             int64  `json:"cpu"`
	CpuUsed         int64  `json:"cpuUsed"`
	CpuUsedRatio    string `json:"cpuUsedRatio"`
	Memory          int64  `json:"memory"`
	MemoryUsed      int64  `json:"memoryUsed"`
	MemoryUsedRatio string `json:"memoryUsedRatio"`
	Pod             int64  `json:"pod"`
	PodUsed         int64  `json:"podUsed"`
	PodUsedRatio    string `json:"podUsedRatio"`

	PodsUseMostCPU    []*PodCpuInfo    `json:"podsUseMostCPU,omitempty"`
	PodsUseMostMemory []*PodMemoryInfo `json:"podsUseMostMemory,omitempty"`
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

type PodByCpuUsage []*PodCpuInfo

func (a PodByCpuUsage) Len() int           { return len(a) }
func (a PodByCpuUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PodByCpuUsage) Less(i, j int) bool { return a[i].CpuUsed > a[j].CpuUsed }

type PodByMemoryUsage []*PodMemoryInfo

func (a PodByMemoryUsage) Len() int           { return len(a) }
func (a PodByMemoryUsage) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PodByMemoryUsage) Less(i, j int) bool { return a[i].MemoryUsed > a[j].MemoryUsed }
