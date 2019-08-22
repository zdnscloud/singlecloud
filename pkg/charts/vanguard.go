package charts

type Vanguard struct {
	Deployment VanguardDeploy    `json:"deployment"`
	Configmap  VanguardConfigmap `json:"configmap"`
}

type VanguardDeploy struct {
	Replicas int           `json:"replicas" rest:"required=false,default=1"`
	Image    VanguardImage `json:"image"`
	Probe    LivenessProbe `json:"livenessProbe"`
}

type VanguardImage struct {
	Repository string `json:"repository" rest:"required=false,default=zdnscloud/vanguard"`
	Tag        string `json:"tag" rest:"required=false,default=v0.1"`
}

type LivenessProbe struct {
	HttpGet ProbeHttpGet `json:"httpGet"`
}

type ProbeHttpGet struct {
	Port int `json:"port" rest:"required=false,default=9000"`
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
	Enable bool `json:"enable" rest:"required=false,default=false"`
}

type ConfigCache struct {
	Enable bool `json:"enable" rest:"required=false,default=false"`
}

type ConfigKubernetes struct {
	ClusterDnsServer      string `json:"cluster_dns_server" rest:"required=false,default=10.43.0.10"`
	ClusterDomain         string `json:"cluster_domain" rest:"required=false,default=cluster.local"`
	ClusterCIDR           string `json:"cluster_cidr" rest:"required=false,default=10.42.0.0/16"`
	ClusterServiceIpRange string `json:"cluster_service_ip_range" rest:"required=false,default=10.43.0.0/16"`
}
