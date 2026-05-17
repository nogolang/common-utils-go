package configUtils

import (
	"testing"
)

// 从文件里获取配置，支持多个配置文件
func Test_read_Config(t *testing.T) {
	err := ReadConfigInFile("./testFile/config/dev.yaml")
	if err != nil {
		t.Error(err)
	}
	select {}
}
