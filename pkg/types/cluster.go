package types

import (
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/zke/core"
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
	CSInit        ClusterStatus = "Init"
	CSDestroy     ClusterStatus = "Destroy"

	CSCancelAction        = "cancel"
	CSGetKubeConfigAction = "getkubeconfig"
	CSImportAction        = "import"

	DefaultNetworkPlugin       = "flannel"
	DefaultClusterCIDR         = "10.42.0.0/16"
	DefaultServiceCIDR         = "10.43.0.0/16"
	DefaultClusterDNSServiceIP = "10.43.0.10"
	DefaultClusterDomain       = "cluster.local"

	ScVersionImported = "imported"
)

var DefaultClusterUpstreamDNS = []string{
	"223.5.5.5",
	"114.114.114.114",
}

type Cluster struct {
	resource.ResourceBase `json:",inline"`
	Nodes                 []Node            `json:"nodes" rest:"required=true"`
	Network               ClusterNetwork    `json:"network"`
	PrivateRegistries     []PrivateRegistry `json:"privateRegistrys"`
	SingleCloudAddress    string            `json:"singleCloudAddress" rest:"required=true,minLen=1,maxLen=128"`
	Name                  string            `json:"name" rest:"required=true,minLen=1,maxLen=128"`
	Status                ClusterStatus     `json:"status"`
	NodesCount            int               `json:"nodeCount"`
	Version               string            `json:"version"`
	ScVersion             string            `json:"zcloudVersion"`

	Cpu             int64  `json:"cpu"`
	CpuUsed         int64  `json:"cpuUsed"`
	CpuUsedRatio    string `json:"cpuUsedRatio"`
	Memory          int64  `json:"memory"`
	MemoryUsed      int64  `json:"memoryUsed"`
	MemoryUsedRatio string `json:"memoryUsedRatio"`
	Pod             int64  `json:"pod"`
	PodUsed         int64  `json:"podUsed"`
	PodUsedRatio    string `json:"podUsedRatio"`

	SSHUser string `json:"sshUser" rest:"required=true,minLen=1,maxLen=128"`
	//sshkey is necessary for create, but we cat't get it by get or list api due to some security problem(all user can get the cluster sshkey by get or list api), so we do this required check in cluster handler and it's not necessary for update
	SSHKey              string   `json:"sshKey"`
	SSHPort             string   `json:"sshPort"`
	DockerSocket        string   `json:"dockerSocket,omitempty"`
	KubernetesVersion   string   `json:"kubernetesVersion,omitempty"`
	IgnoreDockerVersion bool     `json:"ignoreDockerVersion"`
	ClusterCidr         string   `json:"clusterCidr"`
	ServiceCidr         string   `json:"serviceCidr"`
	ClusterDomain       string   `json:"clusterDomain"`
	ClusterDNSServiceIP string   `json:"clusterDNSServiceIP,omitempty"`
	ClusterUpstreamDNS  []string `json:"clusterUpstreamDNS"`
	DisablePortCheck    bool     `json:"disablePortCheck"`
}

type ClusterNetwork struct {
	Plugin string `json:"plugin" rest:"options=flannel|calico"`
	Iface  string `json:"iface"`
}

type PrivateRegistry struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	CAcert   string `json:"caCert"`
}

func (c Cluster) CreateDefaultResource() resource.Resource {
	return &Cluster{
		Network: ClusterNetwork{
			Plugin: DefaultNetworkPlugin,
		},
		ClusterCidr:         DefaultClusterCIDR,
		ServiceCidr:         DefaultServiceCIDR,
		ClusterDomain:       DefaultClusterDomain,
		ClusterDNSServiceIP: DefaultClusterDNSServiceIP,
		ClusterUpstreamDNS:  DefaultClusterUpstreamDNS,
	}
}

func (c Cluster) CreateAction(name string) *resource.Action {
	switch name {
	case CSCancelAction:
		return &resource.Action{
			Name: CSCancelAction,
		}
	case CSGetKubeConfigAction:
		return &resource.Action{
			Name: CSGetKubeConfigAction,
		}
	case CSImportAction:
		return &resource.Action{
			Name:  CSImportAction,
			Input: &core.FullState{},
		}
	default:
		return nil
	}
}
