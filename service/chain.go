package service

import (
	"context"
	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/storage"
	"os"
	"sync"
	"sync/atomic"
)

type Chain struct {
	broker.Broker
	storage.Storage
	repository.Repository
	atomic.Bool
	sync.WaitGroup
}

func (s *Chain) Run(ctx context.Context, base, file, pipe string) error {
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
	s.Broker.Handle("file", s.FileHandle)
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
	return nil
}

func (s *Chain) Stop() {
	if s.Bool.CompareAndSwap(false, true) {
		return
	}
}
