package cmd

import (
	"gremote/pkg/minio"
	"os"
	"os/signal"

	"context"
	"gremote/config"
	"gremote/pkg/elasticsearch"
	"gremote/pkg/logger"
	"gremote/pkg/redis"
	"gremote/router"

	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// rootCmd Cobra 根命令，启动后调用 Run() 启动 HTTP 服务
var rootCmd = &cobra.Command{
	Use:   "gremote",
	Short: "Go Remote Terminal",
	Long:  "Go Remote Terminal",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

// Execute 执行 Cobra 根命令，程序入口
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// init 按顺序初始化各组件：配置 → 日志 → Redis → Elasticsearch → S3
func init() {
	config.Init()
	logger.Init(logger.LogConfig{
		Filename:   config.Conf.Logger.Filename,
		MaxSize:    config.Conf.Logger.MaxSize,
		MaxBackups: config.Conf.Logger.MaxBackups,
		MaxAge:     config.Conf.Logger.MaxAge,
	})
	redis.Init()
	elasticsearch.Init()
	minio.Init()
}

// Run 启动 HTTP 服务并监听系统中断信号实现优雅关闭
func Run() {
	addr := fmt.Sprintf("%s:%d", config.Conf.Server.Host, config.Conf.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        router.Engine(),
		ReadTimeout:    time.Duration(config.Conf.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.Conf.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// 优雅关闭：监听中断信号，收到后等待进行中的请求完成
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Conf.Server.ShutdownTimeout)*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			fmt.Println(err.Error())
		}
	}()
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err.Error())
	}
}
