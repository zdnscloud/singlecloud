package types

type ZKEConfig struct {
	ClusterName string          `yaml:"cluster_name" json:"clusterName"`
	Option      ZKEConfigOption `yaml:"option" json:"option"`
	Nodes       []ZKEConfigNode `yaml:"nodes" json:"nodes"`
	// Kubernetes components
	Core               ZKEConfigCore     `yaml:"core,omitempty" json:"core,omitempty"`
	Network            ZKEConfigNetwork  `yaml:"network,omitempty" json:"network,omitempty"`
	Image              ZKEConfigImages   `yaml:"image,omitempty" json:"image,omitempty"`
	PrivateRegistries  []PrivateRegistry `yaml:"private_registries,omitempty" json:"privateRegistries,omitempty"`
	Authentication     ZKEConfigAuthn    `yaml:"authentication,omitempty" json:"authentication,omitempty"`
	Authorization      ZKEConfigAuthz    `yaml:"authorization,omitempty" json:"authorization,omitempty"`
	Monitor            ZKEConfigMonitor  `yaml:"monitor,omitempty" json:"monitor,omitempty"`
	SingleCloudAddress string            `yaml:"single_cloud_address,omitempty" json:"singleCloudAddress"`
	ConfigVersion      string            `yaml:"config_version" json:"configVersion"`
}

type ZKEConfigOption struct {
	SSHUser             string   `yaml:"ssh_user" json:"sshUser"`
	SSHKey              string   `yaml:"ssh_key" json:"sshKey"`
	SSHKeyPath          string   `yaml:"ssh_key_path" json:"sshKeyPath"`
	SSHPort             string   `yaml:"ssh_port" json:"sshPort"`
	DockerSocket        string   `yaml:"docker_socket,omitempty" json:"dockerSocket,omitempty"`
	KubernetesVersion   string   `yaml:"kubetnetes_version,omitempty" json:"kubernetesVersion,omitempty"`
	IgnoreDockerVersion bool     `yaml:"ignore_docker_version" json:"ignoreDockerVersion"`
	ClusterCidr         string   `yaml:"cluster_cidr" json:"clusterCidr"`
	ServiceCidr         string   `yaml:"service_cidr" json:"serviceCidr"`
	ClusterDomain       string   `yaml:"cluster_domain" json:"clusterDomain"`
	ClusterDNSServiceIP string   `yaml:"cluster_dns_serviceip,omitempty" json:"clusterDNSServiceIP,omitempty"`
	ClusterUpstreamDNS  []string `yaml:"up_stream_name_servers" json:"upStreamNameServers"`
	DisablePortCheck    bool     `yaml:"disable_port_check" json:"disablePortCheck"`
	PrefixPath          string   `yaml:"prefix_path,omitempty" json:"prefixPath,omitempty"`
}

type ZKEConfigNode struct {
	NodeName string `yaml:"name" json:"name"`
	Address  string `yaml:"address" json:"address"`
	// Optional - Internal address that will be used for components communication
	InternalAddress string `yaml:"internal_address,omitempty" json:"internalAddress,omitempty"`
	// Node role in kubernetes cluster (controlplane, worker, etcd, storage or edge)
	Role []string `yaml:"roles" json:"roles"`
	// SSH config
	User         string            `yaml:"user,omitempty" json:"sshUser,omitempty"`
	Port         string            `yaml:"port,omitempty" json:"sshPort,omitempty"`
	SSHKey       string            `yaml:"ssh_key,omitempty" json:"sshKey,omitempty"`
	SSHKeyPath   string            `yaml:"ssh_key_path,omitempty" json:"sshKeyPath,omitempty"`
	DockerSocket string            `yaml:"docker_socket,omitempty" json:"dockerSocket,omitempty"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type ZKEConfigCore struct {
	Etcd           ETCDService           `yaml:"etcd" json:"etcd"`
	KubeAPI        KubeAPIService        `yaml:"kube-api" json:"kubeApi"`
	KubeController KubeControllerService `yaml:"kube-controller" json:"kubeController"`
	Scheduler      SchedulerService      `yaml:"scheduler" json:"scheduler"`
	Kubelet        KubeletService        `yaml:"kubelet" json:"kubelet"`
	Kubeproxy      KubeproxyService      `yaml:"kubeproxy" json:"kubeproxy"`
}

type ZKEConfigNetwork struct {
	Plugin  string        `yaml:"plugin" json:"plugin"`
	Iface   string        `yaml:"iface" json:"iface"`
	DNS     DNSConfig     `yaml:"dns" json:"dns"`
	Ingress IngressConfig `yaml:"ingress" json:"ingress"`
}

type ZKEConfigImages struct {
	Etcd string `yaml:"etcd" json:"etcd"`
	// ZKE image
	Alpine                    string `yaml:"alpine" json:"alpine"`
	NginxProxy                string `yaml:"nginx_proxy" json:"nginxProxy"`
	CertDownloader            string `yaml:"cert_downloader" json:"certDownloader"`
	ZKERemover                string `yaml:"zke_remover" json:zkeRemover`
	KubernetesServicesSidecar string `yaml:"kubernetes_services_sidecar" json:"kubernetesServicesSidecar"`
	// CoreDNS image
	CoreDNS           string `yaml:"coredns" json:"coredns"`
	CoreDNSAutoscaler string `yaml:"coredns_autoscaler" json:"corednsAutoscaler"`
	// Kubernetes image
	Kubernetes string `yaml:"kubernetes" json:"kubernetes"`
	// Flannel image
	Flannel        string `yaml:"flannel" json:"flannel"`
	FlannelCNI     string `yaml:"flannel_cni" json:"flannelCni"`
	FlannelSidecar string `yaml:"flannel_sidecar" json:"flannelSidecar"`
	// Calico image
	CalicoNode        string `yaml:"calico_node" json:"calicoNode"`
	CalicoCNI         string `yaml:"calico_cni" json:"calicoCni"`
	CalicoControllers string `yaml:"calico_controllers" json:"calicoControllers"`
	CalicoCtl         string `yaml:"calico_ctl" json:"calicoCtl"`
	// Pod infra container image
	PodInfraContainer string `yaml:"pod_infra_container" json:"podInfraContainer"`
	// Ingress Controller image
	Ingress        string `yaml:"ingress" json:"ingress"`
	IngressBackend string `yaml:"ingress_backend" json:"ingressBackend"`
	MetricsServer  string `yaml:"metrics_server" json:"metricsServer"`
	// Zcloud image
	ClusterAgent    string `yaml:"cluster_agent" json:"clusterAgent"`
	NodeAgent       string `yaml:"node_agent" json:"nodeAgent"`
	StorageOperator string `yaml:"storage_operator" json:"storageOperator"`
	ZcloudShell     string `yaml:"zcloud_shell" json:"zcloud_shell"`
}
