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

type Block struct {
	http.Server
	broker.Broker
	storage.Storage
	repository.Repository
	atomic.Uint64
	atomic.Bool
	sync.WaitGroup
	sync.Mutex
	Size int64
}

func (b *Block) Run(ctx context.Context, addr, port, base, file, pipe string, size int64) error {
	time.Sleep(2 * time.Second)
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	if !b.Bool.CompareAndSwap(false, true) {
		return context.Canceled
	}
	b.WaitGroup.Add(1)
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	b.Broker, err = broker.New(pipe)
	if err != nil {
		return err
	}
	err = b.Broker.Listen(ctx, "block", host, "1")
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
	b.Mutex.Unlock()
	defer b.Mutex.Lock()
	return b.ListenAndServe()
}

func (b *Block) Stop() {
	b.Mutex.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if b.Broker != nil {
		_ = b.Broker.Shutdown(ctx)
		_ = b.Server.Shutdown(ctx)
		if b.Storage != nil {
			_ = b.Storage.Shutdown(ctx)
			if b.Repository != nil {
				_ = b.Repository.Shutdown(ctx)
			}
		}
	}
	if !b.Bool.CompareAndSwap(false, true) {
		defer b.WaitGroup.Done()
	}
	b.Mutex.Unlock()
}
