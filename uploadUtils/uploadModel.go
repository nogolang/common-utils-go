package uploadUtils

// 上传的通用配置
type UploadConfig struct {
	NowUse    string     `json:"nowUse"`
	AliYunOss *AliYunOss `json:"aliYunOss"`
}

type AliYunOss struct {
	BucketName string `json:"bucketName"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
}

// AliYunAccount 阿里云账户凭证
type AliYunAccount struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
}

// 限制
type UploadLimit struct {
	IncludeType   []string `json:"includeType"`
	MinUploadSize string   `json:"minUploadSize"`
	MaxUploadSize string   `json:"maxUploadSize"`
}
