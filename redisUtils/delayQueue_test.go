package redisUtils

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 本测试连接本机 dev redis（localhost:30018，无密码）验证延迟队列真实行为。
// 依赖外部 Redis——CI/无 Redis 环境会自动 SKIP（ping 失败即跳过），不阻塞构建。
func testRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:30018"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Logf("redis not reachable, skip: %v", err)
		return nil
	}
	return rdb
}

func TestDelayQueue_Delivery(t *testing.T) {
	rdb := testRedisClient(t)
	if rdb == nil {
		t.Skip("redis not reachable")
	}
	ctx := context.Background()
	prefix := "test:delay:q"
	topic := "test:delay:stream"

	// 清理历史（幂等）
	rdb.Del(ctx, DelayQueueKey(prefix), DelayQueueDoneKey(prefix))
	rdb.Del(ctx, topic)

	// 1. 生产者：入队一条 1s 延迟消息
	msg := DelayMessage{
		Id:      "msg-001",
		Topic:   topic,
		Payload: `{"orderId":"123","stateVersion":1}`,
	}
	if err := EnqueueDelay(ctx, rdb, prefix, msg, 1*time.Second); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	n, err := DelayQueueLen(ctx, rdb, prefix)
	if err != nil || n != 1 {
		t.Fatalf("queue len = %d, err=%v, want 1", n, err)
	}

	// 2. 消费者：起 Run，等待投递（预计 1~2s）
	q := NewDelayQueue(rdb, prefix, 300*time.Millisecond, nil)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { q.Run(runCtx); close(done) }()

	// 轮询等待 Stream 出现消息 + 主 ZSet 清空（两条件都满足才算投递完成，
	// 避免"XADD 已可见但 ZRem 尚未执行"的瞬时窗口误判——dispatchOne 里
	// XAdd 与 ZRem 是两条独立 Redis 命令，非原子）
	deadline := time.Now().Add(5 * time.Second)
	var entries []redis.XMessage
	var left int64
	for time.Now().Before(deadline) {
		entries, _ = rdb.XRange(ctx, topic, "-", "+").Result()
		left, _ = DelayQueueLen(ctx, rdb, prefix)
		if len(entries) > 0 && left == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(entries) == 0 {
		t.Fatal("stream got no message within 5s")
	}
	if body := entries[0].Values["body"]; body != msg.Payload {
		t.Fatalf("body=%v, want %v", body, msg.Payload)
	}
	if left != 0 {
		t.Fatalf("queue len after delivery = %d, want 0", left)
	}
	t.Logf("delivered OK: streamId=%s", entries[0].ID)

	// 清理
	cancel()
	<-done
	rdb.Del(ctx, DelayQueueKey(prefix), DelayQueueDoneKey(prefix), topic)
}

func TestDelayQueue_NotDeliveredBeforeDue(t *testing.T) {
	rdb := testRedisClient(t)
	if rdb == nil {
		t.Skip("redis not reachable")
	}
	ctx := context.Background()
	prefix := "test:delay:q2"
	topic := "test:delay:stream2"
	rdb.Del(ctx, DelayQueueKey(prefix), DelayQueueDoneKey(prefix))
	rdb.Del(ctx, topic)

	// 入队一条 10 秒延迟
	_ = EnqueueDelay(ctx, rdb, prefix, DelayMessage{Id: "m2", Topic: topic, Payload: `{"a":1}`}, 10*time.Second)
	// 起消费者，短跑 1.5s：不应投递
	q := NewDelayQueue(rdb, prefix, 300*time.Millisecond, nil)
	runCtx, cancel := context.WithCancel(ctx)
	go q.Run(runCtx)
	time.Sleep(1500 * time.Millisecond)
	cancel()
	entries, _ := rdb.XRange(ctx, topic, "-", "+").Result()
	if len(entries) != 0 {
		t.Fatal("should not deliver before due time")
	}
	t.Log("OK: not delivered before due")
	rdb.Del(ctx, DelayQueueKey(prefix), DelayQueueDoneKey(prefix), topic)
}
