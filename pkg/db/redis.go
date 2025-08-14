package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jacl-coder/PixelStorm-Server/config"
)

var (
	// RedisClient 全局Redis客户端实例
	RedisClient *redis.Client
	// Ctx 全局上下文
	Ctx = context.Background()
)

// InitRedis 初始化Redis连接
func InitRedis() error {
	redisConfig := config.GlobalConfig.Redis

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisConfig.GetRedisAddr(),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(Ctx, 5*time.Second)
	defer cancel()

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis连接失败: %w", err)
	}

	log.Println("成功连接到Redis服务器")
	return nil
}

// CloseRedis 关闭Redis连接
func CloseRedis() {
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			log.Printf("关闭Redis连接时发生错误: %v", err)
			return
		}
		log.Println("Redis连接已关闭")
	}
}
