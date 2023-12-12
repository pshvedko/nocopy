package service

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	mime, date, size, blockIDs, err := s.GetBlockID(r.Context(), r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blockIDs) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Add("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Add("Last-Modified", date.Format(time.RFC1123))
		if len(mime) != 0 {
			w.Header().Add("Content-Type", mime)
		}
		w.WriteHeader(http.StatusOK)
		for _, blockID := range blockIDs {
			if func() (err error) {
				body, err := s.Load(r.Context(), blockID.String())
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
