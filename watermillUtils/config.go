package watermillUtils

// RabbitMqConfig rabbitmq 连接配置
//
// ⚠️ 本项目（3-my-shop-single）的 MQ 已改为 Redis Stream + outbox，该配置段无人消费。
// 仅为 2-my-shop 等旧工程保留，勿在新代码里使用。
type RabbitMqConfig struct {
	Url string `json:"url"`
}
