package node

import (
	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/handler"
)

func SetSchema(schema *types.Schema) {
	schema.Handler = &handler.Handler{}
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}
