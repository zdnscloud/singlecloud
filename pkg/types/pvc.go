package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetPersistentVolumeClaimSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type PersistentVolumeClaim struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	RequestStorageSize string `json:"requestStorageSize"`
	StorageClassName   string `json:"storageClassName"`
	VolumeName         string `json:"volumeName"`
	ActualStorageSize  string `json:"actualStorageSize"`
	Status             string `json:"status"`
}

var PersistentVolumeClaimType = resttypes.GetResourceType(PersistentVolumeClaim{})
