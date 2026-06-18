package logUtils

import (
	"log"
	"os"
	"slices"
	"time"

	"github.com/nogolang/common-utils-go/configUtils"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewZapAtomicLevel(allConfig *configUtils.CommonConfig) *zap.AtomicLevel {
	level := zap.NewAtomicLevel()
	if allConfig.Log == nil {
		level.SetLevel(zapcore.InfoLevel)
		return &level
	}
	switch allConfig.Log.Level {
	case "debug":
		level.SetLevel(zapcore.DebugLevel)
	case "info":
		level.SetLevel(zapcore.InfoLevel)
	case "warn":
		level.SetLevel(zapcore.WarnLevel)
	case "error":
		level.SetLevel(zapcore.ErrorLevel)
	default:
		//为空默认就是info
		level.SetLevel(zapcore.InfoLevel)
	}
	return &level
}

func NewZapConfig(allConfig *configUtils.CommonConfig, level zapcore.Level) *zap.Logger {
	var logger *zap.Logger

	if configUtils.IsDev() {
		//输出日志，向控制台输出，如果设置的是warn，那么info是不会输出的
		devCore := zapcore.NewCore(getEncoding(allConfig), getConsoleWriter(), level)
		//这里不添加本身的日志堆栈信息，但是添加caller信息
		//  因为错误堆栈信息我们会直接输出，而不是用日志堆栈
		//caller是文件信息，大部分时候用不到，因为放中间件，小部分要直接打印用
		logger = zap.New(devCore,
			zap.AddCaller(),
		)
	} else {
		//生产环境,直接输出到文件
		//文件则要分为error和info
		//至于控制台,我们输出error即可,方便pod里查看
		//  pod不能放info,不然会全是无意义的内容,会撑爆容器volume
		prodCoreConsole := zapcore.NewCore(getEncoding(allConfig), getConsoleWriter(), zapcore.ErrorLevel)
		prodFileInfo := zapcore.NewCore(getEncoding(allConfig), getLogWriterAll(), level)
		prodFileError := zapcore.NewCore(getEncoding(allConfig), getLogWriterError(), zapcore.ErrorLevel)
		logger = zap.New(zapcore.NewTee(prodCoreConsole, prodFileInfo, prodFileError), zap.AddCaller())
	}
	//这里使用了wire，严格准守di原则
	//但是有些地方可能不太方便传递logger对象，比如中间件的地方使用全局的也可以
	zap.ReplaceGlobals(logger)
	return logger
}

func getEncoding(common *configUtils.CommonConfig) zapcore.Encoder {
	var newEncoder zapcore.Encoder
	encodeTime := func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(t.Format(time.DateTime))
	}
	if configUtils.IsDev() {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = encodeTime
		newEncoder = zapcore.NewConsoleEncoder(config)

		//进行脱敏
		newEncoder = &SanitizingEncoder{newEncoder, common.Log.HiddenField}
	} else {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = encodeTime
		newEncoder = zapcore.NewJSONEncoder(config)

		//进行脱敏
		newEncoder = &SanitizingEncoder{newEncoder, common.Log.HiddenField}
	}
	return newEncoder
}

func getConsoleWriter() zapcore.WriteSyncer {
	//开发环境向控制台输出info和error
	return zapcore.AddSync(os.Stdout)
}

func getLogWriterAll() zapcore.WriteSyncer {
	return zapcore.AddSync(lumberJackAll())
}
func getLogWriterError() zapcore.WriteSyncer {
	return zapcore.AddSync(lumberJackError())
}

// 日志切割
func lumberJackAll() *lumberjack.Logger {
	//获取项目目录，如果本目录下logs目录不存在
	//就在当前项目运行目录下创建logs目录
	dir, _ := os.Getwd()
	dir = dir + "/logs"

	//判断有没有logs目录
	_, err := os.ReadDir(dir)
	if err != nil {
		//目录不存在，则创建
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			log.Fatal("logs目录创建失败，请手动创建")
			return nil
		}
	}

	fileExt := ".all.log"

	//统一输出到app.log里，暂时不区分error和info
	//到时候再统一处理日志
	fileName := dir + "/app" + fileExt

	return &lumberjack.Logger{
		Filename: fileName,

		//日志文件的最大尺寸,单位MB
		//切割出来的每个文件都是xMB,但是最开始的主文件可能会小一点
		MaxSize: 10,

		//保留的旧的最大个数.此时我们输出了10MB的内容.
		//但是只有5个切割文件+1个主文件.其余5个都删掉了.按照切割出来的日期.早期的会优先进行删除
		//如果旧日志一直没有删除(没有满5个).但是已经过去30天了.这时候会自动删除
		MaxBackups: 5,

		//保留旧文件的最大天数
		MaxAge: 15,

		//是否压缩旧文件
		Compress: true,
	}
}
func lumberJackError() *lumberjack.Logger {
	//获取项目目录，如果本目录下logs目录不存在
	//就在当前项目运行目录下创建logs目录
	dir, _ := os.Getwd()
	dir = dir + "/logs"

	//判断有没有logs目录
	_, err := os.ReadDir(dir)
	if err != nil {
		//目录不存在，则创建
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			log.Fatal("logs目录创建失败，请手动创建")
			return nil
		}
	}

	fileExt := ".error.log"

	//统一输出到app.log里，暂时不区分error和info
	//到时候再统一处理日志
	fileName := dir + "/app" + fileExt

	return &lumberjack.Logger{
		Filename: fileName,
		//日志文件的最大尺寸,单位MB
		MaxSize: 10,

		//保留的旧的最大个数
		MaxBackups: 10,

		//保留旧文件的最大天数
		MaxAge: 30,

		//是否压缩旧文件
		Compress: false,
	}
}

// 自定义脱敏编码器 - 更简单直观的方式
type SanitizingEncoder struct {
	zapcore.Encoder
	SensitiveFields []string
}

func (receiver *SanitizingEncoder) Clone() zapcore.Encoder {
	return &SanitizingEncoder{
		Encoder: receiver.Encoder.Clone(),
	}
}

func (receiver *SanitizingEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// 在这里脱敏，然后调用原始编码器
	sanitizedFields := make([]zapcore.Field, len(fields))
	for i, field := range fields {
		if slices.Contains(receiver.SensitiveFields, field.Key) {
			sanitizedFields[i] = zap.String(field.Key, "***MASKED***")
		} else {
			sanitizedFields[i] = field
		}
	}
	return receiver.Encoder.EncodeEntry(entry, sanitizedFields)
}
