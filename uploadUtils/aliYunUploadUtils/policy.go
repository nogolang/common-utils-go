package aliYunUploadUtils

type UploadUrlResponse struct {
	Url           string            `json:"url"`
	SignedHeaders map[string]string `json:"signedHeaders"`
}

// UploadPolic用于在form表单上传的时候返回给前台
type UploadPolicyResponse struct {
	OssAccessKeyId string `json:"ossAccessKeyId"`
	Host           string `json:"host"`
	Signature      string `json:"signature"`
	Policy         string `json:"policy"`
	Key            string `json:"key"`
}
