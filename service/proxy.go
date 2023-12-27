package service

import (
	"context"
	"os"
	"sync"
	"sync/atomic"

	"github.com/pshvedko/nocopy/broker"
)

type Proxy struct {
	broker.Broker
	atomic.Bool
	sync.WaitGroup
}

func (s *Proxy) Run(ctx context.Context, pipe string) error {
	defer s.WaitGroup.Wait()
	s.WaitGroup.Add(1)
	defer s.WaitGroup.Done()
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
	s.Broker.Handle("file", s.FileQuery)
	s.Broker.Catch("file", s.FileReply)
	err = s.Broker.Listen(ctx, "proxy", host, "1")
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (s *Proxy) Stop() {
	if s.Bool.CompareAndSwap(false, true) {
		return
	}
}
