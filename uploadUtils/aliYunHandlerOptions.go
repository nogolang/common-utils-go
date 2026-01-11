package uploadUtils

type option func(*UploadAliYunOssHandler)

func AliYun_WithEndpoint(endpoint string) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.Endpoint = endpoint
	}
}

func AliYun_WithBucketName(bucketName string) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.BucketName = bucketName
	}
}

func AliYun_WithAccessKeyId(accessKeyId string) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.AccessKeyId = accessKeyId
	}
}
func AliYun_WithAccessKeySecret(accessKeySecret string) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.AccessKeySecret = accessKeySecret
	}
}
func AliYun_WithRegion(regin string) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.Regin = regin
	}
}
func AliYun_WithIncludeType(includeType []string) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.IncludeType = includeType
	}
}
func AliYun_WithMinUploadSize(minUploadSize int64) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.MinUploadSize = minUploadSize
	}
}
func AliYun_WithMaxUploadSize(maxUploadSize int64) option {
	return func(handler *UploadAliYunOssHandler) {
		handler.MaxUploadSize = maxUploadSize
	}
}
