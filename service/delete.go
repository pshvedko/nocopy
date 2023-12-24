package service

import (
	"log/slog"
	"net/http"
	"path"

	"github.com/google/uuid"
)

func (b *Block) Delete(w http.ResponseWriter, r *http.Request) {
	blocks, err := b.Repository.Delete(r.Context(), path.Clean(r.URL.Path))
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
			slog.Warn("delete", "id", id)
			err = b.Storage.Purge(r.Context(), id.String())
			if err != nil {
				slog.Error("delete", "err", err)
			}
		}
	}
}
