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

type Boundary struct {
}

type Block struct {
	http.Server
	sync.WaitGroup
	broker.Broker
	storage.Storage
	repository.Repository
	atomic.Uint64
	Size int64
}

func (b *Block) Run(ctx context.Context, addr, port, base, file, pipe string, size int64) error {
	var err error
	b.Broker, err = broker.New(pipe)
	if err != nil {
		return err
	}
	b.Storage, err = storage.New(file)
	if err != nil {
		return err
	}
	b.Repository, err = repository.New(base)
	if err != nil {
		return err
	}
	h := chi.NewRouter()
	h.Use(middleware.SetHeader("Server", "NoCopy"))
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)
	h.Put("/*", b.Put)
	h.Get("/*", b.Get)
	h.Delete("/*", b.Delete)
	b.Size = size
	b.Handler = h
	b.Addr = net.JoinHostPort(addr, port)
	b.BaseContext = func(net.Listener) context.Context { return ctx }
	b.WaitGroup.Add(1)
	go b.WaitForContextCancel(ctx)
	return b.ListenAndServe()
}

func (b *Block) WaitForContextCancel(ctx context.Context) {
	<-ctx.Done()
	_ = b.Broker.Shutdown(ctx)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = b.Server.Shutdown(ctx)
	b.Done()
	b.Wait()
	_ = b.Storage.Shutdown(ctx)
	_ = b.Repository.Shutdown(ctx)
}

func (b *Block) Context() context.Context {
	return b.BaseContext(nil)
}
