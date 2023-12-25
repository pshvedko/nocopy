package service

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/repository"
	"github.com/pshvedko/nocopy/service/io"
	"github.com/pshvedko/nocopy/storage"
)

type Chain struct {
	sync.WaitGroup
	broker.Broker
	storage.Storage
	repository.Repository
}

func (c *Chain) Run(ctx context.Context, base, file, pipe string) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	c.Broker, err = broker.New(pipe)
	if err != nil {
		return err
	}
	c.Storage, err = storage.New(file)
	if err != nil {
		return err
	}
	c.Repository, err = repository.New(base)
	if err != nil {
		return err
	}
	c.Handle("file", c.FileHandle)
	defer c.Shutdown()
	c.Add(1)
	panic(1) // TODO
	return c.Listen(ctx, "chain", host, "1")
}

func (c *Chain) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c.Done()
	c.Wait()
	_ = c.Storage.Shutdown(ctx)
	_ = c.Repository.Shutdown(ctx)
}

func (c *Chain) FileHandle(ctx context.Context, q message.Query) (message.Reply, error) {
	c.Add(1)
	defer c.Done()
	slog.Warn(q.BY(), "id", q.ID(), "at", q.AT(), "re", q.RE())
	var file api.File
	err := q.Unmarshal(&file)
	if err != nil {
		return nil, err
	}
	err = c.File(ctx, file.Chains, file.Blocks, file.Hashes, file.Sizes)
	return nil, err
}

func (c *Chain) File(ctx context.Context, chains []uuid.UUID, blocks []uuid.UUID, hashes [][]byte, sizes []int64) error {
	slog.Warn("copy", "chains", chains, "blocks", blocks, "sizes", sizes)
	for i := range blocks {
		similarities, err := c.Repository.Lookup(ctx, hashes[i], sizes[i])
		if err != nil {
			slog.Error("copy lookup", "err", err)
			continue
		}
		if len(similarities) == 0 || similarities[0] == blocks[i] {
			continue
		}
		var origin io.ReadSeekCloser
		origin, err = c.Storage.Load(ctx, blocks[i].String())
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
				similar, err = c.Storage.Load(ctx, similarities[j].String())
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
					err = c.Repository.Link(ctx, chains[0], blocks[i], similarities[j])
					if err == nil {
						slog.Warn("copy purge", "block", blocks[i])
						_ = origin.Close()
						_ = similar.Close()
						_ = c.Storage.Purge(ctx, blocks[i].String())
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
	oldies, err := c.Repository.Break(ctx, chains[1])
	if err != nil {
		slog.Error("copy link", "chain", chains[1], "err", err)
		return err
	}
	for i := range oldies {
		if oldies[i] == uuid.Nil {
			continue
		}
		slog.Warn("copy purge", "block", oldies[i])
		_ = c.Storage.Purge(ctx, oldies[i].String())
	}
	return nil
}
