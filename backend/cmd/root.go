package cmd

import (
	"os"
	"os/signal"
	"gwebssh/pkg/s3"

	"context"
	"gwebssh/config"
	"gwebssh/pkg/es"
	"gwebssh/pkg/logger"
	"gwebssh/pkg/redis"
	"gwebssh/router"

	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gwebssh",
	Short: "go版本的webssh",
	Long:  "go版本的webssh",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	config.Init()
	if err := config.Validate(); err != nil {
		logger.Error(fmt.Sprintf("配置校验失败: %s", err.Error()))
		os.Exit(1)
	}
	logger.Init()
	redis.Init()
	es.Init()
	s3.Init()
}

func Run() {
	addr := fmt.Sprintf("%s:%d", config.Conf.Server.Host, config.Conf.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        router.Engine(),
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		// 创建一个5秒的上下文，以便优雅关闭
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("服务关闭失败: %s", err.Error()))
		}
	}()
	if err := server.ListenAndServe(); err != nil {
		logger.Error(fmt.Sprintf("服务启动失败: %s", err.Error()))
	}
}
