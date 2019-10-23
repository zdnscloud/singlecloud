package charts

type Vanguard struct {
	Deployment VanguardDeploy    `json:"deployment"`
	Configmap  VanguardConfigmap `json:"configmap"`
}

type VanguardDeploy struct {
	Replicas int `json:"replicas"`
}

type VanguardConfigmap struct {
	Logger     ConfigLog        `json:"logger"`
	Cache      ConfigCache      `json:"cache"`
	Kubernetes ConfigKubernetes `json:"kubernetes"`
}

type ConfigLog struct {
	GeneralLog GeneralLog `json:"general_log"`
}

type GeneralLog struct {
	Enable bool `json:"enable"`
}

type ConfigCache struct {
	Enable bool `json:"enable"`
}

type ConfigKubernetes struct {
	ClusterDnsServer      string `json:"cluster_dns_server"`
	ClusterDomain         string `json:"cluster_domain"`
	ClusterCIDR           string `json:"cluster_cidr"`
	ClusterServiceIpRange string `json:"cluster_service_ip_range"`
}
