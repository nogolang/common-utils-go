package configUtils

import (
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 从文件里获取配置，支持多个配置文件
func ReadConfigInFile(configPath string) error {
	multiConfig := strings.Split(configPath, ";")
	for _, cfgPath := range multiConfig {
		v := viper.New()

		//读取多个主配置文件
		v.SetConfigFile(cfgPath)
		err := v.ReadInConfig()
		if err != nil {
			log.Fatal("配置文件读取失败:", err)
		}
		v.OnConfigChange(func(in fsnotify.Event) {
			log.Println(in.Name, in.String(), "配置文件更新了")
		})
		v.WatchConfig()

		//每个主配置文件里，可能又会有commonConfigPath
		//  但是所有的配置文件，都要合并到主配置文件中
		mergeAllConfig(v)

		//然后把合并后的实例，再合并到全局中
		var temp map[string]any
		err = v.Unmarshal(&temp)
		if err != nil {
			log.Fatal("配置文件序列化失败:", err)
		}
		err = viper.MergeConfigMap(temp)
		if err != nil {
			log.Fatal("配置文件合并失败:", err)
		}

	}

	return nil
}

func mergeAllConfig(mainConfig *viper.Viper) {
	//读取common配置文件
	allCommonConfigPath := mainConfig.GetStringSlice("commonConfigPath")
	for _, cfgPath := range allCommonConfigPath {
		v := viper.New()
		v.SetConfigFile(cfgPath)
		err := v.ReadInConfig()
		if err != nil {
			log.Fatal("配置文件读取失败:", err)
		}
		//监听配置文件,会自动重写读取进内存
		//  每次修改单个文件，单个文件可能会触发2次OnConfigChange
		//  注意，和循环无关，因为我们是独立的实例
		v.OnConfigChange(func(in fsnotify.Event) {
			log.Println(in.Name, in.String(), "配置文件更新了")
		})
		v.WatchConfig()

		var temp map[string]any
		err = v.Unmarshal(&temp)
		if err != nil {
			log.Fatal("配置文件序列化失败:", err)
		}
		err = mainConfig.MergeConfigMap(temp)
		if err != nil {
			log.Fatal("配置文件合并失败:", err)
		}
	}
}

func GetCommonConfig() *CommonConfig {
	var commonConfig CommonConfig
	err := viper.Unmarshal(&commonConfig)
	if err != nil {
		log.Fatal("配置文件序列化失败:", err)
	}
	return &commonConfig
}
