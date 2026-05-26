package config

import (
	"fmt"
	"github.com/spf13/viper"
	"gremote/pkg/logger"
)

type ServerConfig struct {
	Host               string `yaml:"Host" comment:"服务监听地址"`
	Port               int    `yaml:"Port" comment:"服务端口"`
	SessionTTL         int    `yaml:"SessionTTL" comment:"会话密钥过期时间（秒）"`
	ReadTimeout        int    `yaml:"ReadTimeout" comment:"HTTP读取超时（秒）"`
	WriteTimeout       int    `yaml:"WriteTimeout" comment:"HTTP写入超时（秒）"`
	ShutdownTimeout    int    `yaml:"ShutdownTimeout" comment:"优雅关闭超时（秒）"`
	InsecureSkipVerify bool   `yaml:"InsecureSkipVerify" comment:"跳过SSH主机密钥验证"`
}

type RedisConfig struct {
	Addr     string `yaml:"Addr" comment:"Redis地址"`
	Password string `yaml:"Password" comment:"Redis密码"`
	DB       int    `yaml:"DB" comment:"Redis数据库编号"`
}

type ESConfig struct {
	Url      string `yaml:"Url" comment:"ES地址"`
	Username string `yaml:"Username" comment:"ES用户名"`
	Password string `yaml:"Password" comment:"ES密码"`
}

type AuditConfig struct {
	LoginAuditIndex  string `yaml:"LoginAuditIndex" comment:"登录审计索引前缀"`
	RecordAuditIndex string `yaml:"RecordAuditIndex" comment:"操作审计索引前缀"`
}

type S3Config struct {
	Endpoint        string `yaml:"Endpoint" comment:"MinIO地址"`
	AccessKeyID     string `yaml:"AccessKeyID" comment:"AccessKey"`
	SecretAccessKey string `yaml:"SecretAccessKey" comment:"SecretKey"`
	UseSSL          bool   `yaml:"UseSSL" comment:"是否使用HTTPS"`
	Bucket          string `yaml:"Bucket" comment:"桶名"`
}

type GuacdConfig struct {
	Host           string `yaml:"Host" comment:"guacd主机地址"`
	Port           int    `yaml:"Port" comment:"guacd端口"`
	RecordingPath  string `yaml:"RecordingPath" comment:"后端读取录制文件路径"`
	GuacdPath      string `yaml:"GuacdPath" comment:"guacd容器内录制文件路径"`
	DefaultWidth   int    `yaml:"DefaultWidth" comment:"RDP默认宽度"`
	DefaultHeight  int    `yaml:"DefaultHeight" comment:"RDP默认高度"`
	DefaultDPI     int    `yaml:"DefaultDPI" comment:"RDP默认DPI"`
	SessionTimeout int    `yaml:"SessionTimeout" comment:"RDP会话超时（秒）"`
}

type GuacWorkerConfig struct {
	URL     string `yaml:"URL" comment:"Worker服务地址"`
	Timeout int    `yaml:"Timeout" comment:"转换超时（秒）"`
}

type LoggerConfig struct {
	Filename   string `yaml:"Filename" comment:"日志文件路径"`
	MaxSize    int    `yaml:"MaxSize" comment:"日志文件最大大小（MB）"`
	MaxBackups int    `yaml:"MaxBackups" comment:"最大保留旧日志数量"`
	MaxAge     int    `yaml:"MaxAge" comment:"旧日志最长保留天数"`
}

type Config struct {
	Server        ServerConfig     `yaml:"Server"`
	Redis         RedisConfig      `yaml:"Redis"`
	ElasticSearch ESConfig         `yaml:"ElasticSearch"`
	Audit         AuditConfig      `yaml:"Audit"`
	S3            S3Config         `yaml:"S3"`
	Guacd         GuacdConfig      `yaml:"Guacd"`
	GuacWorker    GuacWorkerConfig `yaml:"GuacWorker"`
	Logger        LoggerConfig     `yaml:"Logger"`
}

var Conf = new(Config)

func Init() {
	viper.SetConfigFile("./config/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		logger.Error(fmt.Sprintf("读取配置文件失败:%s", err.Error()))
	}
	viper.SetEnvPrefix("GREMOTE")
	viper.AutomaticEnv()
	if err := viper.Unmarshal(&Conf); err != nil {
		logger.Error(fmt.Sprintf("解析配置文件失败:%s", err.Error()))
	}
	logger.Info("配置文件加载完成")
}

func Validate() error {
	if Conf.Server.Port <= 0 || Conf.Server.Port > 65535 {
		return fmt.Errorf("Server.Port 必须在 1-65535 之间")
	}
	if Conf.Redis.Addr == "" {
		return fmt.Errorf("Redis.Addr 不能为空")
	}
	return nil
}
