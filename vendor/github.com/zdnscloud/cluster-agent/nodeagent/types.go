package nodeagent

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetNodeAgentSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"POST", "GET"}
}

type NodeAgent struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	Address            string `json:"address"`
}

var NodeAgentType = resttypes.GetResourceType(NodeAgent{})
