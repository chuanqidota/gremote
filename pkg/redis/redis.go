package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"webssh-go/config"
	"webssh-go/pkg/logger"

	"github.com/redis/go-redis/v9"
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
		logger.Info(fmt.Sprintf("redis连接失败：%s", err.Error()))
		return
	}
	logger.Info(fmt.Sprintf("redis连接成功信息：%v", client))
}

func Set(key string, value any, expiration time.Duration) error {
	value_, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := RedisClient.Set(context.Background(), key, value_, expiration).Err(); err != nil {
		return err
	} else {
		return nil
	}
}

func Get(key string, value any) error {
	result, err := RedisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(result, value)
}

func DeleteKey(key string) error {
	// 删除指定的键
	err := RedisClient.Del(context.Background(), key).Err()
	if err != nil {
		logger.Error(fmt.Sprintf("删除键：%s", err.Error()))
		return err
	}
	return nil
}
