package charts

type Redis struct {
	Cluster  RedisCluster `json:"cluster"`
	Master   RedisMaster  `json:"master"`
	Password string       `json:"password" rest:"required=true"`
}

type RedisCluster struct {
	SlaveCount int `json:"slaveCount"`
}

type RedisMaster struct {
	Persistence PersistentVolumeConfig `json:"persistence"`
}

type PersistentVolumeConfig struct {
	StorageClass string `json:"storageClass" rest:"required=true,options=lvm|cephfs"`
	Size         string `json:"size"`
}
