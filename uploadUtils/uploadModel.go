package uploadUtils

type UploadConfig struct {
	NowUse    string     `json:"nowUse"`
	AliYunOss *AliYunOss `json:"aliYunOss"`

	//下面是form签名的限制条件
	IncludeType   []string `json:"includeType"`
	MinUploadSize string   `json:"minUploadSize"`
	MaxUploadSize string   `json:"maxUploadSize"`
}

type AliYunOss struct {
	BucketName string `json:"bucketName"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
}
