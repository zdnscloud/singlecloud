package cluster

import (
	"github.com/zdnscloud/gorest/types"
)

func SetSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET"}
}
