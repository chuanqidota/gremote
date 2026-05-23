package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"gwebssh/config"
	"gwebssh/pkg/logger"

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

// Set 设置指定键
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

// Get 获取值 value 是传入的值
func Get(key string, value any) error {
	result, err := RedisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(result, value)
}

// DeleteKey 删除指定的键
func DeleteKey(key string) {
	isConnectedKey := key+"_connected"
	RedisClient.Del(context.Background(), key)
	RedisClient.Del(context.Background(),isConnectedKey)
}

// Exist 判断key存不存在
func Exist(key string) bool {
	exists, err := RedisClient.Exists(context.Background(), key).Result()
	if err != nil {
		logger.Error(fmt.Sprintf("获取redis中的key错误-%s", err.Error()))
		return false
	}
	if exists == 0 {
		return false
	} else {
		return true
	}
}

// IsConnected 判断有没有连接过
func IsConnected(key string) bool {
	isConnectedKey := key + "_connected"
	ok, err := RedisClient.SetNX(context.Background(), isConnectedKey, true, 24*60*60*time.Second).Result()
	if err != nil {
		logger.Error(fmt.Sprintf("SetNX失败-%s", err.Error()))
		return true
	}
	return !ok
}
