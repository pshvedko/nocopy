package service

import (
	"context"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/storage"
)

type Options struct {
	Context       context.Context
	ListenAddress string
	FileStorage   string
	DataBase      string
}

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

type Boundary struct {
	atomic.Uint64
}

type Block struct {
	http.Server
	broker.Broker
	storage.Storage
	repository.Repository
	atomic.Bool
	sync.WaitGroup
	Boundary
	Size int64
}

func (s *Block) Run(ctx context.Context, addr, port, base, file, pipe string, size int64) error {
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
	s.Broker.Catch("file", s.FileReply)
	err = s.Broker.Listen(ctx, "block", host, "1")
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
	h := chi.NewRouter()
	h.Use(middleware.SetHeader("Server", "NoCopy"))
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)
	h.Put("/*", s.Put)
	h.Get("/*", s.Get)
	h.Delete("/*", s.Delete)
	h.Head("/*", s.Head)
	s.Size = size
	s.Handler = h
	s.Addr = net.JoinHostPort(addr, port)
	s.BaseContext = func(net.Listener) context.Context { return ctx }
	s.Server.RegisterOnShutdown(s.Broker.Finish)
	defer s.WaitGroup.Wait()
	s.WaitGroup.Add(1)
	defer s.WaitGroup.Done()
	return s.Server.ListenAndServe()
}

func (s *Block) Stop() {
	if s.Bool.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = s.Server.Shutdown(ctx)
}
