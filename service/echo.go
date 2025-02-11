package service

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/broker/message"
)

const maxInFly = 64 * 1024

func (s *Proxy) Echo(ctx context.Context, concurrency, quantity int, pipe string) error {
	if !s.Bool.CompareAndSwap(false, true) {
		return context.Canceled
	}
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	s.Broker, err = broker.New(pipe)
	if err != nil {
		return err
	}
	defer s.Broker.Shutdown()
	q := make(chan struct{}, maxInFly)
	s.Broker.Catch("echo", func(ctx context.Context, m message.Message) {
		s.EchoReply(ctx, m)
		<-q
	})
	slog.Info("echo", "concurrency", concurrency, "quantity", quantity)
	err = s.Broker.Listen(ctx, "echo", host, "1")
	if err != nil {
		return err
	}
	c := make(chan int, concurrency)
	e := make(chan error)
	t := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			var err error
			for n := range c {
				s.Add(1)
				q <- struct{}{}
				_, err = s.Broker.Message(ctx, "proxy", "echo", api.Echo{Serial: n})
				if err != nil {
					break
				}
			}
			e <- err
		}()
	}
	var done bool
	for i := 0; i < quantity && concurrency > 0 && !done; i++ {
		select {
		case <-ctx.Done():
			done = true
			slog.Warn("interrupt")
		default:
			select {
			case c <- i:
			case <-e:
				concurrency--
			}
		}
	}
	close(c)
	for i := 0; i < concurrency; i++ {
		<-e
	}
	close(e)
	s.WaitGroup.Wait()
	s.Broker.Finish()
	since := time.Since(t)
	slog.Info("echo", "time", since/time.Duration(quantity))
	return nil
}

func (s *Proxy) EchoQuery(_ context.Context, m message.Message) (any, error) {
	return m, nil
}

func (s *Proxy) EchoReply(_ context.Context, m message.Message) {
	err := m.Unmarshal(&api.Echo{})
	if err != nil {
		slog.Error("echo", "err", err)
	}
	s.Done()
}
