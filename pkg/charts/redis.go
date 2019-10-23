package charts

type Redis struct {
	Cluster  RedisCluster `json:"cluster"`
	Password string       `json:"password"`
}

type RedisCluster struct {
	SlaveCount int `json:"slaveCount"`
}
