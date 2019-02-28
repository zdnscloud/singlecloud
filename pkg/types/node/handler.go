package node

import (
	"github.com/zdnscloud/gorest/types"
)

func SetSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = types.Parent{Name: "cluster"}
}
