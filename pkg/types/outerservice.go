package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetOuterServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{NamespaceType}
}

type OuterService struct {
	resttypes.Resource `json:",inline"`
	EntryPoint         string                  `json:"entryPoint"`
	Services           map[string]InnerService `json:"services"`
}

var OuterServiceType = resttypes.GetResourceType(OuterService{})
