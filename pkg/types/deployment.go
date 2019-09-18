package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	ActionGetHistory = "history"
	ActionRollback   = "rollback"
	ActionSetImage   = "setImage"

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
	Name                  string                     `json:"name,omitempty"`
	Replicas              int                        `json:"replicas"`
	Containers            []Container                `json:"containers"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
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
			Input: RollBackVersion{},
		}
	case ActionSetImage:
		return &resource.Action{
			Name:  ActionSetImage,
			Input: SetImage{},
		}
	default:
		return nil
	}
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
	Version int    `json:"version"`
	Reason  string `json:"reason"`
}

type SetImage struct {
	Reason string           `json:"reason"`
	Images []ContainerImage `json:"images"`
}

type ContainerImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}
