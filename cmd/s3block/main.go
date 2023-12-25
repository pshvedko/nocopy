package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/pshvedko/nocopy/service"
)

func run() error {
	var addrFlag string
	var portFlag string
	var baseFlag string
	var fileFlag string
	var pipeFlag string
	var sizeFlag int64

	var s service.Block
	defer s.Wait()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c := &cobra.Command{
		Use:  "s3block",
		Long: "S3 (not yet) compatible copy less proxy service",
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

	c.Flags().StringVar(&addrFlag, "addr", "", "bind address")
	c.Flags().StringVar(&portFlag, "port", "8080", "bind port")
	c.Flags().StringVar(&baseFlag, "base", "postgres://postgres:postgres@postgres:5432/nocopy", "data base")
	c.Flags().StringVar(&fileFlag, "file", "minio://admin:admin123@minio:9000/nocopy", "file storage")
	c.Flags().StringVar(&pipeFlag, "pipe", "nats://nats", "message broker")
	c.Flags().Int64Var(&sizeFlag, "size", 8*512, "block size")

	return c.Execute()
}

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}
