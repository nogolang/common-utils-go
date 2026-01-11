package configUtils

import (
	"log"

	"github.com/spf13/viper"
)

// 从文件里获取配置
func GetCommonConfigInFile(configPath string) *AllConfig {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("配置文件格式不正确:", err)
	}
	var allConfig AllConfig
	err = viper.Unmarshal(&allConfig)
	if err != nil {
		log.Fatal("配置文件解析失败:", err)
	}
	return &allConfig
}
