package charts

type Postgresql struct {
	Replication        Replication            `json:"replication"`
	PostgresqlUsername string                 `json:"postgresqlUsername" rest:"required=true"`
	PostgresqlPassword string                 `json:"postgresqlPassword" rest:"required=true"`
	PostgresqlDatabase string                 `json:"postgresqlDatabase" rest:"required=true"`
	Persistence        PersistentVolumeConfig `json:"persistence"`
}

type Replication struct {
	SlaveReplicas int `json:"slaveReplicas"`
}
