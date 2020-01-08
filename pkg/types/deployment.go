package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	ActionGetHistory  = "history"
	ActionRollback    = "rollback"
	ActionSetImage    = "setImage"
	ActionSetPodCount = "setPodCount"

	VolumeTypeConfigMap        = "configmap"
	VolumeTypeSecret           = "secret"
	VolumeTypePersistentVolume = "persistentVolume"
)

type ContainerPort struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol" rest:"options=tcp|udp"`
}

type Container struct {
	Name         string          `json:"name" rest:"required=true,isDomain=true"`
	Image        string          `json:"image" rest:"required=true"`
	Command      []string        `json:"command,omitempty"`
	Args         []string        `json:"args,omitempty"`
	ExposedPorts []ContainerPort `json:"exposedPorts,omitempty"`
	Env          []EnvVar        `json:"env,omitempty"`
	Volumes      []Volume        `json:"volumes,omitempty"`
}

type Volume struct {
	Type      string `json:"type,omitempty" rest:"options=configmap|secret|persistentVolume"`
	Name      string `json:"name,omitempty" rest:"isDomain=true"`
	MountPath string `json:"mountPath,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type AdvancedOptions struct {
	ExposedMetric               ExposedMetric `json:"exposedMetric,omitempty"`
	ReloadWhenConfigChange      bool          `json:"reloadWhenConfigChange,omitempty"`
	DeletePVsWhenDeleteWorkload bool          `json:"deletePVsWhenDeleteWorkload,omitempty"`
}

type Deployment struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Replicas              int                        `json:"replicas" rest:"required=true,min=0,max=50"`
	Containers            []Container                `json:"containers" rest:"required=true"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions,omitempty" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes,omitempty"`
	Status                WorkloadStatus             `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

type ExposedMetric struct {
	Path string `json:"path,omitempty"`
	Port int    `json:"port,omitempty"`
}

func (d Deployment) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (d Deployment) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:   ActionGetHistory,
			Output: &VersionHistory{},
		},
		resource.Action{
			Name:  ActionRollback,
			Input: &RollBackVersion{},
		},
		resource.Action{
			Name:   ActionSetPodCount,
			Input:  &SetPodCount{},
			Output: &SetPodCount{},
		},
	}
}

type WorkloadStatus struct {
	ObservedGeneration  int                 `json:"observedGeneration,omitempty"`
	Replicas            int                 `json:"replicas,omitempty"`
	ReadyReplicas       int                 `json:"readyReplicas,omitempty"`
	UpdatedReplicas     int                 `json:"updatedReplicas,omitempty"`
	AvailableReplicas   int                 `json:"availableReplicas,omitempty"`
	UnavailableReplicas int                 `json:"unavailableReplicas,omitempty"`
	CurrentReplicas     int                 `json:"currentReplicas,omitempty"`
	CurrentRevision     string              `json:"currentRevision,omitempty"`
	UpdateRevision      string              `json:"updateRevision,omitempty"`
	CollisionCount      int                 `json:"collisionCount,omitempty"`
	Conditions          []WorkloadCondition `json:"conditions,omitempty"`
}

type WorkloadCondition struct {
	Type               string           `json:"type,omitempty"`
	Status             string           `json:"status,omitempty"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty"`
	LastUpdateTime     resource.ISOTime `json:"lastUpdateTime,omitempty"`
	Reason             string           `json:"reason,omitempty"`
	Message            string           `json:"message,omitempty"`
}

type VersionHistory struct {
	VersionInfos VersionInfos `json:"history"`
}

type VersionInfo struct {
	Name         string      `json:"name"`
	Namespace    string      `json:"namespace"`
	Version      int         `json:"version"`
	ChangeReason string      `json:"changeReason"`
	Containers   []Container `json:"containers"`
}

type VersionInfos []VersionInfo

func (vs VersionInfos) Len() int {
	return len(vs)
}
func (vs VersionInfos) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}
func (vs VersionInfos) Less(i, j int) bool {
	return vs[i].Version < vs[j].Version
}

type RollBackVersion struct {
	Version int    `json:"version" rest:"required=true"`
	Memo    string `json:"memo"`
}

type SetPodCount struct {
	Replicas int `json:"replicas" rest:"required=true,min=1,max=50"`
}
