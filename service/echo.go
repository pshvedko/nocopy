package service

import (
	"context"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/internal/log"
)

func (s *Proxy) Echo(ctx context.Context, concurrency, quantity, size int, delay time.Duration, pipe string) error {
	if !s.Bool.CompareAndSwap(false, true) {
		return context.Canceled
	}
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	s.Broker, err = broker.New(pipe, path.Join("echo", host, "1"))
	if err != nil {
		return err
	}
	defer s.Broker.Shutdown()
	s.Broker.UseMiddleware(Authorize{})
	s.Broker.UseTransport(log.Transport{Transport: s.Transport()})
	var w sync.WaitGroup
	q := make(chan struct{}, size)
	s.Broker.Catch("echo", func(ctx context.Context, m message.Message) bool {
		<-q
		w.Done()
		return s.EchoReply(ctx, m)
	})
	slog.Info("echo", "concurrency", concurrency, "quantity", quantity)
	err = s.Broker.Listen(ctx, "echo", host, "1")
	if err != nil {
		return err
	}
	c := make(chan int, concurrency)
	e := make(chan error)
	for i := 0; i < concurrency; i++ {
		go func() {
			var err error
			for n := range c {
				q <- struct{}{}
				w.Add(1)
				_, err = s.Broker.Message(ctx, "proxy", "echo", message.NewBody(api.Echo{
					Serial: n,
					Delay:  delay,
				}))
				if err != nil {
					<-q
					w.Done()
					break
				}
			}
			e <- err
		}()
	}
	t := time.Now()
	for i := 0; i < quantity && concurrency > 0; i++ {
		select {
		case <-ctx.Done():
			slog.Error("echo", "err", ctx.Err())
			quantity = i
		default:
			select {
			case c <- i:
			case err = <-e:
				if err != nil {
					slog.Error("echo", "err", err)
				}
				concurrency--
			}
		}
	}
	close(c)
	for i := 0; i < concurrency; i++ {
		err = <-e
		if err != nil {
			slog.Error("echo", "err", err)
		}
	}
	close(e)
	context.AfterFunc(ctx, func() {
		os.Exit(1)
	})
	w.Wait()
	s.Broker.Finish()
	since := time.Since(t)
	slog.Info("echo", "time", since/time.Duration(quantity))
	return nil
}

func (s *Proxy) EchoQuery(ctx context.Context, m message.Message) (message.Body, error) {
	var echo api.Echo
	err := m.Decode(&echo)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(echo.Delay):
		return m, nil
	}
}

func (s *Proxy) EchoReply(_ context.Context, m message.Message) bool {
	err := m.Decode(&api.Echo{})
	if err != nil {
		slog.Error("echo", "err", err)
	}
	return true
}
