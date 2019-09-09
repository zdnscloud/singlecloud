package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	StorageClassNameLVM  = "lvm"
	StorageClassNameCeph = "cephfs"
	StorageClassNameTemp = "temporary"
)

func SetStatefulSetSchema(schema *resttypes.Schema, handler resttypes.Handler) {
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

type StatefulSet struct {
	resttypes.Resource `json:",inline"`
	Name               string                     `json:"name,omitempty"`
	Replicas           int                        `json:"replicas"`
	Containers         []Container                `json:"containers"`
	AdvancedOptions    AdvancedOptions            `json:"advancedOptions"`
	PersistentVolumes  []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status             StatefulSetStatus          `json:"status,omitempty"`
}

type PersistentVolumeTemplate struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

type StatefulSetStatus struct {
	ObservedGeneration int                 `json:"observedGeneration,omitempty"`
	Replicas           int                 `json:"replicas,omitempty"`
	ReadyReplicas      int                 `json:"readyReplicas,omitempty"`
	CurrentReplicas    int                 `json:"currentReplicas,omitempty"`
	UpdatedReplicas    int                 `json:"updatedReplicas,omitempty"`
	CurrentRevision    string              `json:"currentRevision,omitempty"`
	UpdateRevision     string              `json:"updateRevision,omitempty"`
	CollisionCount     int                 `json:"collisionCount,omitempty"`
	Conditions         []WorkloadCondition `json:"conditions,omitempty"`
}

var StatefulSetType = resttypes.GetResourceType(StatefulSet{})
