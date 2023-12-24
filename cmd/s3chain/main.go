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
	var baseFlag string
	var fileFlag string
	var pipeFlag string

	var s service.Chain
	defer s.Wait()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c := &cobra.Command{
		Use:  "s3chain",
		Long: "S3 (not yet) compatible copy less proxy service",
		RunE: func(*cobra.Command, []string) error {
			return s.Run(ctx, baseFlag, fileFlag, pipeFlag)
		},
	}

	c.Flags().StringVar(&baseFlag, "base", "postgres://postgres:postgres@postgres:5432/nocopy", "data base")
	c.Flags().StringVar(&fileFlag, "file", "minio://admin:admin123@minio:9000/nocopy", "file storage")
	c.Flags().StringVar(&pipeFlag, "pipe", "nats://nats", "message broker")

	err := c.Execute()
	if err != nil {
		os.Exit(1)
	}
}
