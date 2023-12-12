package io

import (
	"io"
)

type Writer = io.Writer
type WriteCloser = io.WriteCloser
type WriteSeeker = io.WriteSeeker
type Reader = io.Reader
type ReadWriter = io.ReadWriter
type ReadSeekCloser = io.ReadSeekCloser
type ReadCloser = io.ReadCloser
type ReadSeeker = io.ReadSeeker

var EOF = io.EOF

type Copier struct {
	io.Writer
	io.Reader
}

func (x Copier) Read(b []byte) (int, error) {
	n, err := x.Reader.Read(b)
	if n > 0 {
		n, err = x.Writer.Write(b[:n])
	}
	return n, err
}

func CopyReader(r io.Reader, w io.Writer) io.Reader { return Copier{w, r} }

func LimitReader(r io.Reader, n int64) io.Reader { return io.LimitReader(r, n) }

func Compare(a, b io.Reader) bool {

	panic(1)
}
