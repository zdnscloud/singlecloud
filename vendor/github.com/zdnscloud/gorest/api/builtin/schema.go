package builtin

import (
	"github.com/zdnscloud/gorest/types"
)

var (
	Version = types.APIVersion{
		Group:   "zdns.cloud.cn",
		Version: "v1",
		Path:    "/api",
	}

	Schema = types.Schema{
		ID:         "schema",
		PluralName: "schemas",
		Version:    Version,
	}
)
