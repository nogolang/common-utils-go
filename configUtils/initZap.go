package configUtils

import (
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewZapConfig(allConfig *AllConfig) *zap.Logger {
	var logger *zap.Logger
	var level zapcore.Level
	switch allConfig.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		//为空默认就是info
		level = zapcore.InfoLevel
	}

	if allConfig.IsDev() {
		//输出日志，向控制台输出，如果设置的是warn，那么info是不会输出的
		devCore := zapcore.NewCore(getEncoding(allConfig), getConsoleWriter(), level)
		//这里不添加本身的日志堆栈信息，而是输出错误堆栈信息，因为我们的日志是放到中间件的
		logger = zap.New(devCore)
	} else {
		//输出日志，向文件输出，这里设置了多个级别
		//  all里默认输出所有，但还要看我们本身的设置，如果设置warn，
		//    那么info是不会输出的，只会输出warn到all文件里
		//  但是error，fatal等是固定向error文件输出的
		prodCoreAll := zapcore.NewCore(getEncoding(allConfig), getLogWriterAll(), level)
		prodCoreError := zapcore.NewCore(getEncoding(allConfig), getLogWriterError(), zapcore.ErrorLevel)
		logger = zap.New(zapcore.NewTee(prodCoreAll, prodCoreError))
	}
	//这里使用了wire，严格准守di原则
	//但是有些地方可能不太方便传递logger对象，比如中间件的地方使用全局的也可以
	zap.ReplaceGlobals(logger)
	return logger
}

func getEncoding(allConfig *AllConfig) zapcore.Encoder {
	var newEncoder zapcore.Encoder
	encodeTime := func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(t.Format(time.DateTime))
	}
	if allConfig.IsDev() {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = encodeTime
		newEncoder = zapcore.NewConsoleEncoder(config)
	} else {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = encodeTime
		newEncoder = zapcore.NewJSONEncoder(config)
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
		MaxAge: 30,

		//是否压缩旧文件
		Compress: false,
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
		//切割出来的每个文件都是xMB,但是最开始的主文件可能会小一点
		MaxSize: 10,

		//保留的旧的最大个数.此时我们输出了10MB的内容.
		//但是只有5个切割文件+1个主文件.其余5个都删掉了.按照切割出来的日期.早期的会优先进行删除
		//如果旧日志一直没有删除(没有满5个).但是已经过去30天了.这时候会自动删除
		MaxBackups: 5,

		//保留旧文件的最大天数
		MaxAge: 30,

		//是否压缩旧文件
		Compress: false,
	}
}
