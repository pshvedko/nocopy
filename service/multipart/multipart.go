package multipart

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/google/uuid"

	"github.com/gotd/contrib/http_range"
)

type Range = http_range.Range

var ErrOverlap = errors.New("ranges overlap")

func ParseRange(bytes string, size int64) (ranges []Range, err error) {
	ranges, err = http_range.ParseRange(bytes, size)
	if err != nil || len(ranges) == 0 {
		return
	}
	slices.SortFunc(ranges, func(a, b Range) int {
		return int(a.Start - b.Start)
	})
	for i, r := range ranges[1:] {
		if ranges[i].Start+ranges[i].Length > r.Start {
			err = ErrOverlap
			break
		}
	}
	return
}

func Digits(int64s ...int64) (n int64) {
	for _, i := range int64s {
		if i == 0 {
			n++
			continue
		}
		for i != 0 {
			i /= 10
			n++
		}
	}
	return
}

func BlocksInFull(blocks []uuid.UUID, sizes []int64) (offsets [][]int64, lengths [][]int64) {
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

func BlocksOfRange(blocks []uuid.UUID, sizes []int64, r Range) (offsets [][]int64, lengths [][]int64) {
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

func WriteHeader(w io.Writer, offset, length, size int64, boundary, mime string) (err error) {
	_, err = fmt.Fprintln(w)
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "--", boundary, "\n")
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(w, "Content-Type:", mime)
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(w, "Content-Range:", "bytes", fmt.Sprintf("%d-%d/%d", offset, offset+length-1, size))
	if err != nil {
		return
	}
	_, err = fmt.Fprintln(w)
	return
}

func WriteFooter(w io.Writer, boundary string) (err error) {
	_, err = fmt.Fprintln(w)
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "--", boundary, "--\n")
	return
}
