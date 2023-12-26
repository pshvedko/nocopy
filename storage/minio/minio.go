package minio

import (
	"context"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	*url.URL
	*minio.Client
}

func (s *Storage) Shutdown() {}

func (s *Storage) Purge(ctx context.Context, name string) error {
	err := s.RemoveObject(ctx, s.Path[1:], name, minio.RemoveObjectOptions{})
	return err
}

func (s *Storage) Load(ctx context.Context, name string) (io.ReadSeekCloser, error) {
	r, err := s.Client.GetObject(ctx, s.Path[1:], name, minio.GetObjectOptions{})
	return r, err
}

func (s *Storage) Store(ctx context.Context, name string, size int64, r io.Reader) (int64, error) {
	info, err := s.PutObject(ctx, s.Path[1:], name, r, size, minio.PutObjectOptions{})
	return info.Size, err
}

func New(ur1 *url.URL) (*Storage, error) {
	user := ur1.User.Username()
	secret, _ := ur1.User.Password()
	token := ur1.Query().Get("token")
	secure := ur1.Query().Has("secure")
	client, err := minio.New(ur1.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(user, secret, token),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{
		URL:    ur1,
		Client: client,
	}, nil
}
