package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/service/io"
	"github.com/pshvedko/nocopy/service/multipart"
)

func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	var err error
	var ranges []multipart.Range
	var blocks []uuid.UUID
	var sizes []int64
	var size int64
	var date time.Time
	var mime string
	if mime, date, size, blocks, sizes, err = s.Repository.Get(r.Context(), r.URL.Path); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blocks) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else if ranges, err = multipart.ParseRange(r.Header.Get("Range"), size); err != nil {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
	} else {
		slog.Warn("get", "range", ranges)
		var part []string
		var status int
		var length int64
		var offsets [][]int64
		var lengths [][]int64
		switch len(ranges) {
		case 0:
			length = size
			status = http.StatusOK
			offsets, lengths = multipart.BlocksInFull(blocks, sizes)
		case 1:
			length = ranges[0].Length
			status = http.StatusPartialContent
			offsets, lengths = multipart.BlocksOfRange(blocks, sizes, ranges[0])
		default:
			length = 26
			status = http.StatusPartialContent
			part = append(part, fmt.Sprintf("%020d", s.Uint64.Add(1)))
			switch len(mime) {
			case 0:
				part = append(part, "application/octet-stream")
			default:
				part = append(part, mime)
			}
			mime = "multipart/byte" + "ranges; boundary=" + part[0]
			for n := range ranges {
				o := offsets
				l := lengths
				offsets, lengths = multipart.BlocksOfRange(blocks, sizes, ranges[n])
				for i := range o {
					offsets[i] = append(o[i], offsets[i]...)
					lengths[i] = append(l[i], lengths[i]...)
				}
				length += 64
				length += ranges[n].Length
				length += int64(len(part[1]))
				length += multipart.Digits(ranges[n].Start, ranges[n].Start+ranges[n].Length-1, size)
			}
		}
		w.Header().Add("Content-Length", strconv.FormatInt(length, 10))
		w.Header().Add("Last-Modified", date.Format(time.RFC1123))
		if len(mime) > 0 {
			w.Header().Add("Content-Type", mime)
		}
		w.WriteHeader(status)
		slog.Warn("get", "blocks", blocks, "sizes", sizes)
		slog.Warn("get", "offsets", offsets)
		slog.Warn("get", "lengths", lengths)
		var m int64
		for i, id := range blocks {
			err = func() (err error) {
				if len(offsets[i]) == 0 {
					return
				}
				body, err := s.Storage.Load(r.Context(), id.String())
				if err != nil {
					return
				}
				defer func(c io.Closer) {
					e := c.Close()
					if err == nil {
						err = e
					}
				}(body)
				body, err = io.Decompressor(body)
				if err != nil {
					return
				}
				defer func(c io.Closer) {
					e := c.Close()
					if err == nil {
						err = e
					}
				}(body)
				var z int64
				for j := range offsets[i] {
					if len(part) == 2 && len(ranges) > 0 && m == 0 {
						err = multipart.WriteHeader(w, ranges[0].Start, ranges[0].Length, size, part[0], part[1])
						if err != nil {
							return
						}
						m = ranges[0].Length
						ranges = ranges[1:]
					}
					slog.Warn("get", "id", id, "offset", offsets[i][j]-z, "length", lengths[i][j])
					var n int64
					n, err = io.CopyRange(w, body, offsets[i][j]-z, lengths[i][j])
					if err != nil {
						return
					}
					z += n + offsets[i][j] - z
					m -= n
				}
				return
			}()
			if err != nil {
				break
			}
		}
		if err == nil {
			if len(part) == 2 && len(ranges) == 0 && m == 0 {
				err = multipart.WriteFooter(w, part[0])
			}
			if err == nil {
				return
			}
		}
	}
	slog.Error("get", "err", err)
}
