package service

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/storage"
)

type Options struct {
	Context       context.Context
	ListenAddress string
	FileStorage   string
	DataBase      string
}

type Boundary struct {
}

type Service struct {
	http.Server
	sync.WaitGroup
	storage.Storage
	repository.Repository
	atomic.Uint64
	Size int64
}

func (s *Service) Run(ctx context.Context, addr, port, base, file string, size int64) error {
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
	h.Use(middleware.SetHeader("Server", "NoCopy"))
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)
	h.Put("/*", s.Put)
	h.Get("/*", s.Get)
	h.Delete("/*", s.Delete)
	s.Size = size
	s.Handler = h
	s.Addr = net.JoinHostPort(addr, port)
	s.BaseContext = func(net.Listener) context.Context { return ctx }
	s.WaitGroup.Add(1)
	go s.WaitForContextCancel(ctx)
	return s.ListenAndServe()
}

func (s *Service) WaitForContextCancel(ctx context.Context) {
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = s.Server.Shutdown(ctx)
	s.Done()
	s.Wait()
	_ = s.Storage.Shutdown(ctx)
	_ = s.Repository.Shutdown(ctx)
}

func (s *Service) Context() context.Context {
	return s.BaseContext(nil)
}
