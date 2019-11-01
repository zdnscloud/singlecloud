package charts

type Kafka struct {
	Replicas    int                    `json:"replicas"`
	External    External               `json:"external"`
	Persistence PersistentVolumeConfig `json:"persistence"`
	Zookeeper   Zookeeper              `json:"zookeeper"`
}

type External struct {
	ClusterDomain string `json:"clusterDomain" rest:"required=true"`
}

type Zookeeper struct {
	Replicas      int                    `json:"replicaCount"`
	ClusterDomain string                 `json:"clusterDomain" rest:"required=true"`
	Persistence   PersistentVolumeConfig `json:"persistence"`
}
