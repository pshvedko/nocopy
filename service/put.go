package service

import (
	"crypto/sha1"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/repository/block"
	"github.com/pshvedko/nocopy/repository/util"
)

func (s *Service) Put(w http.ResponseWriter, r *http.Request) {
	slog.Warn("put", "size", r.ContentLength, "path", r.URL.Path)
	fileID, err := s.GetFileID(r.Context(), r.URL.Path)
	var blocks []block.Block
	for size, length := r.ContentLength, r.ContentLength; length != 0; length -= size {
		blockID := uuid.New()
		hash := sha1.New()
		size, err = s.Store(r.Context(), blockID.String(), min(length, s.Size),
			util.HashReader(hash, io.LimitReader(r.Body, s.Size)))
		if err != nil {
			break
		}
		blocks = append(blocks, block.Block{
			ID:   blockID,
			Hash: hash.Sum([]byte{}),
			Size: size,
		})
		slog.Warn("put", "size", size, "block_id", blockID)
		if size < s.Size {
			var chainIDs []uuid.UUID
			chainIDs, err = s.SetFileID(r.Context(), fileID, blocks)
			if err == nil {
				slog.Warn("put", "id", fileID, "chain_id", chainIDs)
				return
			}
		}
	}
	w.WriteHeader(http.StatusInsufficientStorage)
	slog.Error("put", "err", err)
}
