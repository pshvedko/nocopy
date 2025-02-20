package service

import (
	"context"
	"os"
	"sync/atomic"

	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/internal/log"
)

type Proxy struct {
	broker.Broker
	atomic.Bool
}

func (s *Proxy) Run(ctx context.Context, pipe string) error {
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
	s.Broker.Handle("head", s.HeadQuery)
	s.Broker.Catch("head", s.HeadReply)
	s.Broker.Handle("echo", s.EchoQuery)
	s.Broker.UseTransport(log.Transport{Transport: s.Broker.Transport()})
	err = s.Broker.Listen(ctx, "proxy", host, "1")
	if err != nil {
		return err
	}
	<-ctx.Done()
	s.Broker.Finish()
	return nil
}

func (s *Proxy) Stop() {
	if s.Bool.CompareAndSwap(false, true) {
		return
	}
}
