package configUtils

import (
	"log"
	"os"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

func NewElasticClient(allConfig *AllConfig) *elasticsearch.TypedClient {
	//需要把configs/certs目录的证书给复制过来
	//如果开启了安全认证的话
	//file, err := os.ReadFile("../files/http_ca.crt")
	//if err != nil {
	//	log.Fatal(err)
	//	return nil
	//}
	client, err := elasticsearch.NewTypedClient(
		elasticsearch.Config{
			Addresses: allConfig.Elastic.Url,
			//Username: "elastic",
			//Password: "elastic",
			//CACert:   file,
			Logger: &elastictransport.ColorLogger{
				Output:            os.Stdout,
				EnableRequestBody: true,
			},
		})
	if err != nil {
		log.Fatal("连接es失败", zap.Error(err))
		return nil
	}

	log.Panicln("连接es成功")
	return client
}
