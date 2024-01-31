package config

import (
	"github.com/spf13/viper"
	"webssh-go/pkg/logger"
	"fmt"
)
 
type Config struct {
	Redis struct {
		Addr string `json:"add" comment:"redis主机地址"`
		Password string `json:"password" commnet:"redis密码"`
		DB int `json:"db" comment:"redis数据库编号"`
	}
	ElasticSearch struct {
		Url string `json:"url" comment:"es地址"`
		Username string `json:"username" comment:"es用户名"`
		Password string `json:"password" comment:"es密码"`
	}
}

var Conf = new(Config)

func Init() {
	viper.SetConfigFile("./config/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		logger.Error(fmt.Sprintf("读取配置文件失败:%s", err.Error()))
	}
	// 解析配置文件
	if err := viper.Unmarshal(&Conf); err != nil {
		logger.Error(fmt.Sprintf("解析配置文件失败:%s", err.Error()))
	}
	logger.Info(fmt.Sprintf("解析配置文件：%v", *Conf))
}