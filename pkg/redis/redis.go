package redis

import (
	"context"
	"fmt"
	"time"
	"webssh-go/config"
	"webssh-go/pkg/logger"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func Init() {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Conf.Redis.Addr,
		Password: config.Conf.Redis.Password,
		DB:       config.Conf.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	RedisClient = client

	// 检查连接是否正常
	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.Info(fmt.Sprintf("redis连接失败：%v", client))
		return
	}
	logger.Info(fmt.Sprintf("redis连接成功信息：%v", client))
}
