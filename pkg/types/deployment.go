package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	ActionGetHistory = "history"
	ActionRollback   = "rollback"
	ActionSetImage   = "setImage"

	VolumeTypeConfigMap        = "configmap"
	VolumeTypeSecret           = "secret"
	VolumeTypePersistentVolume = "persistentVolume"
)

func SetDeploymentSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE", "POST"}
	schema.Parents = []string{NamespaceType}
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name: ActionGetHistory,
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionRollback,
		Input: RollBackVersion{},
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionSetImage,
		Input: SetImage{},
	})
}

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
	resttypes.Resource `json:",inline"`
	Name               string                     `json:"name,omitempty"`
	Replicas           int                        `json:"replicas"`
	Containers         []Container                `json:"containers"`
	AdvancedOptions    AdvancedOptions            `json:"advancedOptions"`
	PersistentVolumes  []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status             WorkloadStatus             `json:"status,omitempty"`
}

type ExposedMetric struct {
	Path string `json:"path"`
	Port int    `json:"port"`
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
	Type               string            `json:"type,omitempty"`
	Status             string            `json:"status,omitempty"`
	LastTransitionTime resttypes.ISOTime `json:"lastTransitionTime,omitempty"`
	LastUpdateTime     resttypes.ISOTime `json:"lastUpdateTime,omitempty"`
	Reason             string            `json:"reason,omitempty"`
	Message            string            `json:"message,omitempty"`
}

var DeploymentType = resttypes.GetResourceType(Deployment{})

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
