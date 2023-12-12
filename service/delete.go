package service

import (
	"net/http"

	"github.com/google/uuid"
)

func (s *Service) Delete(w http.ResponseWriter, r *http.Request) {
	blocks, err := s.Repository.Delete(r.Context(), r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blocks) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusNoContent)
		for _, id := range blocks {
			if id == uuid.Nil {
				continue
			}
			_ = s.Storage.Purge(r.Context(), id.String())
		}
	}
}
