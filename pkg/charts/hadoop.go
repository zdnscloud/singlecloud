package charts

type Hadoop struct {
	HDFS        HDFS              `json:"hdfs"`
	Yarn        Yarn              `json:"yarn"`
	Persistence HadoopPersistence `json:"persistence"`
	Zeppelin    HadoopZeppelin    `json:"zeppelin"`
}

type HDFS struct {
	DataNode DataNode `json:"dataNode"`
}

type DataNode struct {
	Replicas int `json:"replicas"`
}

type Yarn struct {
	NodeManager NodeManager `json:"nodeManager"`
}

type NodeManager struct {
	Replicas int `json:"replicas"`
}

type HadoopPersistence struct {
	NameNodePersistence PersistentVolumeConfig `json:"nameNode"`
	DataNodePersistence PersistentVolumeConfig `json:"dataNode"`
}

type HadoopZeppelin struct {
	ZeppelinConfig ZeppelinConfig `json:"zeppelin"`
	SparkConfig    SparkConfig    `json:"spark"`
}

type ZeppelinConfig struct {
	Replicas int `json:"replicas"`
}

type SparkConfig struct {
	DriverMemory   string `json:"driverMemory"`
	ExecutorMemory string `json:"executorMemory"`
	NumExecutors   int    `json:"numExecutors"`
}
