package uploadUtils

// UploadConfig 上传通道的【连接】配置（yaml 配置文件维护）。
//
// 2026-09-18 拆分（业务方定调）：大小/类型这类【业务限制】不再放在公共库配置模型里——
// 公共库不承载业务限制代码；IncludeType/MinUploadSize/MaxUploadSize 三字段已删除，
// 限制参数改由业务侧（shop-server 的 sys_business_config code=upload.image）维护，
// 调用 ossUtil.GetUploadForm 时以参数传入（见 ossUtil.UploadFormLimit）。
type UploadConfig struct {
	NowUse    string     `json:"nowUse"`
	AliYunOss *AliYunOss `json:"aliYunOss"`
	// PublicKeySalt 公共读**固定文件名**的加盐值（"同 dataId 覆盖旧图"语义用）。
	//
	// 2026-09-26：从「借用 jwt.secret」解耦出来。原先盐取自 JWT 签名密钥，于是
	// 「改 JWT 密钥」会连带改掉所有品牌 Logo / 广告位 / 首页图的固定文件名（历史文件成孤儿），
	// 而这两件事本无关系。独立成配置后，改密钥不再影响任何文件路径。
	// **留空 = 用包内默认常量**（多实例天然一致即可，不必配）。
	PublicKeySalt string `json:"publicKeySalt"`
}

type AliYunOss struct {
	BucketName string `json:"bucketName"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
}
