package types

import (
	"github.com/zdnscloud/gorest/types"
)

type ClusterStatus string

const (
	CSRunning     ClusterStatus = "Running"
	CSUnreachable ClusterStatus = "Unreachable"
	CSCreateing   ClusterStatus = "Creating"
	CSUpdateing   ClusterStatus = "Updating"
	CSConnecting  ClusterStatus = "Connecting"
	CSUnavailable ClusterStatus = "Unavailable"
	CSCanceling   ClusterStatus = "Canceling"

	CSCancelAction = "cancel"

	ScVersionImported = "imported"
)

func SetClusterSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE", "POST", "PUT"}
	schema.ResourceActions = append(schema.ResourceActions, types.Action{
		Name: CSCancelAction,
	})
}

type Cluster struct {
	types.Resource     `json:",inline"`
	Nodes              []Node            `json:"nodes" rest:"required=true"`
	Network            ClusterNetwork    `json:"network" rest:"required=false"`
	PrivateRegistries  []PrivateRegistry `json:"privateRegistrys" rest:"required=false"`
	SingleCloudAddress string            `json:"singleCloudAddress" rest:"required=true"`
	Name               string            `json:"name" rest:"required=true"`
	Status             ClusterStatus     `json:"status"`
	NodesCount         int               `json:"nodeCount"`
	Version            string            `json:"version"`
	ScVersion          string            `json:"zcloudVersion"`

	Cpu             int64  `json:"cpu"`
	CpuUsed         int64  `json:"cpuUsed"`
	CpuUsedRatio    string `json:"cpuUsedRatio"`
	Memory          int64  `json:"memory"`
	MemoryUsed      int64  `json:"memoryUsed"`
	MemoryUsedRatio string `json:"memoryUsedRatio"`
	Pod             int64  `json:"pod"`
	PodUsed         int64  `json:"podUsed"`
	PodUsedRatio    string `json:"podUsedRatio"`

	SSHUser             string   `json:"sshUser" rest:"required=true"`
	SSHKey              string   `json:"sshKey" rest:"required=true"`
	SSHPort             string   `json:"sshPort" rest:"required=false,default=22"`
	DockerSocket        string   `json:"dockerSocket,omitempty"`
	KubernetesVersion   string   `json:"kubernetesVersion,omitempty"`
	IgnoreDockerVersion bool     `json:"ignoreDockerVersion" rest:"required=false,default=false"`
	ClusterCidr         string   `json:"clusterCidr" rest:"required=false,default=10.42.0.0/16"`
	ServiceCidr         string   `json:"serviceCidr" rest:"required=false,default=10.43.0.0/16"`
	ClusterDomain       string   `json:"clusterDomain" rest:"required=true"`
	ClusterDNSServiceIP string   `json:"clusterDNSServiceIP,omitempty" rest:"required=false,default=10.43.0.10"`
	ClusterUpstreamDNS  []string `json:"clusterUpstreamDNS"`
	DisablePortCheck    bool     `json:"disablePortCheck" rest:"required=false,default=false"`
}

type ClusterNetwork struct {
	Plugin string `json:"plugin" rest:"required=false,default=flannel"`
	Iface  string `json:"iface"`
}

type PrivateRegistry struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	CAcert   string `json:"caCert"`
}

var ClusterType = types.GetResourceType(Cluster{})
