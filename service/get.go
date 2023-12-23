package service

import (
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gotd/contrib/http_range"

	"github.com/pshvedko/nocopy/service/io"
)

func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	var err error
	var ranges []http_range.Range
	var blocks []uuid.UUID
	var sizes []int64
	var size int64
	var date time.Time
	var mime string
	if mime, date, size, blocks, sizes, err = s.Repository.Get(r.Context(), r.URL.Path); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(blocks) == 0 {
		w.WriteHeader(http.StatusNotFound)
	} else if ranges, err = http_range.ParseRange(r.Header.Get("Range"), size); err != nil {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
	} else {
		slog.Warn("get", "range", ranges)
		status := http.StatusOK
		var offsets [][]int64
		var lengths [][]int64
		switch len(ranges) {
		case 0:
			offsets, lengths = blocksInFull(blocks, sizes)
			break
		case 1:
			offsets, lengths = blocksOfRange(blocks, sizes, ranges[0])
			size = ranges[0].Length
			ranges = ranges[1:]
			fallthrough
		default:
			status = http.StatusPartialContent
		}
		w.Header().Add("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Add("Last-Modified", date.Format(time.RFC1123))
		if len(mime) != 0 {
			w.Header().Add("Content-Type", mime)
		}
		w.WriteHeader(status)
		slog.Warn("get", "blocks", blocks, "sizes", sizes)
		slog.Warn("get", "offsets", offsets)
		slog.Warn("get", "lengths", lengths)
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
				for j := range offsets[i] {
					slog.Warn("get", "id", id, "offset", offsets[i][j], "length", lengths[i][j])
					_, err = io.CopyRange(w, body, offsets[i][j], lengths[i][j])
					if err != nil {
						break
					}
				}
				return
			}()
			if err != nil {
				break
			}
		}
		if err == nil {
			return
		}
	}
	slog.Error("get", "err", err)
}

func blocksInFull(blocks []uuid.UUID, sizes []int64) (offsets [][]int64, lengths [][]int64) {
	offsets = make([][]int64, len(blocks))
	lengths = make([][]int64, len(blocks))
	for i := range blocks {
		if sizes[i] == 0 {
			continue
		}
		offsets[i] = append(offsets[i], 0)
		lengths[i] = append(lengths[i], sizes[i])
	}
	return
}

func blocksOfRange(blocks []uuid.UUID, sizes []int64, r http_range.Range) (offsets [][]int64, lengths [][]int64) {
	var i int
	var size int64
	offsets = make([][]int64, len(blocks))
	lengths = make([][]int64, len(blocks))
	for i = range blocks {
		size += sizes[i]
		if r.Start < size {
			offsets[i] = append(offsets[i], r.Start-size+sizes[i])
			break
		}
	}
	for {
		if r.Start+r.Length <= size {
			lengths[i] = append(lengths[i], min(r.Length, r.Length-size+sizes[i]+r.Start))
			break
		}
		lengths[i] = append(lengths[i], sizes[i]-offsets[i][0])
		i++
		if i == len(blocks) {
			break
		}
		size += sizes[i]
		offsets[i] = append(offsets[i], 0)
	}
	return
}
