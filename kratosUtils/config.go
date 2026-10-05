package kratosUtils

// K8sConfig k8s 部署相关配置
type K8sConfig struct {
	//是否注册到 k8s 服务注册表；false 时返回空 registry，不注册（本地开发用）
	Register bool `json:"register"`
}
