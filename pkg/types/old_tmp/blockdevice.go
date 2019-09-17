package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
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
}

type Dev struct {
	Name string `json:"name"`
	Size string `json:"size"`
}
