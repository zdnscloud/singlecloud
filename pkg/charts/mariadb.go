package charts

type Mariadb struct {
	RootUser    MariadbRootUser    `json:"rootUser"`
	DB          MaridbDataBase     `json:"db"`
	Replication MariadbReplication `json:"replication"`
	Master      MariadbMaster      `json:"master"`
	Slave       MariadbSlave       `json:"slave"`
	Service     MariadbService     `json:"service"`
}

type MariadbRootUser struct {
	Password string `json:"password"`
}

type MaridbDataBase struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type MariadbReplication struct {
	Password string `json:"password"`
}

type MariadbMaster struct {
	Persistence MariadbPersistence `json:"persistence"`
}

type MariadbPersistence struct {
	StorageClass string `json:"storageclass" rest:"options=cephfs|lvm"`
	Size         string `json:"size"`
}

type MariadbSlave struct {
	Replicas    int                `json:"replicas"`
	Persistence MariadbPersistence `json:"persistence"`
}

type MariadbService struct {
	Type string `json:"type" rest:"options=ClusterIP|NodePort"`
}
