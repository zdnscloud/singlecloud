package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetPersistentVolumeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{ClusterType}
}

type PersistentVolume struct {
	resttypes.Resource `json:",inline"`
	Name               string   `json:"name"`
	StorageSize        string   `json:"storageSize"`
	StorageClassName   string   `json:"storageClassName"`
	ClaimRef           ClaimRef `json:"claimRef"`
	Status             string   `json:"status"`
}

type ClaimRef struct {
	Kind      string `json:"string"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

var PersistentVolumeType = resttypes.GetResourceType(PersistentVolume{})
