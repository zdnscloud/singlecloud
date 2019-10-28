package charts

type Spark struct {
	Master   SparkMaster  `json:"master"`
	Worker   SparkWorker  `json:"worker"`
	Zeppelin SparkZepplin `json:"zeppelin"`
}

type SparkMaster struct {
	Replicas     int    `json:"replicas"`
	DaemonMemory string `json:"daemonMemory"`
}

type SparkWorker struct {
	Replicas       int    `json:"replicas"`
	DaemonMemory   string `json:"daemonMemory"`
	ExecutorMemory string `json:"executorMemory"`
}

type SparkZepplin struct {
	Replicas    int              `json:"replicas"`
	Persistence SparkPersistence `json:"persistence"`
}

type SparkPersistence struct {
	ConfigPersistence   PersistentVolumeConfig `json:"config"`
	NotebookPersistence PersistentVolumeConfig `json:"notebook"`
}
