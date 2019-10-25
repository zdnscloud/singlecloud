package charts

type Mariadb struct {
	RootUser    MariadbRootUser    `json:"rootUser" rest:"required=true"`
	Slave       MariadbSlave       `json:"slave"`
	Persistence MariadbPersistence `json:"persistence"`
	Service     MariadbService     `json:"service"`
}

type MariadbRootUser struct {
	Password string `json:"password" rest:"required=true"`
}

type MariadbPersistence struct {
	StorageClass string `json:"storageclass" rest:"options=cephfs|lvm"`
	Size         int    `json:"size" rest:"min=1,max=300"`
}

type MariadbSlave struct {
	Replicas int `json:"replicas" rest:"min=1,max=9"`
}

type MariadbService struct {
	Type string `json:"type" rest:"options=ClusterIP|NodePort"`
}
