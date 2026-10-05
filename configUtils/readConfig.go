// Package configUtils 配置文件装载工具（纯工具，不认识任何业务配置模型）。
//
// 职责只有一件：把若干个 yaml 读进 viper 的全局单例，并把每个文件里
// `otherConfigPath` 列出的其它配置文件一并合并进来，附带 fsnotify 热更新。
// **它不定义、也不知道任何配置结构体**——业务侧要什么形状的聚合体，
// 由业务侧自己用 viper.Unmarshal 装配（例：本项目 shop-common/commonConf）。
//
// 为什么留在公共库（2026-09-26 回迁）：「一个进程装载多个配置文件、后读覆盖先读」
// 是与业务无关的通用能力，各项目都需要；放在某个业务仓里等于让别的项目
// 抄一份或反向依赖。原先它带着一份 17 段的 CommonConfig 上帝结构体，导致
// 所有引入方都得按它的形状组织 yaml（加一个配置段要改库 + 改全部消费方，耦合方向反了）。
// 2026-09-26 拆分：配置模型下沉到各消费包、聚合模型归业务侧，**本包只留装载能力**。
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

// OtherConfigPathKey 配置文件中「其它配置文件路径列表」这个键的名字。
//
// 原名 commonConfigPath（2026-09-26 改名）：叫「common」有误导——这些文件并不比
// 主文件更"公共"，它们只是**同一个进程要一起装载的其它文件**（公共配置、私有凭据、
// 中间件白名单都可能是它）。改名后语义中性，且与"公共库"不再撞词。
const OtherConfigPathKey = "otherConfigPath"

// ReadConfigInFile 从文件里获取配置，支持多个配置文件（分号分隔）。
//
// 合并策略：按传入顺序依次读入并 MergeConfigMap 到全局 viper，**后读的覆盖先读的**，
// 因此调用方应把「公共/默认」放前、「私有/环境相关」放后。
// 每个文件里 OtherConfigPathKey 列出的文件会在该文件读完之后立即合并。
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

		//配置文件里，可能会有 otherConfigPath 用于引入其他配置文件
		err = mergeOtherConfig(v)
		if err != nil {
			return errors.Wrap(err, "配置文件合并失败")
		}
	}
	return nil
}

// mergeOtherConfig 把 mainConfig 里 otherConfigPath 列出的文件合并进全局 viper
func mergeOtherConfig(mainConfig *viper.Viper) error {
	allOtherConfigPath := mainConfig.GetStringSlice(OtherConfigPathKey)
	for _, cfgPath := range allOtherConfigPath {
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
