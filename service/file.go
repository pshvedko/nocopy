package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/internal/io"
)

func (s *Proxy) FileQuery(ctx context.Context, m message.Message) (message.Body, error) {
	_, err := s.Broker.Forward(ctx, "chain", m)
	return nil, err
}

func (s *Proxy) FileReply(ctx context.Context, m message.Message) bool {
	_, _ = s.Broker.Backward(ctx, m)
	return true
}

func (s *Block) FileReply(_ context.Context, m message.Message) bool {
	var reply api.FileReply
	err := m.Decode(&reply)
	if err != nil {
		slog.Warn("file", "err", err)
	} else {
		slog.Warn("file", "time", reply.Time)
	}
	return true
}

func (s *Chain) FileQuery(ctx context.Context, m message.Message) (message.Body, error) {
	var file api.File
	err := m.Decode(&file)
	if err != nil {
		return nil, err
	}
	slog.Warn("file", "file", file)
	err = s.File(ctx, file.Chains, file.Blocks, file.Hashes, file.Sizes)
	if err != nil {
		return nil, err
	}
	return message.NewBody(api.FileReply{Time: time.Now()}), nil
}

func (s *Chain) File(ctx context.Context, chains []uuid.UUID, blocks []uuid.UUID, hashes [][]byte, sizes []int64) error {
	for i := range blocks {
		similarities, err := s.Repository.Lookup(ctx, hashes[i], sizes[i])
		if err != nil {
			slog.Error("file lookup", "err", err)
			continue
		}
		if len(similarities) == 0 || similarities[0] == blocks[i] {
			continue
		}
		var origin io.ReadSeekCloser
		origin, err = s.Storage.Load(ctx, blocks[i].String())
		if err != nil {
			slog.Error("file load", "name", blocks[i], "err", err)
			continue
		}
		slog.Warn("file lookup", "similarities", similarities, "i", i)
		if func() bool {
			var n int
			for j := range similarities {
				if similarities[j] == blocks[i] {
					return true
				}
				var similar io.ReadCloser
				similar, err = s.Storage.Load(ctx, similarities[j].String())
				if err != nil {
					slog.Error("file load", "id", similarities[j], "err", err)
					continue
				}
				if n > 0 {
					slog.Warn("file seek")
					_, _ = origin.Seek(0, 0)
				}
				var ok bool
				ok, err = io.Compare(origin, similar)
				if err != nil {
					slog.Error("file equal", "j", j, "err", err)
				} else if ok {
					slog.Warn("file equal", "blocks", []uuid.UUID{blocks[i], similarities[j]})
					err = s.Repository.Link(ctx, chains[0], blocks[i], similarities[j])
					if err == nil {
						slog.Warn("file purge", "block", blocks[i])
						_ = origin.Close()
						_ = similar.Close()
						_ = s.Storage.Purge(ctx, blocks[i].String())
						return false
					}
					slog.Error("file link", "chain", chains[0], "blocks", []uuid.UUID{blocks[i], similarities[j]}, "err", err)
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
		slog.Error("file link", "chain", chains[1], "err", err)
		return err
	}
	for i := range oldies {
		if oldies[i] == uuid.Nil {
			continue
		}
		slog.Warn("file purge", "block", oldies[i])
		_ = s.Storage.Purge(ctx, oldies[i].String())
	}
	return nil
}
