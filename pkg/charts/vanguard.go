package charts

type Vanguard struct {
	Deployment VanguardDeploy    `json:"deployment"`
	Service    VanguardService   `json:"service"`
	Configmap  VanguardConfigmap `json:"configmap"`
}

type VanguardDeploy struct {
	Replicas int           `json:"replicas" singlecloud:"required=false,default=1"`
	Image    VanguardImage `json:"image"`
	Probe    LivenessProbe `json:"livenessProbe"`
}

type VanguardImage struct {
	Repository string `json:"repository" singlecloud:"required=false,default=zdnscloud/vanguard"`
	Tag        string `json:"tag" singlecloud:"required=false,default=v0.1"`
}

type LivenessProbe struct {
	HttpGet ProbeHttpGet `json:"httpGet"`
}

type ProbeHttpGet struct {
	Port int `json:"port" singlecloud:"required=false,default=9000"`
}

type VanguardService struct {
	Port     int    `json:"port" singlecloud:"required=false,default=53"`
	Protocol string `json:"protocol" singlecloud:"required=false,default=UDP,options=UDP|TCP"`
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
	Enable bool `json:"enable" singlecloud:"required=false,default=false"`
}

type ConfigCache struct {
	Enable bool `json:"enable" singlecloud:"required=false,default=false"`
}

type ConfigKubernetes struct {
	ClusterDnsServer      string `json:"cluster_dns_server" singlecloud:"required=false,default=10.43.0.10"`
	ClusterDomain         string `json:"cluster_domain" singlecloud:"required=false,default=cluster.local"`
	ClusterCIDR           string `json:"cluster_cidr" singlecloud:"required=false,default=10.42.0.0/16"`
	ClusterServiceIpRange string `json:"cluster_service_ip_range" singlecloud:"required=false,default=10.43.0.0/16"`
}
