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
	"github.com/pshvedko/nocopy/internal/log"
	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/storage"
)

type Block struct {
	http.Server
	http.Handler
	sync.WaitGroup
	broker.Broker
	storage.Storage
	repository.Repository
	atomic.Bool
	atomic.Uint64
	Size int64
}

func (s *Block) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.WaitGroup.Add(1)
	s.Handler.ServeHTTP(w, r)
	s.WaitGroup.Done()
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
	s.Broker.UseMiddleware(Auth{})
	s.Broker.UseTransport(log.Transport{Transport: s.Broker.Transport()})
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
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)
	h.Use(middleware.SetHeader("Server", "NoCopy"))
	h.Put("/*", s.Put)
	h.Get("/*", s.Get)
	h.Delete("/*", s.Delete)
	h.Head("/*", s.Head)
	s.Handler = h
	s.Server.Handler = s
	s.Size = size
	s.Addr = net.JoinHostPort(addr, port)
	s.BaseContext = func(net.Listener) context.Context { return ctx }
	defer s.WaitGroup.Wait()
	defer s.Broker.Finish()
	return s.Server.ListenAndServe()
}

func (s *Block) Stop() {
	if s.Bool.CompareAndSwap(false, true) {
		return
	}
	defer s.WaitGroup.Done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s.WaitGroup.Add(1)
	_ = s.Server.Shutdown(ctx)
}
