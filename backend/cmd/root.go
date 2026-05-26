package cmd

import (
	"gremote/pkg/s3"
	"os"
	"os/signal"

	"context"
	"gremote/config"
	"gremote/pkg/es"
	"gremote/pkg/logger"
	"gremote/pkg/redis"
	"gremote/router"

	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gremote",
	Short: "Go Remote Terminal",
	Long:  "Go Remote Terminal",
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
	logger.Init(logger.LogConfig{
		Filename:   config.Conf.Logger.Filename,
		MaxSize:    config.Conf.Logger.MaxSize,
		MaxBackups: config.Conf.Logger.MaxBackups,
		MaxAge:     config.Conf.Logger.MaxAge,
	})
	redis.Init()
	es.Init()
	s3.Init()
}

func Run() {
	addr := fmt.Sprintf("%s:%d", config.Conf.Server.Host, config.Conf.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        router.Engine(),
		ReadTimeout:    time.Duration(config.Conf.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.Conf.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
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
