package util

import (
	"hash"
	"io"
)

type HashMaker struct {
	hash.Hash
	io.Reader
}

func (x HashMaker) Read(b []byte) (int, error) {
	n, err := x.Reader.Read(b)
	if n > 0 {
		n, err = x.Write(b[:n])
	}
	return n, err
}

func HashReader(h hash.Hash, r io.Reader) io.Reader {
	return HashMaker{
		Hash:   h,
		Reader: r,
	}
}
