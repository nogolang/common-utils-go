package configUtils

import (
	"log"
	"time"

	"go.uber.org/zap"
)
import "github.com/redis/go-redis/v9"

// 集群连接
func NewRedisClusterClient(allConfig *AllConfig) *redis.ClusterClient {
	var redisDB *redis.ClusterClient

	//如果集群
	if !allConfig.Redis.IsSingle {
		log.Println("当前启动的是redis集群模式")

		if len(allConfig.Redis.ClusterUrl) == 0 {
			log.Fatal("未配置redis集群链接")
			return nil
		}

		//初始化链接,内部自带了链接池
		redisDB = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    allConfig.Redis.ClusterUrl,
			Password: "",
			//最大连接数量,默认是10，没有初始连接数，
			//看样子它的初始连接数也是动态调整的
			PoolSize: 100,

			//等待连接超时时间,默认是1s
			PoolTimeout: time.Second,

			//最小空闲连接数,在线程池里的最小空闲连接数，默认0,不限制
			MinIdleConns: 0,

			//最大空闲连接数,在线程池里的最小空闲连接数，默认是0,不限制
			MaxIdleConns: 0,

			//最大空闲时间,超过最大空闲时间后，连接便删除
			//默认30分钟,-1为禁用
			ConnMaxIdleTime: time.Minute * 30,
		})
		return redisDB
	}

	return nil
}

// 单机连接
func NewRedisClient(allConfig *AllConfig, logger *zap.Logger) *redis.Client {
	var redisDB *redis.Client

	//如果单机
	if allConfig.Redis.IsSingle {
		logger.Info("当前启动的是redis单机模式")

		if allConfig.Redis.SingleUrl == "" {
			logger.Fatal("未配置redis单机链接")
			return nil
		}

		//初始化链接,内部自带了链接池
		redisDB = redis.NewClient(&redis.Options{
			Addr:     allConfig.Redis.SingleUrl,
			Password: "",
			DB:       0,
			//最大连接数量,默认是10，没有初始连接数，
			//看样子它的初始连接数也是动态调整的
			PoolSize: 100,

			//等待连接超时时间,默认是1s
			PoolTimeout: time.Second,

			//最小空闲连接数,在线程池里的最小空闲连接数，默认0,不限制
			MinIdleConns: 0,

			//最大空闲连接数,在线程池里的最小空闲连接数，默认是0,不限制
			MaxIdleConns: 0,

			//最大空闲时间,超过最大空闲时间后，连接便删除
			//默认30分钟,-1为禁用
			ConnMaxIdleTime: time.Minute * 30,
		})

		return redisDB

	}

	return nil
}
