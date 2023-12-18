package io

import (
	"bytes"
	"compress/gzip"
	"io"
)

type Compress struct {
	r io.Reader
	w interface {
		io.WriteCloser
		Flush() error
	}
	b bytes.Buffer
	n *int64
}

func (c *Compress) Read(p []byte) (n int, err error) {
	for c.b.Len() == 0 {
		var b [1024]byte
		n, err = io.TeeReader(c.r, c.w).Read(b[:])
		*c.n += int64(n)
		if err != nil {
			if err == io.EOF {
				err = c.w.Close()
				if err != nil {
					return
				}
				break
			}
			return
		}
		err = c.w.Flush()
		if err != nil {
			return
		}
	}
	return c.b.Read(p)
}

func Compressor(r io.Reader, m int64, n *int64, w io.Writer) io.Reader {
	var c Compress
	c.r = io.TeeReader(io.LimitReader(r, m), w)
	c.w = gzip.NewWriter(&c.b)
	c.n = n
	return &c
}

func Copy(w io.Writer, r io.Reader) (int64, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = z.Close()
	}()
	return io.Copy(w, z)
}
