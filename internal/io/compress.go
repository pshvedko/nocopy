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

func TeeLimitReader(r io.Reader, m int64, w io.Writer) io.Reader {
	return io.TeeReader(io.LimitReader(r, m), w)
}

func Compressor(r io.Reader, m int64, n *int64, w io.Writer) io.Reader {
	var c Compress
	c.r = io.TeeReader(io.LimitReader(r, m), w)
	c.w = gzip.NewWriter(&c.b)
	c.n = n
	return &c
}

type Decompress struct {
	z io.ReadCloser
}

func (d Decompress) Read(b []byte) (n int, err error) {
	return d.z.Read(b)
}

func (d Decompress) Seek(o int64, _ int) (int64, error) {
	return io.CopyN(io.Discard, d.z, o)
}

func (d Decompress) Close() error {
	return d.z.Close()
}

func Decompressor(r io.ReadSeekCloser) (io.ReadSeekCloser, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return Decompress{
		z: z,
	}, nil
}
