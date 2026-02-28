package uploadUtils

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/nogolang/common-utils-go/configUtils"
	"github.com/pkg/errors"
)

type UploadAliYunOssSvc struct {
	OssClient    *oss.Client
	commonConfig *configUtils.CommonConfig
}

func NewUploadAliYunOss(commonConfig *configUtils.CommonConfig) *UploadAliYunOssSvc {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.
			NewStaticCredentialsProvider(commonConfig.Upload.AliYunOss.AccessKeyId, commonConfig.Upload.AliYunOss.AccessKeySecret)).
		WithRegion(commonConfig.Upload.AliYunOss.Rigion).
		WithEndpoint(commonConfig.Upload.AliYunOss.Endpoint)
	// 创建OSS客户端
	client := oss.NewClient(cfg)

	handler := UploadAliYunOssSvc{
		OssClient:    client,
		commonConfig: commonConfig,
	}
	return &handler
}

type UploadUrlResponse struct {
	Url           string            `json:"url"`
	SignedHeaders map[string]string `json:"signedHeaders"`
}

// UploadPolic用于在form表单上传的时候返回给前台
type UploadPolicyResponse struct {
	AccessKeyId string `json:"ossAccessKeyId"`
	Host        string `json:"host"`
	Signature   string `json:"signature"`
	Policy      string `json:"policy"`
	Key         string `json:"key"`
}

// 这是oss表单上传的配置，可以通过这些配置，生成policy，并且可以用于校验
type uploadOssConfigForSignature struct {
	Expiration string `json:"expiration"`

	//官方的案例有问题，这里应该是使用interface类型，不然指定不了整数型和数组型
	Conditions [][]interface{} `json:"conditions"`
}

// 获取url签名用于传送文件，签名这种方式只能用于后台管理系统，不能用于对外的前面，因为缺少校验
func (receiver *UploadAliYunOssSvc) GetUploadUrl(uploadPath string, expired time.Duration) (*UploadUrlResponse, error) {
	var res UploadUrlResponse
	ext := path.Ext(uploadPath)
	extNoPint := strings.Replace(ext, ".", "", -1)
	result, err := receiver.OssClient.Presign(context.Background(), &oss.PutObjectRequest{
		Bucket:      oss.Ptr(receiver.commonConfig.Upload.AliYunOss.BucketName),
		Key:         oss.Ptr(uploadPath),
		ContentType: oss.Ptr("image/" + extNoPint),
	}, oss.PresignExpires(expired))
	if err != nil {
		return nil, errors.Wrap(err, "获取url签名失败")
	}
	if len(result.SignedHeaders) > 0 {
		//当返回结果包含签名头时，使用签名URL发送Put请求时，需要设置相应的请求头
		res.SignedHeaders = result.SignedHeaders
	}

	res.Url = result.URL
	return &res, nil
}

// 获取url签名，用于预览文件，比如我们阻止了公共访问读
func (receiver *UploadAliYunOssSvc) GetUrlForPreview(uploadPath string, expired time.Duration) (*UploadUrlResponse, error) {
	//去掉最左边的/，因为前端传递的pathname是带有前缀的，比如/upload/xxx.png
	//  而oss里是不需要的，所以我们去掉最左边的
	uploadPath = strings.TrimLeft(uploadPath, "/")
	var res UploadUrlResponse
	result, err := receiver.OssClient.Presign(context.Background(), &oss.GetObjectRequest{
		Bucket: oss.Ptr(receiver.commonConfig.Upload.AliYunOss.BucketName),
		Key:    oss.Ptr(uploadPath),
	}, oss.PresignExpires(expired))
	if err != nil {
		return nil, errors.Wrap(err, "获取url签名失败")
	}
	if len(result.SignedHeaders) > 0 {
		//当返回结果包含签名头时，使用签名URL发送Put请求时，需要设置相应的请求头
		res.SignedHeaders = result.SignedHeaders
	}

	res.Url = result.URL
	return &res, nil
}

// 获取上传文件的form表单，后续如果阻止了公共访问读，那么可以用GetUploadUrlForPreview获取签名去访问
func (receiver *UploadAliYunOssSvc) GetUploadForm(uploadPath string, expiredSecond int64) (*UploadPolicyResponse, error) {
	//设置签名的过期时间,需要ISO8601格式
	expireTime := time.Now().Add(time.Second * time.Duration(expiredSecond)).
		Format("2006-01-02T15:04:05Z")
	var config uploadOssConfigForSignature
	config.Expiration = expireTime

	//starts-with $key product/ 代表必须以product/开头
	//  也就是上传到指定目录
	//  如果是eq，那么就是完整的目录+文件名称，这也是我们需要的
	//注意，这里设置了之后，前端必须要通过form-data携带对应的参数key
	var conditionDir []interface{}
	//conditionDir = append(conditionDir, "starts-with")
	conditionDir = append(conditionDir, "eq")
	conditionDir = append(conditionDir, "$key")
	conditionDir = append(conditionDir, uploadPath)

	//这个是在前台直接传递的，不能进行签名
	//var conditionStatus []interface{}
	//conditionStatus = append(conditionStatus, "eq")
	//conditionStatus = append(conditionStatus, "$success_action_status")
	//conditionStatus = append(conditionStatus, "200")

	//限制上传的类型
	//前端在formData里必须传递x-oss-content-type
	var conditionFileType []interface{}
	conditionFileType = append(conditionFileType, "in")
	conditionFileType = append(conditionFileType, "$content-type")
	//比如 []string{"image/png", "image/jpg", "image/jpeg"}
	conditionFileType = append(conditionFileType, receiver.commonConfig.Upload.IncludeType)

	//限制上传的大小，单位是字节
	var conditionFileSize []interface{}
	conditionFileSize = append(conditionFileSize, "content-length-range")
	conditionFileSize = append(conditionFileSize, receiver.commonConfig.Upload.MinUploadSize)
	conditionFileSize = append(conditionFileSize, receiver.commonConfig.Upload.MaxUploadSize)

	config.Conditions = append(config.Conditions, conditionDir,
		//conditionStatus,
		conditionFileSize,
		conditionFileType)

	//转换成json
	result, err := json.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "转换json失败")
	}
	//转换成base64编码
	encodedResult := base64.StdEncoding.EncodeToString(result)

	//以指定的方式进行hash运算生成签名
	h := hmac.New(sha1.New, []byte(receiver.commonConfig.Upload.AliYunOss.AccessKeySecret))
	_, err = io.WriteString(h, encodedResult)
	if err != nil {
		return nil, errors.Wrap(err, "生成签名失败")
	}
	signedStr := base64.StdEncoding.EncodeToString(h.Sum(nil))
	policyToken := UploadPolicyResponse{
		AccessKeyId: receiver.commonConfig.Upload.AliYunOss.AccessKeyId,
		//Bucket域名的固定格式
		Host:      "https://" + receiver.commonConfig.Upload.AliYunOss.BucketName + "." + receiver.commonConfig.Upload.AliYunOss.Endpoint,
		Signature: signedStr,
		Policy:    encodedResult,
		Key:       uploadPath,
	}
	return &policyToken, nil

}

func (receiver *UploadAliYunOssSvc) IsUrlsExist(urlPath []string) (bool, error) {
	for _, keyPath := range urlPath {
		exist, err := receiver.OssClient.IsObjectExist(context.Background(), receiver.commonConfig.Upload.AliYunOss.BucketName, keyPath)
		if err != nil {
			return false, errors.Wrap(err, "查询文件失败")
		}
		//只要有一个不存在，那么就都失败
		if !exist {
			return false, nil
		}
	}
	return false, nil
}
