package service

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/service/io"
	"github.com/pshvedko/nocopy/storage"
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

func (s *Chain) FileHandle(ctx context.Context, q message.Query) (message.Reply, error) {
	s.Add(1)
	defer s.Done()
	slog.Warn(q.BY(), "id", q.ID(), "at", q.AT(), "re", q.RE())
	var file api.File
	err := q.Unmarshal(&file)
	if err != nil {
		return nil, err
	}
	err = s.File(ctx, file.Chains, file.Blocks, file.Hashes, file.Sizes)
	return nil, err
}

func (s *Chain) File(ctx context.Context, chains []uuid.UUID, blocks []uuid.UUID, hashes [][]byte, sizes []int64) error {
	slog.Warn("copy", "chains", chains, "blocks", blocks, "sizes", sizes)
	for i := range blocks {
		similarities, err := s.Repository.Lookup(ctx, hashes[i], sizes[i])
		if err != nil {
			slog.Error("copy lookup", "err", err)
			continue
		}
		if len(similarities) == 0 || similarities[0] == blocks[i] {
			continue
		}
		var origin io.ReadSeekCloser
		origin, err = s.Storage.Load(ctx, blocks[i].String())
		if err != nil {
			slog.Error("copy load", "name", blocks[i], "err", err)
			continue
		}
		slog.Warn("copy lookup", "similarities", similarities, "i", i)
		if func() bool {
			var n int
			for j := range similarities {
				if similarities[j] == blocks[i] {
					return true
				}
				var similar io.ReadCloser
				similar, err = s.Storage.Load(ctx, similarities[j].String())
				if err != nil {
					slog.Error("copy load", "id", similarities[j], "err", err)
					continue
				}
				if n > 0 {
					slog.Warn("copy seek")
					_, _ = origin.Seek(0, 0)
				}
				var ok bool
				ok, err = io.Compare(origin, similar)
				if err != nil {
					slog.Error("copy equal", "j", j, "err", err)
				} else if ok {
					slog.Warn("copy equal", "blocks", []uuid.UUID{blocks[i], similarities[j]})
					err = s.Repository.Link(ctx, chains[0], blocks[i], similarities[j])
					if err == nil {
						slog.Warn("copy purge", "block", blocks[i])
						_ = origin.Close()
						_ = similar.Close()
						_ = s.Storage.Purge(ctx, blocks[i].String())
						return false
					}
					slog.Error("copy link", "chain", chains[0], "blocks", []uuid.UUID{blocks[i], similarities[j]}, "err", err)
				}
				_ = similar.Close()
				n++
			}
			return true
		}() {
			_ = origin.Close()
		}
	}
	if chains[1] == uuid.Nil {
		return nil
	}
	oldies, err := s.Repository.Break(ctx, chains[1])
	if err != nil {
		slog.Error("copy link", "chain", chains[1], "err", err)
		return err
	}
	for i := range oldies {
		if oldies[i] == uuid.Nil {
			continue
		}
		slog.Warn("copy purge", "block", oldies[i])
		_ = s.Storage.Purge(ctx, oldies[i].String())
	}
	return nil
}
