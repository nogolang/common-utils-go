package elasticUtils

import (
	"log"
	"os"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

func NewElasticClient(cfg *ElasticConfig) *elasticsearch.TypedClient {
	if cfg == nil {
		return nil
	}
	var esConfig elasticsearch.Config
	if cfg.EnableTls {
		caFile, err := os.ReadFile(cfg.CaCrt)
		if err != nil {
			log.Fatal("读取elastic ca文件失败")
			return nil
		}
		esConfig = elasticsearch.Config{
			Addresses: cfg.Url,
			Username:  cfg.Username,
			Password:  cfg.Password,
			CACert:    caFile,
			Logger: &elastictransport.ColorLogger{
				Output:            os.Stdout,
				EnableRequestBody: true,
			},
		}
	} else {
		esConfig = elasticsearch.Config{
			Addresses: cfg.Url,
			Username:  cfg.Username,
			Password:  cfg.Password,
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
	return client
}
