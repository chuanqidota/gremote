/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"webssh-go/pkg/as3"

	"webssh-go/config"
	"webssh-go/pkg/es"
	"webssh-go/pkg/logger"
	"webssh-go/pkg/redis"
	"webssh-go/router"

	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "webssh-go",
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
	logger.Init()
	redis.Init()
	es.Init()
	as3.Init()
}

func Run() {
	addr := fmt.Sprintf("%s:%d", config.Conf.Server.Ip, config.Conf.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        router.Engine(),
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err.Error())
	}
}
