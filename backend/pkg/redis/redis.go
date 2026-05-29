package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"gremote/config"
	"gremote/pkg/logger"

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

// Set 设置指定键，值会被 JSON 序列化后存储
func Set(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return RedisClient.Set(context.Background(), key, data, expiration).Err()
}

// Get 获取值 value 是传入的值
func Get(key string, value any) error {
	result, err := RedisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(result, value)
}

// DeleteKey 删除会话键及其关联的连接标记键
func DeleteKey(key string) {
	isConnectedKey := key + "_connected"
	RedisClient.Del(context.Background(), key)
	RedisClient.Del(context.Background(), isConnectedKey)
}

// IsConnected 判断有没有连接过
func IsConnected(key string) bool {
	isConnectedKey := key + "_connected"
	ok, err := RedisClient.SetNX(context.Background(), isConnectedKey, true, time.Duration(config.Conf.Server.SessionTTL)*time.Second).Result()
	if err != nil {
		logger.Error(fmt.Sprintf("SetNX失败-%s", err.Error()))
		return true
	}
	return !ok
}
