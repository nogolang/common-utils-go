package configUtils

import (
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/cockroachdb/errors"
	"github.com/nogolang/common-utils-go/etcdUtils"
	"github.com/spf13/viper"
	etcdClientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type SnowId struct {
	SnowMap map[string]*snowIdStruct

	NowServiceName string

	//程序关闭的时候，可以在平滑里释放租约
	EtcdUtil *etcdUtils.EtcdUtils
}

type snowIdStruct struct {
	Node     *snowflake.Node
	Num      int32
	allNodes []int32
}

// 测试的时候，因为是在idea里测试，idea可能读取不到，需要重新打开所有的idea才行
// 我们可以直接在idea里设置环境变量去测试
func NewSnowIdFromK8sEnv(allConfig *CommonConfig, logger *zap.Logger) *SnowId {
	//开发环境就是为1
	var num int
	if allConfig.IsDev() {
		num = 1
	} else {
		err := viper.BindEnv("POD_NAME")
		if err != nil {
			logger.Fatal("获取POD_NAME失败", zap.Error(err))
			return nil
		}
		podName := viper.GetString("POD_NAME")
		if podName == "" {
			logger.Fatal("获取POD_NAME失败，环境变量为空")
		}
		index := strings.LastIndex(podName, "-")
		numStr := podName[index+1:]
		num, err = strconv.Atoi(numStr)
		if err != nil {
			logger.Fatal("获取POD_NAME失败", zap.Error(err))
			return nil
		}
	}

	var snowId SnowId
	snowId.SnowMap = make(map[string]*snowIdStruct)
	var cfg snowIdStruct
	cfg.Num = int32(num)
	node, err := snowflake.NewNode(int64(num))
	if err != nil {
		logger.Fatal("创建snowflake node失败", zap.Error(err))
		return nil
	}
	cfg.Node = node
	snowId.SnowMap[allConfig.Server.ServerName] = &cfg
	snowId.NowServiceName = allConfig.Server.ServerName
	return &snowId
}

// 从第三方中间件获取唯一node，支持根据指定的key获取
func NewSnowIdFromEtcd(
	etcdClient *etcdClientv3.Client,
	logger *zap.Logger,
	allConfig *CommonConfig) *SnowId {

	// 这种形式，最好在程序结束的时候释放租约，暂时不考虑，直接使用k8s即可
	var snowId SnowId
	snowId.SnowMap = make(map[string]*snowIdStruct)
	util := etcdUtils.NewEtcdUtils(etcdClient, logger)
	snowId.EtcdUtil = util
	for _, key := range allConfig.SnowId.Keys {
		var snowConfig snowIdStruct
		var allNodes = make([]int32, 1024)
		//初始化数组
		for i := 0; i < 1024; i++ {
			allNodes[i] = int32(i)
		}
		snowConfig.allNodes = allNodes
		var nowNum int32 = 0
		var err error
		var ttlSecond int64 = 30

		nowNum, err = util.CreateUniqueNum(snowConfig.allNodes, key, ttlSecond)
		if err != nil {
			log.Fatal("创建etcd kv失败", zap.Error(err))
			return nil
		}
		snowConfig.Num = nowNum
		newNode, err := snowflake.NewNode(int64(nowNum))
		if err != nil {
			log.Fatal("创建snowflake node失败", zap.Error(err))
			return nil
		}
		snowConfig.Node = newNode
		snowId.SnowMap[key] = &snowConfig
	}
	return &snowId
}

func (receiver *SnowId) GetSnowNodeFromNowService() (*snowflake.Node, error) {
	if receiver.SnowMap[receiver.NowServiceName] != nil {
		return receiver.SnowMap[receiver.NowServiceName].Node, nil
	}
	return nil, errors.New("获取snowNode失败,key不存在")
}

func (receiver *SnowId) GetSnowNode(key string) (*snowflake.Node, error) {
	if receiver.SnowMap[key] != nil {
		return receiver.SnowMap[key].Node, nil
	}
	return nil, errors.New("获取snowNode失败,key不存在")
}
func (receiver *SnowId) GetSnowNum(key string) (int32, error) {
	if receiver.SnowMap[key] != nil {
		return receiver.SnowMap[key].Num, nil
	}
	return 0, errors.New("获取snowNum失败,key不存在")
}
func (receiver *SnowId) GetSnowNumFromNowService() (int32, error) {
	if receiver.SnowMap[receiver.NowServiceName] != nil {
		return receiver.SnowMap[receiver.NowServiceName].Num, nil
	}
	return 0, errors.New("获取snowNum失败,key不存在")
}
