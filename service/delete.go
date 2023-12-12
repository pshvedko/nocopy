package service

import (
	"net/http"

	"github.com/google/uuid"
)

func (s *Service) Delete(w http.ResponseWriter, r *http.Request) {
	blockIDs, err := s.DeleteBlockID(r.Context(), r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blockIDs) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusNoContent)
		for _, blockID := range blockIDs {
			if blockID == uuid.Nil {
				continue
			}
			_ = s.Purge(r.Context(), blockID.String())
		}
	}
}
