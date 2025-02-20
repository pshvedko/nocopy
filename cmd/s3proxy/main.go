package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/pshvedko/nocopy/internal/log"
	"github.com/pshvedko/nocopy/service"
)

func main() {
	var pipeFlag string
	var concurrencyFlag int
	var quantityFlag int
	var levelFlag slog.Level

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s := service.Proxy{}
	c := &cobra.Command{
		Use:  "s3proxy",
		Long: "S3 (not yet) compatible copy less proxy service",
		PersistentPreRun: func(*cobra.Command, []string) {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: levelFlag})))
		},
		PreRun: func(*cobra.Command, []string) {
			context.AfterFunc(ctx, s.Stop)
		},
		RunE: func(*cobra.Command, []string) error {
			return s.Run(ctx, pipeFlag)
		},
	}

	var pressureFlag int
	var delayFlag time.Duration

	t := &cobra.Command{
		Use:   "echo",
		Short: "Echo performance",
		PreRun: func(*cobra.Command, []string) {
			context.AfterFunc(ctx, s.Stop)
		},
		RunE: func(*cobra.Command, []string) error {
			return s.Echo(ctx, concurrencyFlag, quantityFlag, pressureFlag, delayFlag, pipeFlag)
		},
	}

	c.PersistentFlags().VarP(log.NewLogLevel(&levelFlag, slog.LevelInfo), "level", "l", "log level")
	c.PersistentFlags().StringVar(&pipeFlag, "pipe", "nats://nats", "message broker")
	t.Flags().IntVarP(&concurrencyFlag, "concurrency", "c", 1, "concurrency")
	t.Flags().IntVarP(&quantityFlag, "quantity", "n", 1, "quantity")
	t.Flags().IntVarP(&pressureFlag, "pressure", "p", 64*1024, "pressure")
	t.Flags().DurationVarP(&delayFlag, "delay", "d", 0, "delay")
	c.AddCommand(t)

	err := c.Execute()
	if err != nil {
		os.Exit(1)
	}
}
