package service

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/storage"
)

type Options struct {
	Context       context.Context
	ListenAddress string
	FileStorage   string
	DataBase      string
}

type Service struct {
	http.Server
	sync.WaitGroup
	storage.Storage
	repository.Repository
	Size int64
}

func (s *Service) Run(ctx context.Context, addr, port, base, file string) error {
	var err error
	s.Storage, err = storage.New(file)
	if err != nil {
		return err
	}
	s.Repository, err = repository.New(base)
	if err != nil {
		return err
	}
	h := chi.NewRouter()
	h.Put("/*", s.Put)
	s.Size = 512
	s.Handler = h
	s.Addr = net.JoinHostPort(addr, port)
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	s.Add(1)
	go s.WaitForContextCancel(ctx)
	return s.ListenAndServe()
}

func (s *Service) WaitForContextCancel(ctx context.Context) {
	defer s.Done()
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
}
