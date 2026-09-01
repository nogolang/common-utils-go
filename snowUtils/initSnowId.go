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
func NewSnowIdFromK8sEnv(allConfig *configUtils.CommonConfig) *snowflake.Node {
	var num int
	if allConfig.Snow != nil && allConfig.Snow.WorkerIdFromPodName {
		//K8s StatefulSet 部署，从 POD_NAME 后缀解析节点号
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
	} else if allConfig.Snow != nil && allConfig.Snow.WorkerId != 0 {
		//直接指定节点号，本地开发用
		num = int(allConfig.Snow.WorkerId)
	} else {
		//未配置时默认节点号为1
		num = 1
	}

	node, err := snowflake.NewNode(int64(num))
	if err != nil {
		log.Fatal("创建snowflake node失败", zap.Error(err))
		return nil
	}
	return node
}
