package service

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pshvedko/nocopy/service/io"
)

func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	mime, date, size, blocks, err := s.Repository.Get(r.Context(), r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blocks) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Add("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Add("Last-Modified", date.Format(time.RFC1123))
		if len(mime) != 0 {
			w.Header().Add("Content-Type", mime)
		}
		w.WriteHeader(http.StatusOK)
		for _, id := range blocks {
			if func() (err error) {
				body, err := s.Storage.Load(r.Context(), id.String())
				if err != nil {
					return
				}
				defer func() {
					err = body.Close()
				}()
				_, err = io.Copy(w, body)
				return
			}() != nil {
				break
			}
		}
	}
}
