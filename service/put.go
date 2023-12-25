package service

import (
	"crypto/sha1"
	"log/slog"
	"net/http"
	"path"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/api"
	"github.com/pshvedko/nocopy/service/io"
)

func (b *Block) Put(w http.ResponseWriter, r *http.Request) {
	var blocks []uuid.UUID
	var hashes [][]byte
	var sizes []int64
	file, err := b.Repository.Put(r.Context(), path.Clean(r.URL.Path))
	for err == nil {
		var size int64
		bid := uuid.New()
		hash := sha1.New()
		//_, err = b.Storage.Store(r.Context(), bid.String(), -1, io.Compressor(r.Body, b.Size, &size, hash))
		size, err = b.Storage.Store(r.Context(), bid.String(), -1, io.TeeLimitReader(r.Body, b.Size, hash))
		if err != nil {
			slog.Error("put store", "err", err)
			break
		}
		blocks = append(blocks, bid)
		hashes = append(hashes, hash.Sum([]byte{}))
		sizes = append(sizes, size)
		if size < b.Size {
			var chains []uuid.UUID
			chains, err = b.Repository.Update(r.Context(), file, blocks, hashes, sizes)
			if err == nil {
				w.WriteHeader(http.StatusCreated)
				_, _ = b.Broker.Query(r.Context(), "chain", "file", api.File{
					Chains: chains,
					Blocks: blocks,
					Hashes: hashes,
					Sizes:  sizes,
				})
				return
			}
			slog.Error("put update", "err", err)
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
}
