package globaldns

type EdgeNode struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type FullState struct {
	DesiredState State `json:"desiredState,omitempty"`
}

type State struct {
	ZKEConfig ZcloudKubernetesEngineConfig `json:"zkeConfig,omitempty"`
}

type ZcloudKubernetesEngineConfig struct {
	Services ZKEConfigServices `json:"services,omitempty"`
}

type ZKEConfigServices struct {
	Kubelet KubeletService `json:"kubelet,omitempty"`
}

type KubeletService struct {
	ClusterDomain string `json:"clusterDomain,omitempty"`
}
