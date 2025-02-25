package main

import (
	"context"
	"github.com/pshvedko/nocopy/internal/log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/pshvedko/nocopy/service"
)

func main() {
	var baseFlag string
	var fileFlag string
	var pipeFlag string
	var levelFlag slog.Level

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s := service.Chain{}
	c := &cobra.Command{
		Use:  "s3chain",
		Long: "S3 (not yet) compatible copy less proxy service",
		PersistentPreRun: func(*cobra.Command, []string) {
			slog.SetDefault(log.NewLogger(os.Stdout, levelFlag))
		},
		PreRun: func(*cobra.Command, []string) {
			context.AfterFunc(ctx, s.Stop)
		},
		RunE: func(*cobra.Command, []string) error {
			return s.Run(ctx, baseFlag, fileFlag, pipeFlag)
		},
	}

	c.Flags().VarP(log.NewLogLevel(&levelFlag, slog.LevelInfo), "level", "l", "log level")
	c.Flags().StringVar(&baseFlag, "base", "postgres://postgres:postgres@postgres:5432/nocopy", "data base")
	c.Flags().StringVar(&fileFlag, "file", "minio://admin:admin123@minio:9000/nocopy", "file storage")
	c.Flags().StringVar(&pipeFlag, "pipe", "nats://nats", "message broker")

	err := c.Execute()
	if err != nil {
		os.Exit(1)
	}
}
