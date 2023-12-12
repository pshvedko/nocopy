package service

import (
	"crypto/sha1"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/repository/block"
	"github.com/pshvedko/nocopy/repository/util"
)

func (s *Service) Put(w http.ResponseWriter, r *http.Request) {
	var blocks []block.Block
	fileID, err := s.Repository.Put(r.Context(), r.URL.Path)
	for err == nil {
		blockID := uuid.New()
		hash := sha1.New()
		body := io.LimitReader(r.Body, s.Size)
		body = util.HashReader(hash, body)
		var size int64
		size, err = s.Storage.Store(r.Context(), blockID.String(), min(r.ContentLength, s.Size), body)
		if err != nil {
			break
		} else if r.ContentLength > 0 {
			r.ContentLength -= size
		}
		blocks = append(blocks, block.Block{
			ID:   blockID,
			Hash: hash.Sum([]byte{}),
			Size: size,
		})
		if size < s.Size {
			var chainIDs []uuid.UUID
			chainIDs, err = s.Repository.Update(r.Context(), fileID, blocks)
			if err == nil {
				w.WriteHeader(http.StatusCreated)
				s.Add(1)
				go s.AfterPut(chainIDs[0], chainIDs[1])
				return
			}
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Service) AfterPut(chainID0, chainID1 uuid.UUID) {
	defer s.Done()
}
