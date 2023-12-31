package io

import (
	"bytes"
	"errors"
	"io"
)

type ReadSeekCloser = io.ReadSeekCloser
type ReadCloser = io.ReadCloser
type Closer = io.Closer
type Writer = io.Writer

var EOF = io.EOF
var EOK error

func Compare(r1, r2 io.Reader) (bool, error) {
	var b1, b2 [512]byte
	for {
		n1, err1 := ReadBytes(r1, b1[:])
		n2, err2 := ReadBytes(r2, b2[:])
		switch {
		case !bytes.Equal(b1[:n1], b2[:n2]):
			return false, nil
		case errors.Is(err1, EOK) && errors.Is(err2, EOK):
			continue
		case errors.Is(err1, EOF) && errors.Is(err2, EOF):
			return true, nil
		default:
			return false, nil
		}
	}
}

func ReadBytes(r io.Reader, b []byte) (n int, err error) {
	for x := 0; err == nil && n < len(b); n += x {
		x, err = r.Read(b[n:])
	}
	return
}

func CopyRange(w io.Writer, r io.ReadSeeker, o, n int64) (int64, error) {
	if o > 0 {
		_, err := r.Seek(o, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
	}
	return io.CopyN(w, r, n)
}
