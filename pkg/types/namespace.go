package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Namespace struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`
}
