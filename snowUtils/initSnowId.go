package snowUtils

import (
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/nogolang/common-utils-go/configUtils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// 测试的时候，因为是在idea里测试，idea可能读取不到，需要重新打开所有的idea才行
// 我们可以直接在idea里设置环境变量去测试
func NewSnowIdFromK8sEnv() *snowflake.Node {
	//开发环境就是为1
	var num int
	if configUtils.IsDev() {
		num = 1
	} else {
		err := viper.BindEnv("POD_NAME")
		if err != nil {
			log.Fatal("获取POD_NAME失败", zap.Error(err))
			return nil
		}
		podName := viper.GetString("POD_NAME")
		if podName == "" {
			log.Fatal("获取POD_NAME失败，环境变量为空")
		}
		index := strings.LastIndex(podName, "-")
		numStr := podName[index+1:]
		num, err = strconv.Atoi(numStr)
		if err != nil {
			log.Fatal("获取POD_NAME失败", zap.Error(err))
			return nil
		}
	}

	node, err := snowflake.NewNode(int64(num))
	if err != nil {
		log.Fatal("创建snowflake node失败", zap.Error(err))
		return nil
	}
	return node
}
