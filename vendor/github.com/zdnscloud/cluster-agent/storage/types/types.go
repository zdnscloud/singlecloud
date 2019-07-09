package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetStorageSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

var StorageType = resttypes.GetResourceType(Storage{})

type Storage struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	Size               string `json:"size"`
	FreeSize           string `json:"freesize"`
	UsedSize           string `json:"usedsize"`
	Nodes              []Node `json:"nodes"`
	PVs                []PV   `json:"pvs"`
}

type PV struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	Pods             []Pod  `json:"pods"`
	StorageClassName string `json:"-"`
}

type Node struct {
	Name     string `json:"name"`
	Size     string `json:"size"`
	FreeSize string `json:"freesize"`
	UsedSize string `json:"usedsize"`
}

type Pod struct {
	Name string `json:"name"`
}
