/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"webssh-go/config"
	"webssh-go/pkg/es"
	"webssh-go/pkg/logger"
	"webssh-go/pkg/redis"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "webssh-go",
	Short: "go版本的webssh",
	Long:  "go版本的webssh",
	Run:   func(cmd *cobra.Command, args []string) {},
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
}
