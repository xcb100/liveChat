package tools

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
	Ctx         = context.Background()
)

// RedisConfig 定义Redis配置结构，避免直接依赖common包
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

// InitRedis 初始化Redis客户端
// 修改为接收配置参数，而不是直接依赖common包
func InitRedis(conf RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conf.Host, conf.Port),
		Password: conf.Password,
		DB:       conf.DB,
		PoolSize: conf.PoolSize,
	})

	// 测试连接
	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		return err
	}

	return nil
}

// GetRedis 获取Redis客户端
func GetRedis() *redis.Client {
	return RedisClient
}
