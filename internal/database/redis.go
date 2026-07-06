package database

import (
	"context"
	"log/slog"

	"github.com/go-redis/redis/v8"
)

// NewRedisClient 負責產出 Redis 客戶端
func NewRedisClient(addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// 🛡️ 及早失敗 (Fail-Fast)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Redis 連線失敗", "component", "database", "error", err.Error())
		return nil, err
	}

	slog.Info("Redis 連線成功", "component", "database")
	return rdb, nil
}
