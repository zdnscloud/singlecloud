package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetBlockDeviceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type Data struct {
	Type         string            `json:"type"`
	ResourceType string            `json:"resourceType"`
	Links        map[string]string `json:"links"`
	Data         []BlockDevice     `json:"data"`
}

type BlockDevice struct {
	resttypes.Resource `json:",inline"`
	Hosts              []Host `json:"hosts"`
}

type Host struct {
	NodeName     string `json:"nodeName"`
	BlockDevices []Dev  `json:"blockDevices"`
}
type Dev struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	Parted     bool   `json:"parted"`
	Filesystem bool   `json:"filesystem"`
	Mount      bool   `json:"mount"`
}

var BlockDeviceType = resttypes.GetResourceType(BlockDevice{})
