package service

import (
	"io"
	"net/http"
	"strconv"
)

func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	mime, size, blockIDs, err := s.GetBlockID(r.Context(), r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blockIDs) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Add("Content-Length", strconv.FormatInt(size, 10))
		if mime != nil && len(*mime) != 0 {
			w.Header().Add("Content-Type", *mime)
		}
		w.WriteHeader(http.StatusOK)
		for _, blockID := range blockIDs {
			func() {
				var body io.ReadCloser
				body, err = s.Load(r.Context(), blockID.String())
				if err != nil {
					return
				}
				defer func() {
					err = body.Close()
					if err != nil {
						return
					}
				}()
				_, err = io.Copy(w, body)
				if err != nil {
					return
				}
			}()
		}
	}
}
