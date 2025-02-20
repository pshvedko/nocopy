package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/pshvedko/nocopy/internal/log"
	"github.com/pshvedko/nocopy/service"
)

func main() {
	var addrFlag string
	var portFlag string
	var baseFlag string
	var fileFlag string
	var pipeFlag string
	var sizeFlag int64
	var levelFlag slog.Level

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s := service.Block{}
	c := &cobra.Command{
		Use:  "s3block",
		Long: "S3 (not yet) compatible copy less proxy service",
		PersistentPreRun: func(*cobra.Command, []string) {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: levelFlag})))
		},
		PreRun: func(*cobra.Command, []string) {
			context.AfterFunc(ctx, s.Stop)
		},
		RunE: func(*cobra.Command, []string) error {
			err := s.Run(ctx, addrFlag, portFlag, baseFlag, fileFlag, pipeFlag, sizeFlag)
			switch {
			case errors.Is(err, http.ErrServerClosed):
				return nil
			default:
				return err
			}
		},
	}

	c.Flags().VarP(log.NewLogLevel(&levelFlag, slog.LevelInfo), "level", "l", "log level")
	c.Flags().StringVar(&addrFlag, "addr", "", "bind address")
	c.Flags().StringVar(&portFlag, "port", "8080", "bind port")
	c.Flags().StringVar(&baseFlag, "base", "postgres://postgres:postgres@postgres:5432/nocopy", "data base")
	c.Flags().StringVar(&fileFlag, "file", "minio://admin:admin123@minio:9000/nocopy", "file storage")
	c.Flags().StringVar(&pipeFlag, "pipe", "nats://nats", "message broker")
	c.Flags().Int64Var(&sizeFlag, "size", 8*512, "block size")

	err := c.Execute()
	if err != nil {
		os.Exit(1)
	}
}
