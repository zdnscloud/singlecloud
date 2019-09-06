package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetBlockDeviceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type BlockDevice struct {
	resttypes.Resource `json:",inline"`
	NodeName           string `json:"nodeName"`
	BlockDevices       []Dev  `json:"blockDevices"`
	UsedBy             string `json:"usedby"`
}

type Dev struct {
	Name   string `json:"name"`
	Size   string `json:"size"`
	UsedBy string `json:"-"`
}
