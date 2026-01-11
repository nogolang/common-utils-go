package watermillUtils

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/components/requeuer"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type RequeuePoisonUtils struct {
	WaterLog watermill.LoggerAdapter
	AmqpUrl  string
	Logger   *zap.Logger
}

func NewRequeuePoisonUtils(WaterLog watermill.LoggerAdapter, Logger *zap.Logger, rabbitMQUrl string) *RequeuePoisonUtils {
	return &RequeuePoisonUtils{
		WaterLog: WaterLog,
		AmqpUrl:  rabbitMQUrl,
		Logger:   Logger,
	}
}

// 创建一个死信队列的中间件，当消费者发生错误，消息会重新发送到死信队列
func (receiver *RequeuePoisonUtils) CreatePositionMiddle(topic string) (message.HandlerMiddleware, error) {
	//创建死信队列
	poisonConfig := amqp.NewDurableQueueConfig(receiver.AmqpUrl)

	//创建一个发送者
	poisonPublisher, err := amqp.NewPublisher(poisonConfig, receiver.WaterLog)
	if err != nil {
		return nil, errors.Wrap(err, "初始化BackMoneyPosition失败")
	}
	//把死信发送者封装为一个中间件，返回回去，这样我们的正常消费者就可以使用这个中间件了
	//到时候会往这个topic里发送消息
	poisonQueueMiddleware, err := middleware.PoisonQueue(poisonPublisher, topic)
	if err != nil {
		return nil, errors.Wrap(err, "初始化PoisonQueue middleware失败")
	}

	return poisonQueueMiddleware, nil
}

// 然后利用requeue重新把死信队列里的消息发送回当前队列
func (receiver *RequeuePoisonUtils) CreateRequeue(poisonSubscriber *amqp.Subscriber, publisher *amqp.Publisher, poisonTopic string, businessTopic string) error {
	newRequeue, err := requeuer.NewRequeuer(requeuer.Config{
		//订阅者，传递死信订阅者即可
		//  然后从死信topic里订阅
		//  只要死信队列里有消息，那么就requeue就可以获取到
		Subscriber:     poisonSubscriber,
		SubscribeTopic: poisonTopic,

		//传递一个发送者，到时候由这个发送者去发送，主要是用到发送者里面的配置
		//  这个发送者的配置，必须和业务发送者是一样的
		//  因为这里是把消息重新发送到 业务队列中，所以你这个发送者肯定也是业务publish
		Publisher: publisher,

		//发送到哪里，肯定是发送到我们的业务队列
		GeneratePublishTopic: func(params requeuer.GeneratePublishTopicParams) (string, error) {
			return businessTopic, nil
		},

		//延迟发送
		Delay: time.Millisecond * 3000,
	}, receiver.WaterLog)
	if err != nil {
		return errors.Wrap(err, "创建water requeue失败")
	}

	//启动newRequeue
	go func() {
		err := newRequeue.Run(context.Background())
		if err != nil {
			receiver.Logger.Fatal("启动water requeue失败", zap.Error(err))
			return
		}
	}()
	return nil
}
