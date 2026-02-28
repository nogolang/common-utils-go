package configUtils

import (
	"log"
	"os"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

func NewElasticClient(allConfig *CommonConfig, logger *zap.Logger) *elasticsearch.TypedClient {
	if allConfig.Elastic == nil {
		return nil
	}
	var esConfig elasticsearch.Config
	if allConfig.Elastic.EnableTls {
		caFile, err := os.ReadFile(allConfig.Elastic.CaCrt)
		if err != nil {
			logger.Sugar().Fatal("读取elastic ca文件失败")
			return nil
		}
		esConfig = elasticsearch.Config{
			Addresses: allConfig.Elastic.Url,
			Username:  allConfig.Elastic.Username,
			Password:  allConfig.Elastic.Password,
			CACert:    caFile,
			Logger: &elastictransport.ColorLogger{
				Output:            os.Stdout,
				EnableRequestBody: true,
			},
		}
	} else {
		esConfig = elasticsearch.Config{
			Addresses: allConfig.Elastic.Url,
			Username:  allConfig.Elastic.Username,
			Password:  allConfig.Elastic.Password,
			Logger: &elastictransport.ColorLogger{
				Output:            os.Stdout,
				EnableRequestBody: true,
			},
		}
	}
	client, err := elasticsearch.NewTypedClient(esConfig)
	if err != nil {
		log.Fatal("连接es失败", zap.Error(err))
		return nil
	}

	log.Println("连接es成功")
	return client
}
