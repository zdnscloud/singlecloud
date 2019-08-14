package charts

type Redis struct {
	Cluster RedisCluster `json:"cluster"`
	Master  RedisMaster  `json:"master"`
}

type RedisCluster struct {
	SlaveCount int `json:"slaveCount" rest:"required=false,default=1"`
}

type RedisMaster struct {
	Port             int                   `json:"port" rest:"required=false,default=6379"`
	PersistentVolume RedisPersistentVolume `json:"persistence"`
}

type RedisPersistentVolume struct {
	Enabled      bool     `json:"enabled" rest:"required=true"`
	StorageClass string   `json:"storageClass" rest:"required=true,options=cephfs|lvm"`
	AccessModes  []string `json:"accessModes" rest:"required=true,options=ReadWriteMany|ReadWriteOnce"`
}
