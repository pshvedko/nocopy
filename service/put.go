package service

import (
	"crypto/sha1"
	"log/slog"
	"net/http"
	"path"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/broker/message"
	"github.com/pshvedko/nocopy/internal/io"
)

func (s *Block) Put(w http.ResponseWriter, r *http.Request) {
	var blocks []uuid.UUID
	var hashes []api.Hash
	var sizes []int64
	file, err := s.Repository.Put(r.Context(), path.Clean(r.URL.Path))
	for err == nil {
		var size int64
		bid := uuid.New()
		hash := sha1.New()
		//_, err = s.Storage.Store(r.Context(), bid.String(), -1, io.Compressor(r.Body, s.Size, &size, hash))
		size, err = s.Storage.Store(r.Context(), bid.String(), -1, io.TeeLimitReader(r.Body, s.Size, hash))
		if err != nil {
			break
		}
		blocks = append(blocks, bid)
		hashes = append(hashes, hash.Sum([]byte{}))
		sizes = append(sizes, size)
		if size < s.Size {
			var chains []uuid.UUID
			chains, err = s.Repository.Update(r.Context(), file, blocks, hashes, sizes)
			if err == nil {
				w.WriteHeader(http.StatusCreated)
				_, err = s.Broker.Message(r.Context(), "proxy", "file", message.NewBody(api.File{
					Chains: chains,
					Blocks: blocks,
					Hashes: hashes,
					Sizes:  sizes,
				}))
				if err == nil {
					return
				}
			}
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	slog.Error("put", "err", err)
}
