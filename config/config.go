package config

import (
	"fmt"
	"github.com/spf13/viper"
	"webssh-go/pkg/logger"
)

type Config struct {
	Server struct {
		Ip   string `json:"ip" comment:"服务地址"`
		Port int    `json:"port" comment:"服务端口"`
	}
	Redis struct {
		Addr     string `json:"add" comment:"redis主机地址"`
		Password string `json:"password" commnet:"redis密码"`
		DB       int    `json:"db" comment:"redis数据库编号"`
	}
	ElasticSearch struct {
		Url      string `json:"url" comment:"es地址"`
		Username string `json:"username" comment:"es用户名"`
		Password string `json:"password" comment:"es密码"`
	}
	Audit struct {
		LoginAuditIndex  string `json:"login_audit" comment:"登录审计-es索引"`
		RecordAuditIndex string `json:"record_audit" comment:"操作审计-es索引"`
	}
	As3 struct {
		EndPoint        string `json:"endpoint" comment:"地址"`
		AccessKeyID     string `json:"accessKeyID" comment:"密钥ID"`
		SecretAccessKey string `json:"secretAccessKey" comment:"密钥KEY"`
		UseSSL          bool   `json:"useSSL" comment:"是否使用SSL"`
		Bucket          string `json:"bucket" comment:"桶名字"`
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
