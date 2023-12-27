package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/pshvedko/nocopy/service"
)

func main() {
	var pipeFlag string

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s := service.Proxy{}
	c := &cobra.Command{
		Use:  "s3proxy",
		Long: "S3 (not yet) compatible copy less proxy service",
		PreRun: func(*cobra.Command, []string) {
			context.AfterFunc(ctx, s.Stop)
		},
		RunE: func(*cobra.Command, []string) error {
			return s.Run(ctx, pipeFlag)
		},
	}

	c.Flags().StringVar(&pipeFlag, "pipe", "nats://nats", "message broker")

	err := c.Execute()
	if err != nil {
		os.Exit(1)
	}
}
