package service

import (
	"context"
	"os"
	"sync/atomic"

	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/internal/log"
	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/storage"
)

type Chain struct {
	broker.Broker
	storage.Storage
	repository.Repository
	atomic.Bool
}

func (s *Chain) Run(ctx context.Context, base, file, pipe string) error {
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
	s.Broker.Handle("head", s.HeadQuery)
	s.Broker.UseTransport(log.Transport{Transport: s.Broker.Transport()})
	err = s.Broker.Listen(ctx, "chain", host, "1")
	if err != nil {
		return err
	}
	s.Storage, err = storage.New(file)
	if err != nil {
		return err
	}
	defer s.Storage.Shutdown()
	s.Repository, err = repository.New(base)
	if err != nil {
		return err
	}
	defer s.Repository.Shutdown()
	<-ctx.Done()
	s.Broker.Finish()
	return nil
}

func (s *Chain) Stop() {
	if s.Bool.CompareAndSwap(false, true) {
		return
	}
}
