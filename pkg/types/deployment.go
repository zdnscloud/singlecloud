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
	Protocol string `json:"protocol"`
}

type Container struct {
	Name         string          `json:"name"`
	Image        string          `json:"image"`
	Command      []string        `json:"command,omitempty"`
	Args         []string        `json:"args,omitempty"`
	ExposedPorts []ContainerPort `json:"exposedPorts,omitempty"`
	Env          []EnvVar        `json:"env,omitempty"`
	Volumes      []Volume        `json:"volumes,omitempty"`
}

type Volume struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

type EnvVar struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type AdvancedOptions struct {
	ExposedMetric               ExposedMetric `json:"exposedMetric"`
	ReloadWhenConfigChange      bool          `json:"reloadWhenConfigChange"`
	DeletePVsWhenDeleteWorkload bool          `json:"deletePVsWhenDeleteWorkload"`
}

type Deployment struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name,omitempty" rest:"description=immutable"`
	Replicas              int                        `json:"replicas" rest:"min=0,max=50"`
	Containers            []Container                `json:"containers"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status                WorkloadStatus             `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

type ExposedMetric struct {
	Path string `json:"path"`
	Port int    `json:"port"`
}

func (d Deployment) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (d Deployment) CreateAction(name string) *resource.Action {
	switch name {
	case ActionGetHistory:
		return &resource.Action{
			Name: ActionGetHistory,
		}
	case ActionRollback:
		return &resource.Action{
			Name:  ActionRollback,
			Input: &RollBackVersion{},
		}
	case ActionSetPodCount:
		return &resource.Action{
			Name:  ActionSetPodCount,
			Input: &SetPodCount{},
		}
	default:
		return nil
	}
}

type WorkloadStatus struct {
	ObservedGeneration  int                 `json:"observedGeneration,omitempty" rest:"description=readonly"`
	Replicas            int                 `json:"replicas,omitempty" rest:"description=readonly"`
	ReadyReplicas       int                 `json:"readyReplicas,omitempty" rest:"description=readonly"`
	UpdatedReplicas     int                 `json:"updatedReplicas,omitempty" rest:"description=readonly"`
	AvailableReplicas   int                 `json:"availableReplicas,omitempty" rest:"description=readonly"`
	UnavailableReplicas int                 `json:"unavailableReplicas,omitempty" rest:"description=readonly"`
	CurrentReplicas     int                 `json:"currentReplicas,omitempty" rest:"description=readonly"`
	CurrentRevision     string              `json:"currentRevision,omitempty" rest:"description=readonly"`
	UpdateRevision      string              `json:"updateRevision,omitempty" rest:"description=readonly"`
	CollisionCount      int                 `json:"collisionCount,omitempty" rest:"description=readonly"`
	Conditions          []WorkloadCondition `json:"conditions,omitempty" rest:"description=readonly"`
}

type WorkloadCondition struct {
	Type               string           `json:"type,omitempty" rest:"description=readonly"`
	Status             string           `json:"status,omitempty" rest:"description=readonly"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty" rest:"description=readonly"`
	LastUpdateTime     resource.ISOTime `json:"lastUpdateTime,omitempty" rest:"description=readonly"`
	Reason             string           `json:"reason,omitempty" rest:"description=readonly"`
	Message            string           `json:"message,omitempty" rest:"description=readonly"`
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
