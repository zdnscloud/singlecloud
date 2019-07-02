package types

import (
	"github.com/zdnscloud/gorest/types"
)

type ClusterStatus string

const (
	CSRunning     ClusterStatus = "Running"
	CSUnreachable ClusterStatus = "Unreachable"
)

func SetClusterSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
}

type Cluster struct {
	types.Resource     `json:",inline"`
	Option             ClusterOption       `json:"option"`
	Nodes              []ClusterConfigNode `json:"nodes"`
	Network            ClusterNetwork      `json:"network"`
	PrivateRegistrys   []PrivateRegistry   `json:"privateRegistrys"`
	SingleCloudAddress string              `json:"singleCloudAddress"`
	Name               string              `json:"name"`
	Status             ClusterStatus       `json:"status"`
	NodesCount         int                 `json:"nodeCount"`
	Version            string              `json:"version"`

	Cpu             int64  `json:"cpu"`
	CpuUsed         int64  `json:"cpuUsed"`
	CpuUsedRatio    string `json:"cpuUsedRatio"`
	Memory          int64  `json:"memory"`
	MemoryUsed      int64  `json:"memoryUsed"`
	MemoryUsedRatio string `json:"memoryUsedRatio"`
	Pod             int64  `json:"pod"`
	PodUsed         int64  `json:"podUsed"`
	PodUsedRatio    string `json:"podUsedRatio"`
}

type ClusterOption struct {
	SSHUser             string   `json:"sshUser"`
	SSHKey              string   `json:"sshKey"`
	SSHPort             string   `json:"sshPort"`
	DockerSocket        string   `json:"dockerSocket,omitempty"`
	KubernetesVersion   string   `json:"kubernetesVersion,omitempty"`
	IgnoreDockerVersion bool     `json:"ignoreDockerVersion"`
	ClusterCidr         string   `json:"clusterCidr"`
	ServiceCidr         string   `json:"serviceCidr"`
	ClusterDomain       string   `json:"clusterDomain"`
	ClusterDNSServiceIP string   `json:"clusterDNSServiceIP,omitempty"`
	ClusterUpstreamDNS  []string `json:"upstreamnameservers"`
	DisablePortCheck    bool     `json:"disablePortCheck"`
}

type ClusterConfigNode struct {
	NodeName string `json:"name"`
	Address  string `json:"address"`
	// Node role in kubernetes cluster (controlplane, worker, etcd, storage or edge)
	Role []string `json:"roles"`
}

type ClusterNetwork struct {
	Plugin string `yaml:"plugin" json:"plugin"`
	Iface  string `yaml:"iface" json:"iface"`
}

type PrivateRegistry struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	CAcert   string `json:"caCert"`
}

var ClusterType = types.GetResourceType(Cluster{})
