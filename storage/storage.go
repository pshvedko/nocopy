package storage

import (
	"context"
	"errors"
	"io"
	"net/url"

	"github.com/pshvedko/nocopy/storage/minio"
)

type Storage interface {
	Store(context.Context, string, int64, io.Reader) (int64, error)
	Load(context.Context, string) (io.ReadSeekCloser, error)
	Purge(context.Context, string) error
	Shutdown()
}

func New(name string) (Storage, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "minio":
		return minio.New(u)
	default:
		return nil, errors.New("invalid storage scheme")
	}
}
