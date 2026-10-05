// Package gormUtils —— 连接池全局排空（2026-09-15 zhparser 词典即时生效配套）
//
// 背景：zhparser 表级词典只对"新连接"生效。五合一进程内各模块（admin/api/business/task）
// 各有自己的 gorm 池，改完词典要全进程立刻生效时，需要把所有池的空闲连接换新。
// 之前用 pg_terminate_backend 硬杀服务端连接，经 frp 隧道会让客户端复用时吃到 RST
// （并发请求偶发 500/401）；改为客户端优雅排空：临时把空闲存活时间调到极小，
// database/sql 的清理协程会在 ~1s 内把空闲连接全部 Close（发 Terminate 干净退出），
// 之后恢复设置——正在执行的查询全程不受影响。
package gormUtils

import (
	"database/sql"
	"sync"
	"time"
)

var (
	pooledMu sync.Mutex
	pooledDB []*sql.DB
)

// registerPool 登记进程内每个 gorm 池的 *sql.DB（SetGormThread 时调用）
func registerPool(db *sql.DB) {
	pooledMu.Lock()
	defer pooledMu.Unlock()
	pooledDB = append(pooledDB, db)
}

// DrainAllIdlePools 优雅排空进程内所有池的空闲连接（等待清理协程完成关闭后恢复设置）。
// 用于词典编译后让全部模块的下一次请求都拿到加载了新词典的新连接。
// 清理协程巡检周期约 1s，这里等 1.5s 覆盖一个周期；期间活跃连接不受影响。
func DrainAllIdlePools() {
	pooledMu.Lock()
	dbs := make([]*sql.DB, len(pooledDB))
	copy(dbs, pooledDB)
	pooledMu.Unlock()

	for _, db := range dbs {
		db.SetConnMaxIdleTime(time.Nanosecond) // 空闲即关（由清理协程执行，非阻塞）
	}
	time.Sleep(1500 * time.Millisecond)
	for _, db := range dbs {
		db.SetConnMaxIdleTime(0) // 恢复不限（避免常态频繁重建；存活期仍由 ConnMaxLifetime 管）
	}
}
