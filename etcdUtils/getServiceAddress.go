package etcdUtils

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 这里只获取http前缀的
func GetHttpServiceAddress(etcdClient *clientv3.Client, prefix string, serviceName string) (string, error) {
	res, err := etcdClient.Get(context.TODO(), prefix+"/"+serviceName, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}
	var TempStruct struct {
		Endpoints []string `json:"endpoints"`
	}

	var allServiceAddress []string

	//获取到多个kv服务
	for _, kv := range res.Kvs {
		err := json.Unmarshal(kv.Value, &TempStruct)
		if err != nil {
			return "", err
		}
		for _, address := range TempStruct.Endpoints {
			if strings.HasPrefix(address, "http://") {
				allServiceAddress = append(allServiceAddress, address)
			}
		}
	}
	//随机获取一个
	return selector(allServiceAddress, serviceName)

}
func selector(address []string, serviceName string) (string, error) {
	if len(address) == 0 {
		return "", errors.New(fmt.Sprintf("指定的服务没有启动,[%s]", serviceName))
	}

	if len(address) == 1 {
		return address[0], nil
	}

	//取随机数组，只有2以上即可
	return address[rand.IntN(len(address)-1)], nil
}
