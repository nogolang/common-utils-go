package redisUtils

import (
	"context"
	"log"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/nogolang/common-utils-go/configUtils"
)
import "github.com/redis/go-redis/v9"

// 集群连接
func NewRedisClusterClient(allConfig *configUtils.CommonConfig) *redis.ClusterClient {
	var redisDB *redis.ClusterClient

	//如果集群
	if !allConfig.Redis.Single {
		log.Println("当前启动的是redis集群模式")

		if len(allConfig.Redis.ClusterUrl) == 0 {
			log.Fatal("未配置redis集群链接")
			return nil
		}

		//初始化链接,内部自带了链接池
		redisDB = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    allConfig.Redis.ClusterUrl,
			Username: allConfig.Redis.Username,
			Password: allConfig.Redis.Password,
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
	//@TODO 需要添加一些判断，不然报空制作，直接退出
	if redisDB == nil {
		log.Fatal("redis初始化错误")
		return nil
	}

	ping := redisDB.Ping(context.Background())
	if ping.Err() != nil {
		log.Fatal("redis连接错误", ping.Err())
		return nil
	}
	return redisDB

	return nil
}

// 单机连接
func NewRedisClient(allConfig *configUtils.CommonConfig) *redis.Client {
	var redisDB *redis.Client

	//如果单机
	if allConfig.Redis.Single {
		log.Println("当前启动的是redis单机模式")

		if allConfig.Redis.SingleUrl == "" {
			log.Fatal("未配置redis单机链接")
			return nil
		}

		//初始化链接,内部自带了链接池
		redisDB = redis.NewClient(&redis.Options{
			Addr:     allConfig.Redis.SingleUrl,
			Username: allConfig.Redis.Username,
			Password: allConfig.Redis.Password,
			DB:       allConfig.Redis.Db,
			//最大连接数量,默认是10，没有初始连接数，
			//看样子它的初始连接数也是动态调整的
			PoolSize: 100,

			//等待连接超时时间,默认是1s
			PoolTimeout: time.Second * 3,

			//最小空闲连接数,在线程池里的最小空闲连接数，默认0,不限制
			MinIdleConns: 0,

			//最大空闲连接数,在线程池里的最小空闲连接数，默认是0,不限制
			MaxIdleConns: 0,

			//最大空闲时间,超过最大空闲时间后，连接便删除
			//默认30分钟,-1为禁用
			ConnMaxIdleTime: time.Minute * 30,
		})
		//@TODO 需要添加一些判断，不然报空制作，直接退出
		if redisDB == nil {
			log.Fatal("redis初始化错误")
			return nil
		}

		ping := redisDB.Ping(context.Background())
		if ping.Err() != nil {
			log.Fatal("redis连接错误", ping.Err())
			return nil
		}
		return redisDB
	}

	return nil
}

// 创建分布式锁
func NewRedisSyncSingle(redisDb *redis.Client) *redsync.Redsync {
	// 创建redisSync的redis连接池对象
	var pool = goredis.NewPool(redisDb)
	// 从连接池里获取锁对象
	var rsync = redsync.New(pool)
	return rsync
}
func NewRedisSyncCluster(redisDb *redis.ClusterClient) *redsync.Redsync {
	// 创建redisSync的redis连接池对象
	var pool = goredis.NewPool(redisDb)
	// 从连接池里获取锁对象
	var rsync = redsync.New(pool)
	return rsync
}
