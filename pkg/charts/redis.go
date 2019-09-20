package charts

type Redis struct {
	Cluster RedisCluster `json:"cluster"`
	Master  RedisMaster  `json:"master"`
}

type RedisCluster struct {
	SlaveCount int `json:"slaveCount"`
}

type RedisMaster struct {
	Port             int                   `json:"port"`
	PersistentVolume RedisPersistentVolume `json:"persistence"`
}

type RedisPersistentVolume struct {
	Enabled bool `json:"enabled" rest:"required=true"`
}
