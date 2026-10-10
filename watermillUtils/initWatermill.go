package watermillUtils

import (
	"context"
	"log"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
)

func NewWaterLogger(logger *slog.Logger) watermill.LoggerAdapter {
	return watermill.NewSlogLogger(logger)
}

// 这个router，是消费者需要使用的，生产者无需使用router
//
// 2026-09-26：原先签名带一个 *configUtils.CommonConfig，但函数体里从未用过它——
// 纯冗余参数，直接删掉。
func NewWaterRouter(waterLogger watermill.LoggerAdapter) *message.Router {
	router, err := message.NewRouter(message.RouterConfig{}, nil)
	if err != nil {
		log.Fatalf("创建water消费者路由失败: %v", err)
		return nil
	}

	//平滑重启插件
	router.AddPlugin(plugin.SignalsHandler)
	router.AddMiddleware(
		middleware.CorrelationID,

		//处理panic
		//它会将 panic 作为错误传递给 Retry 中间件
		middleware.Recoverer,
	)

	go func() {
		//因为这里一开始就启动了，后面我们添加消费者的时候，需要手动的使用一次RunHandlers
		err := router.Run(context.Background())
		if err != nil {
			log.Fatalf("启动water消费者路由失败: %v", err)
			return
		}
	}()
	return router
}
