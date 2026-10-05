package redisUtils

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================================
// 延迟队列（DelayQueue）—— 纯 Redis 实现，通用的延迟消息组件
//
// 适用场景：下单超时关单、延迟退款、定时提醒等"延迟一段时间再投递消费"的消息。
// 相比"DB 轮询 delay_until + outbox 投递"方案，本组件把延迟控制完全放到 Redis
// （ZSet score = 到期时间戳，member = 消息 JSON），无 DB 轮询压力；生产者业务
// 事务外直接入队，消费者循环扫描到期成员再投递到目标 Stream，Stream 侧继续走
// 既有 Redis Stream 消费组（XREADGROUP，多实例天然安全）。
//
// 结构：
//   - 主 ZSet（key = "<prefix>:zset"）：member = 消息 JSON，score = 到期纳秒时间戳
//   - 投递占位 ZSet（key = "<prefix>:done"）：member = 消息 id（NX 占位防重），
//     score = 投递时刻；每轮扫描按保留期裁剪（doneGuardRetention），防止只增不减
//
// 多实例安全：
//   - 到期扫描：ZRANGEBYSCORE 0..now 取到期成员，串行处理。
//   - 防重复投递：先 ZADD <done> NX member=消息id，命中已存在 → 跳过；
//     XADD 成功才从主 ZSet 移除；失败回滚占位下一轮重试。
//   - 注意：本组件自身是"单循环跑到期扫描"，需要多实例并发消费时，由调用方
//     用 redsync 锁做选主（每实例持锁后 Run），或直接每实例各跑一个（占位机制
//     保证同一消息只会被投递一次，多实例只是各自扫各自的，无害）。
//
// 依赖 go-redis v9（与项目现有 redisUtils 一致）。
// ============================================================

// doneGuardRetention done 占位的保留期：占位只需挡住"并发扫描/多实例同一批成员重复投递"
// 这个窗口（一轮扫描，秒级）；投递成功后留着无意义。保留 1 小时远大于任何扫描周期，
// 兼顾"实例 A 占位后 XADD 前卡死 1 小时内，实例 B 重扫不重复投递"的极端场景。
const doneGuardRetention = time.Hour

// DelayQueueKey 返回延迟队列主 ZSet 的 key（前缀派生）
func DelayQueueKey(prefix string) string {
	return prefix + ":zset"
}

// DelayQueueDoneKey 返回投递占位 ZSet 的 key（前缀派生）
func DelayQueueDoneKey(prefix string) string {
	return prefix + ":done"
}

// DelayMessage 延迟队列消息体（member JSON 的约定结构）
//
// 业务自定义字段可 append 到 Payload 里，基础字段必须保留：
//   - Id：消息唯一 id（生产者生成：雪花 id/uuid），双用法：done 占位 + XADD 后的幂等凭据；
//   - Topic：到期后 XADD 到的目标 Stream 名（如 "shop:order:delay"）；
//   - Payload：业务消息体（原样透传，Stream 条目 body=Payload 字符串，与 Publisher 一致）。
type DelayMessage struct {
	Id         string `json:"id"`
	Topic      string `json:"topic"`
	Payload    string `json:"payload"`
	DelayUntil int64  `json:"delayUntil"` // 到期时间（Unix 纳秒 = score）
	CreateAt   int64  `json:"createAt"`   // 入队时间（Unix 纳秒）
}

// DelayQueue 延迟队列处理器（消费者侧：到期扫描 + 投递到 Stream）
type DelayQueue struct {
	rdb      *redis.Client
	prefix   string
	interval time.Duration
	logger   *slog.Logger
}

// NewDelayQueue 创建延迟队列处理器
//
// prefix：队列前缀（如 "shop:order:delay:q"）；生产/消费两侧必须一致。
// interval：到期扫描间隔（<=0 时默认 1s）。
// logger：nil 时使用 slog.Default()。
func NewDelayQueue(rdb *redis.Client, prefix string, interval time.Duration, logger *slog.Logger) *DelayQueue {
	if interval <= 0 {
		interval = time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DelayQueue{
		rdb:      rdb,
		prefix:   prefix,
		interval: interval,
		logger:   logger,
	}
}

// Run 阻塞循环：按 interval 扫描到期消息并投递。ctx 取消时优雅退出。
//
// 单实例/锁选主场景直接各起一个即可（多实例靠 done 占位防重）。
func (q *DelayQueue) Run(ctx context.Context) {
	ticker := time.NewTicker(q.interval)
	defer ticker.Stop()
	q.logger.Info("delayQueue started", "prefix", q.prefix, "interval", q.interval.String())
	for {
		select {
		case <-ctx.Done():
			q.logger.Info("delayQueue stopped", "prefix", q.prefix)
			return
		case <-ticker.C:
			q.dispatchDue(ctx)
		}
	}
}

// dispatchDue 扫描一次到期消息并批量投递（内部串行，panic 不允许带出）
func (q *DelayQueue) dispatchDue(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			q.logger.Error("delayQueue dispatch panic", "panic", r)
		}
	}()
	now := time.Now().UnixNano()
	// done 占位裁剪：清掉保留期外的旧占位（score=投递时刻）。
	// 不裁剪的话每个投递过的消息 id 永久留一条，只增不减；裁剪只影响 1 小时前的
	// 占位，不影响任何在途消息的防重窗口。
	if err := q.rdb.ZRemRangeByScore(ctx, q.doneKey(), "0",
		fmt.Sprintf("%d", time.Now().Add(-doneGuardRetention).UnixNano())).Err(); err != nil {
		q.logger.Warn("delayQueue done-trim failed", "err", err)
	}
	members, err := q.rdb.ZRangeByScore(ctx, q.zsetKey(), &redis.ZRangeBy{
		Min:    "0",
		Max:    fmt.Sprintf("%d", now),
		Offset: 0,
		Count:  100, // 单轮最多 100 条，防一次性刷太久
	}).Result()
	if err != nil && err != redis.Nil {
		q.logger.Error("delayQueue zrange failed", "err", err)
		return
	}
	// 扫描周期日志：每轮 debug（消息量大时 info 会刷屏；delivered 才打 info）
	q.logger.Debug("delayQueue dispatch scan", "due", now, "count", len(members))
	for _, raw := range members {
		q.dispatchOne(ctx, raw)
	}
}

// dispatchOne 投递单条到期消息：占位（防重）→ 解析 → XADD → 移除主 ZSet
func (q *DelayQueue) dispatchOne(ctx context.Context, raw string) {
	var msg DelayMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		q.logger.Error("delayQueue unmarshal failed, drop", "raw", raw, "err", err)
		q.rdb.ZRem(ctx, q.zsetKey(), raw) // 畸形消息直接清掉，避免死循环
		return
	}
	// 1. 占位：done ZSet 加消息 id（NX）。已存在 -> 已投递过（多实例/重扫），跳过
	added, err := q.rdb.ZAddNX(ctx, q.doneKey(), redis.Z{Score: float64(time.Now().UnixNano()), Member: msg.Id}).Result()
	if err != nil {
		q.logger.Error("delayQueue occupy failed", "id", msg.Id, "err", err)
		return
	}
	if added == 0 {
		return
	}
	if msg.Topic == "" {
		q.logger.Error("delayQueue topic empty, drop", "id", msg.Id)
		q.rdb.ZRem(ctx, q.zsetKey(), raw)
		q.rdb.ZRem(ctx, q.doneKey(), msg.Id)
		return
	}
	// 2. XADD 到目标 Stream（body 单键，与 streamUtils.Publisher 一致；
	//    近似 MAXLEN 与 Publisher 同口径，防止只被本组件写入的 Stream 无界增长）
	streamID, err := q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: msg.Topic,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{"body": msg.Payload},
	}).Result()
	if err != nil {
		// XADD 失败回滚占位，下一轮重试
		q.logger.Error("delayQueue xadd failed", "id", msg.Id, "topic", msg.Topic, "err", err)
		q.rdb.ZRem(ctx, q.doneKey(), msg.Id)
		return
	}
	// 3. 成功：主 ZSet 移除 + 日志
	removed := q.rdb.ZRem(ctx, q.zsetKey(), raw).Val()
	if removed == 0 {
		q.logger.Warn("delayQueue zrem failed", "id", msg.Id, "raw", raw)
	}
	q.logger.Info("delayQueue delivered", "id", msg.Id, "topic", msg.Topic, "streamId", streamID, "removed", removed)
}

func (q *DelayQueue) zsetKey() string { return DelayQueueKey(q.prefix) }
func (q *DelayQueue) doneKey() string { return DelayQueueDoneKey(q.prefix) }

// EnqueueDelay 生产者：入队一条延迟消息（延迟 delay 后由队列投递到 msg.Topic）
//
// prefix 与 NewDelayQueue 一致；msg.Id 必填，同 id 重复入队会重置到期时间
// （业务上应每次生成新 id，例外：想"续期"时复用 id 即可）。
func EnqueueDelay(ctx context.Context, rdb *redis.Client, prefix string, msg DelayMessage, delay time.Duration) error {
	if msg.Id == "" {
		return fmt.Errorf("delayQueue: msg.Id required")
	}
	if msg.Topic == "" {
		return fmt.Errorf("delayQueue: msg.Topic required")
	}
	msg.DelayUntil = time.Now().Add(delay).UnixNano()
	msg.CreateAt = time.Now().UnixNano()
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("delayQueue: marshal: %w", err)
	}
	// 以 JSON 原文作为 member：同 payload 同 id 时 Score 会被覆盖（延期语义）
	return rdb.ZAdd(ctx, DelayQueueKey(prefix), redis.Z{Score: float64(msg.DelayUntil), Member: string(b)}).Err()
}

// DelayQueueLen 查询队列中未投递条数（含未到期），便于监控/测试断言
func DelayQueueLen(ctx context.Context, rdb *redis.Client, prefix string) (int64, error) {
	n, err := rdb.ZCard(ctx, DelayQueueKey(prefix)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	return n, err
}

// EnqueueXAddNow 立即向 Stream 投递（无延迟），供"到期即发"之外的直发场景复用；
// 与 Publisher.XAdd 同构（body 单键）。延迟场景请用 EnqueueDelay。
func EnqueueXAddNow(ctx context.Context, rdb *redis.Client, topic, body string) (string, error) {
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{"body": body},
	}).Result()
}
