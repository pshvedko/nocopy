package util

import (
	"bytes"
	"errors"
	"io"
)

func Compare(r1, r2 io.Reader) bool {
	var b1, b2 [512]byte
	for {
		n1, err1 := ReadBytes(r1, b1[:])
		n2, err2 := ReadBytes(r2, b2[:])
		switch {
		case !bytes.Equal(b1[:n1], b2[:n2]):
			return false
		case errors.Is(err1, io.EOF) && errors.Is(err2, io.EOF):
			return true
		default:
			return false
		}
	}
}

func ReadBytes(r io.Reader, b []byte) (n int, err error) {
	for x := 0; err == nil && n < len(b); n += x {
		x, err = r.Read(b[n:])
	}
	return n, err
}
