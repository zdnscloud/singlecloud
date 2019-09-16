package resource

import (
	"path"
)

const GroupPrefix = "/apis"

type APIVersion struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
}

func (v *APIVersion) GetUrl() string {
	return path.Join(GroupPrefix, v.Group, v.Version)
}

func (v *APIVersion) Equal(other *APIVersion) bool {
	return v.Group == other.Group && v.Version == other.Version
}
