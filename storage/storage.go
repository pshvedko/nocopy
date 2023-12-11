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
}

func New(name string) (Storage, error) {
	ur1, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch ur1.Scheme {
	case "minio":
		return minio.New(ur1)
	default:
		return nil, errors.New("invalid storage scheme")
	}
}
