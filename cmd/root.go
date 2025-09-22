package cmd

import (
	"github.com/carousell/ct-go/pkg/logger/log"
	"github.com/nguyentranbao-ct/chat-bot/internal/app"
	"github.com/nguyentranbao-ct/chat-bot/internal/kafka"
	"github.com/nguyentranbao-ct/chat-bot/internal/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "ct-communication-notification-worker",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		app.Invoke(
			server.StartServer,
			kafka.StartConsumeMessages,
		).Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
