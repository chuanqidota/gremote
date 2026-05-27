package logger

import (
	"io"
	"os"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

// LogConfig 日志配置，控制日志文件路径和轮转策略
type LogConfig struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

// Init 初始化日志系统，配置文件轮转和控制台双输出
func Init(cfg ...LogConfig) {
	var logConf LogConfig
	if len(cfg) > 0 {
		logConf = cfg[0]
	} else {
		logConf = LogConfig{
			Filename:   "./log/gremote.log",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     7,
		}
	}
	logFile := &lumberjack.Logger{
		Filename:   logConf.Filename,
		MaxSize:    logConf.MaxSize,
		MaxBackups: logConf.MaxBackups,
		MaxAge:     logConf.MaxAge,
		LocalTime:  true,
		Compress:   false,
	}

	// 创建一个 logrus.Logger 实例，用于日志记录和输出
	// 设置日志级别
	logger.SetLevel(logrus.InfoLevel)

	// 设置日志输出格式
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true, // 禁用颜色输出
		TimestampFormat: "2006-01-02 15:04:05",
	})
	// 将日志输出到文件和控制台
	logger.SetOutput(io.MultiWriter(logFile, os.Stdout))
}

// Info 记录 INFO 级别日志
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Error 记录 ERROR 级别日志
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Debug 记录 DEBUG 级别日志
func Debug(args ...interface{}) {
	logger.Debug(args...)
}
