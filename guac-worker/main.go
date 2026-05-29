package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

// Config guac-worker 配置，仅需要 S3 连接信息
type Config struct {
	S3 struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
		Bucket    string `mapstructure:"bucket"`
		UseSSL    bool   `mapstructure:"use_ssl"`
	} `mapstructure:"s3"`
	ConvertTimeout int `mapstructure:"convert_timeout"` // 每步转换超时（秒），默认600
}

var cfg Config
var s3Client *minio.Client

func getConvertTimeout() time.Duration {
	if cfg.ConvertTimeout > 0 {
		return time.Duration(cfg.ConvertTimeout) * time.Second
	}
	return 600 * time.Second // default 10 minutes
}

// InitConfig 从 config.yaml 加载配置，支持环境变量覆盖
func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: config file not found, using defaults: %v", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	log.Printf("Config loaded: S3.Endpoint=%s, S3.Bucket=%s", cfg.S3.Endpoint, cfg.S3.Bucket)

	// 初始化 S3 客户端单例
	client, err := minio.New(cfg.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3.AccessKey, cfg.S3.SecretKey, ""),
		Secure: cfg.S3.UseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to init S3 client: %v", err)
	}
	s3Client = client
}

func main() {
	InitConfig()

	r := gin.Default()

	// /convert 接收 .guac 转 MP4 的转换请求，/health 用于健康检查
	r.POST("/convert", handleConvert)
	r.GET("/progress", handleProgress)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("guac-worker starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
