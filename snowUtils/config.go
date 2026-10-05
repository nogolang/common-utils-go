package snowUtils

// SnowConfig 雪花ID配置
type SnowConfig struct {
	//直接指定节点号，本地开发用；为0视为未设置
	WorkerId int64 `json:"workerId"`
	//从 POD_NAME 解析节点号（K8s StatefulSet 部署用），优先级高于 workerId
	WorkerIdFromPodName bool `json:"workerIdFromPodName"`
}
