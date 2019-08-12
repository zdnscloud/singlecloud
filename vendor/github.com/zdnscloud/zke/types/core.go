package types

type BaseService struct {
	// Docker image of the service
	Image string `yaml:"image" json:"image"`
	// Extra arguments that are added to the services
	ExtraArgs map[string]string `yaml:"extra_args" json:"extraArgs"`
	// Extra binds added to the nodes
	ExtraBinds []string `yaml:"extra_binds" json:"extraBinds"`
	// this is to provide extra env variable to the docker container running kubernetes service
	ExtraEnv []string `yaml:"extra_env" json:"extraEnv"`
}

type ETCDService struct {
	BaseService `yaml:",inline" json:",inline"`
	// List of etcd urls
	ExternalURLs []string `yaml:"external_urls" json:"externalUrls"`
	CACert       string   `yaml:"ca_cert" json:"caCert"`
	Cert         string   `yaml:"cert" json:"cert"`
	Key          string   `yaml:"key" json:"key"`
	// External etcd prefix
	Path string `yaml:"path" json:"path"`
	// Etcd Recurring snapshot Service
	Snapshot *bool `yaml:"snapshot" json:"snapshot"`
	// Etcd snapshot Retention period
	Retention string `yaml:"retention" json:"retention"`
	// Etcd snapshot Creation period
	Creation string `yaml:"creation" json:"creation"`
	// Backup backend for etcd snapshots, used by zke only
	BackupConfig *BackupConfig `yaml:"backup_config" json:"backupConfig"`
}

type KubeAPIService struct {
	BaseService           `yaml:",inline" json:",inline"`
	ServiceClusterIPRange string `yaml:"service_cluster_ip_range" json:"serviceClusterIpRange"`
	// Port range for services defined with NodePort type
	ServiceNodePortRange string `yaml:"service_node_port_range" json:"serviceNodePortRange""`
	// Enabled/Disable PodSecurityPolicy
	PodSecurityPolicy bool `yaml:"pod_security_policy" json:"podSecurityPolicy"`
	// Enable/Disable AlwaysPullImages admissions plugin
	AlwaysPullImages bool `yaml:"always_pull_images" json:"always_pull_images"`
}

type KubeControllerService struct {
	BaseService           `yaml:",inline" json:",inline"`
	ClusterCIDR           string `yaml:"cluster_cidr" json:"clusterCidr"`
	ServiceClusterIPRange string `yaml:"service_cluster_ip_range" json:"serviceClusterIpRange"`
}

type KubeletService struct {
	BaseService         `yaml:",inline" json:",inline"`
	ClusterDomain       string `yaml:"cluster_domain" json:"clusterDomain"`
	InfraContainerImage string `yaml:"infra_container_image" json:"infraContainerImage"`
	ClusterDNSServer    string `yaml:"cluster_dns_server" json:"clusterDnsServer"`
	FailSwapOn          bool   `yaml:"fail_swap_on" json:"failSwapOn"`
}

type KubeproxyService struct {
	BaseService `yaml:",inline" json:",inline"`
}

type SchedulerService struct {
	BaseService `yaml:",inline" json:",inline"`
}

type BackupConfig struct {
	IntervalHours int `yaml:"interval_hours" json:"intervalHours"`
	// Number of backups to keep
	Retention int `yaml:"retention" json:"retention"`
}

type ZKEConfigAuthn struct {
	Strategy string `yaml:"strategy" json:"strategy"`
	// List of additional hostnames and IPs to include in the api server PKI cert
	SANs []string `yaml:"sans" json:"sans"`
	// Webhook configuration options
	Webhook *AuthWebhookConfig `yaml:"webhook" json:"webhook"`
}

type ZKEConfigAuthz struct {
	Mode    string            `yaml:"mode" json:"mode"`
	Options map[string]string `yaml:"options" json:"options"`
}

type AuthWebhookConfig struct {
	// ConfigFile is a multiline string that represent a custom webhook config file
	ConfigFile string `yaml:"config_file" json:"configFile"`
	// CacheTimeout controls how long to cache authentication decisions
	CacheTimeout string `yaml:"cache_timeout" json:"cacheTimeout"`
}
