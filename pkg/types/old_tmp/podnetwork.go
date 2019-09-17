package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetPodNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type PodNetwork struct {
	resttypes.Resource `json:",inline"`
	NodeName           string  `json:"nodeName"`
	PodCIDR            string  `json:"podCIDR"`
	PodIPs             []PodIP `json:"podIPs"`
}

type PodIP struct {
	Namespace string `json:"-"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
}

var PodNetworkType = resttypes.GetResourceType(PodNetwork{})
