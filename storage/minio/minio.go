package minio

import (
	"context"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client *minio.Client
	path   string
}

func (s *Storage) Shutdown() {}

func (s *Storage) Purge(ctx context.Context, name string) error {
	err := s.client.RemoveObject(ctx, s.path[1:], name, minio.RemoveObjectOptions{})
	return err
}

func (s *Storage) Load(ctx context.Context, name string) (io.ReadSeekCloser, error) {
	r, err := s.client.GetObject(ctx, s.path[1:], name, minio.GetObjectOptions{})
	return r, err
}

func (s *Storage) Store(ctx context.Context, name string, size int64, r io.Reader) (int64, error) {
	info, err := s.client.PutObject(ctx, s.path[1:], name, r, size, minio.PutObjectOptions{})
	return info.Size, err
}

func New(u *url.URL) (*Storage, error) {
	user := u.User.Username()
	secret, _ := u.User.Password()
	token := u.Query().Get("token")
	secure := u.Query().Has("secure")
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(user, secret, token),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{
		path:   u.Path,
		client: client,
	}, nil
}
