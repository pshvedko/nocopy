package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
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
		var part []string
		var status int
		var length int64
		var offsets [][]int64
		var lengths [][]int64
		switch len(ranges) {
		case 0:
			length = size
			status = http.StatusOK
			offsets, lengths = blocksInFull(blocks, sizes)
		case 1:
			length = ranges[0].Length
			status = http.StatusPartialContent
			offsets, lengths = blocksOfRange(blocks, sizes, ranges[0])
		default:
			length = 10000
			status = http.StatusPartialContent
			for n := range ranges {
				o := offsets
				l := lengths
				offsets, lengths = blocksOfRange(blocks, sizes, ranges[n])
				for i := range o {
					offsets[i] = append(o[i], offsets[i]...)
					lengths[i] = append(l[i], lengths[i]...)
				}
				//length += ranges[n].Length
				//length += 1 + 2 + 20 + 1 + 13 + 1 + 1 + 14 + 1 + 5 + 1 + 1 + 1 + 1
				//length += int64(len(p.Mime))
				//length += digits(ranges[n].Start, ranges[n].Start+ranges[n].Length-1)
			}
			part = append(part, fmt.Sprintf("%020d", s.Uint64.Add(1)))
			switch len(mime) {
			case 0:
				part = append(part, "application/octet-stream")
			default:
				part = append(part, mime)
			}
			mime = "multipart/byte" + "ranges; boundary=" + part[0]
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
				var n int64
				for j := range offsets[i] {
					if len(part) == 2 && len(ranges) > 0 && m == 0 {
						err = io.WritePartHeader(w, ranges[0].Start, ranges[0].Length, size, part[0], part[1])
						if err != nil {
							return
						}
						m = ranges[0].Length
						ranges = ranges[1:]
					}
					slog.Warn("get", "id", id, "offset", offsets[i][j]-n, "length", lengths[i][j])
					n, err = io.CopyRange(w, body, offsets[i][j]-n, lengths[i][j])
					if err != nil {
						return
					}
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
				err = io.WritePartFooter(w, part[0])
			}
			if err == nil {
				return
			}
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
