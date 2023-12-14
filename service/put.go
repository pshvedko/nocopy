package service

import (
	"context"
	"crypto/sha1"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/util"
)

func (s *Service) Put(w http.ResponseWriter, r *http.Request) {
	var blocks []uuid.UUID
	var hashes [][]byte
	var sizes []int64
	file, err := s.Repository.Put(r.Context(), r.URL.Path)
	for err == nil {
		bid := uuid.New()
		hash := sha1.New()
		body := io.LimitReader(r.Body, s.Size)
		body = io.TeeReader(body, hash)
		var size int64
		size, err = s.Storage.Store(r.Context(), bid.String(), min(r.ContentLength, s.Size), body)
		if err != nil {
			break
		} else if r.ContentLength > 0 {
			r.ContentLength -= size
		}
		blocks = append(blocks, bid)
		hashes = append(hashes, hash.Sum([]byte{}))
		sizes = append(sizes, size)
		if size < s.Size {
			var chains []uuid.UUID
			chains, err = s.Repository.Update(r.Context(), file, blocks, hashes, sizes)
			if err == nil {
				w.WriteHeader(http.StatusCreated)
				s.deduplicate(r.Context(), chains, blocks, hashes, sizes)
				return
			}
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Service) deduplicate(ctx context.Context, chains []uuid.UUID, blocks []uuid.UUID, hashes [][]byte, sizes []int64) {
	defer slog.Warn("copy done")
	slog.Warn("copy", "chains", chains, "blocks", blocks, "sizes", sizes)
	for i := range blocks {
		similarities, err := s.Repository.Lookup(ctx, hashes[i], sizes[i])
		if err != nil {
			slog.Error("copy lookup", "err", err)
			continue
		}
		slog.Warn("copy lookup", "similarities", similarities)
		if len(similarities) == 0 || similarities[0] == blocks[i] {
			continue
		}
		var origin io.ReadSeekCloser
		origin, err = s.Storage.Load(ctx, blocks[i].String())
		if err != nil {
			slog.Error("copy load", "name", blocks[i], "err", err)
			continue
		}
		if func() bool {
			var n int
			for j := range similarities {
				if similarities[j] == blocks[i] {
					return true
				}
				var similar io.ReadCloser
				similar, err = s.Storage.Load(ctx, similarities[j].String())
				if err != nil {
					slog.Error("copy load", "name", similarities[j], "err", err)
					continue
				}
				if n > 0 {
					slog.Warn("copy seek")
					_, _ = origin.Seek(0, 0)
				}
				if util.Compare(origin, similar) {
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
		return
	}
	oldies, err := s.Repository.Break(ctx, chains[1])
	if err != nil {
		slog.Error("copy link", "chain", chains[1], "err", err)
		return
	}
	for i := range oldies {
		if oldies[i] == uuid.Nil {
			continue
		}
		slog.Warn("copy purge", "block", oldies[i])
		_ = s.Storage.Purge(ctx, oldies[i].String())
	}
}
