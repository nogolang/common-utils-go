package configUtils

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

//func GetCommonConfig() *CommonConfig {
//	return commonConfig
//}

func GetCommonConfig() *CommonConfig {
	var commonConfig CommonConfig
	err := viper.Unmarshal(&commonConfig)
	if err != nil {
		log.Fatal("配置文件序列化失败:", err)
		return nil
	}
	return &commonConfig
}

// 从文件里获取配置，支持多个配置文件
func ReadConfigInFile(configPath string) error {
	multiConfig := strings.Split(configPath, ";")
	for _, cfgPath := range multiConfig {
		v := viper.New()

		//读取配置文件
		v.SetConfigFile(cfgPath)
		err := v.ReadInConfig()
		if err != nil {
			return errors.Wrap(err, "配置文件读取失败")
		}
		//合并到全局
		err = viper.MergeConfigMap(v.AllSettings())
		if err != nil {
			return errors.Wrap(err, "配置文件合并到全局失败")
		}

		//监听
		v.OnConfigChange(func(in fsnotify.Event) {
			//更新之后，重新合并到全局
			log.Println(in.Name, in.String(), "配置文件更新了")
			err := viper.MergeConfigMap(v.AllSettings())
			if err != nil {
				log.Fatal("更新后 merge到主配置失败:", err)
			}
		})
		v.WatchConfig()

		//配置文件里，可能会有commonConfigPath用于引入其他配置文件
		err = mergeCommonConfig(v)
		if err != nil {
			return errors.Wrap(err, "配置文件合并失败")
		}
	}
	return nil
}

func mergeCommonConfig(mainConfig *viper.Viper) error {
	allCommonConfigPath := mainConfig.GetStringSlice("commonConfigPath")
	for _, cfgPath := range allCommonConfigPath {
		v := viper.New()

		_, err := os.Stat(cfgPath)
		if os.IsNotExist(err) {
			return errors.Wrap(err, fmt.Sprintf("配置文件不存在: %s", cfgPath))
		}
		//读取配置文件，这里的文件路径需要处理，因为我们的其他配置文件，应该是相当于主配置文件路径来说的
		//如果我们主配置文件里写 "./common.yaml"，那么这个相对目录实际上是相当于工作目录来说的
		//而不是主配置文件路径
		v.SetConfigFile(cfgPath)
		err = v.ReadInConfig()
		if err != nil {
			return errors.Wrap(err, "配置文件读取失败")
		}
		//合并到全局
		err = viper.MergeConfigMap(v.AllSettings())
		if err != nil {
			return errors.Wrap(err, "配置文件合并到全局失败")
		}

		//监听
		v.OnConfigChange(func(in fsnotify.Event) {
			//更新之后，重新合并到全局
			log.Println(in.Name, in.String(), "配置文件更新了")
			err := viper.MergeConfigMap(v.AllSettings())
			if err != nil {
				log.Fatal("更新后 merge到主配置失败:", err)
			}
		})
		v.WatchConfig()
	}
	return nil
}
