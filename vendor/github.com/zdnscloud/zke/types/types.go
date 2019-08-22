package types

type DNSConfig struct {
	Provider            string            `yaml:"provider" json:"provider"`
	UpstreamNameservers []string          `yaml:"upstreamnameservers" json:"upstreamnameservers"`
	ReverseCIDRs        []string          `yaml:"reversecidrs" json:"reversecidrs"`
	NodeSelector        map[string]string `yaml:"node_selector" json:"nodeSelector"`
}

type IngressConfig struct {
	Provider     string            `yaml:"provider" json:"provider"`
	Options      map[string]string `yaml:"options" json:"options"`
	NodeSelector map[string]string `yaml:"node_selector" json:"nodeSelector"`
	ExtraArgs    map[string]string `yaml:"extra_args" json:"extraArgs"`
}
type PrivateRegistry struct {
	URL      string `yaml:"url" json:"url"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	CAcert   string `yaml:"ca_cert" json:"caCert"`
}

type ZKENodePlan struct {
	Address string `json:"address,omitempty"`
	// map of named processes that should run on the node
	Processes   map[string]Process `json:"processes,omitempty"`
	PortChecks  []PortCheck        `json:"portChecks,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
	Labels      map[string]string  `json:"labels,omitempty"`
}

type Process struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
	Image   string   `json:"image"`
	//AuthConfig for image private registry
	ImageRegistryAuthConfig string `json:"imageRegistryAuthConfig"`
	// Process docker image VolumesFrom
	VolumesFrom []string `json:"volumesFrom"`
	// Process docker container bind mounts
	Binds         []string    `json:"binds"`
	NetworkMode   string      `json:"networkMode"`
	RestartPolicy string      `json:"restartPolicy"`
	PidMode       string      `json:"pidMode"`
	Privileged    bool        `json:"privileged"`
	HealthCheck   HealthCheck `json:"healthCheck"`
	// Process docker container Labels
	Labels map[string]string `json:"labels"`
	// Process docker publish container's port to host
	Publish []string `json:"publish"`
}

type HealthCheck struct {
	URL string `json:"url"`
}

type PortCheck struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type KubernetesServicesOptions struct {
	KubeAPI        map[string]string `json:"kubeapi"`
	Kubelet        map[string]string `json:"kubelet"`
	Kubeproxy      map[string]string `json:"kubeproxy"`
	KubeController map[string]string `json:"kubeController"`
	Scheduler      map[string]string `json:"scheduler"`
}

type ZKEConfigMonitor struct {
	MetricsProvider string            `yaml:"metrics_provider" json:"metricsProvider"`
	MetricsOptions  map[string]string `yaml:"metrics_options" json:"metricsOptions"`
}
