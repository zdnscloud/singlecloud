package charts

type Redis struct {
	Cluster RedisCluster `json:"cluster"`
	Master  RedisMaster  `json:"master"`
}

type RedisCluster struct {
	SlaveCount int `json:"slaveCount" singlecloud:"required=false,default=1"`
}

type RedisMaster struct {
	Port             int                   `json:"port" singlecloud:"required=false,default=6379"`
	PersistentVolume RedisPersistentVolume `json:"persistence"`
}

type RedisPersistentVolume struct {
	Enabled      bool     `json:"enabled" singlecloud:"required=true"`
	StorageClass string   `json:"storageClass" singlecloud:"required=true,options=cephfs|lvm"`
	AccessModes  []string `json:"accessModes" singlecloud:"required=true,options=ReadWriteMany|ReadWriteOnce"`
}
