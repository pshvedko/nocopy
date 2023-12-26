package repository

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/repository/postgres"
)

type Repository interface {
	Put(context.Context, string) (uuid.UUID, error)
	Get(context.Context, string) (string, time.Time, int64, []uuid.UUID, []int64, error)
	Lookup(context.Context, []byte, int64) ([]uuid.UUID, error)
	Link(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	Break(context.Context, uuid.UUID) ([]uuid.UUID, error)
	Update(context.Context, uuid.UUID, []uuid.UUID, [][]byte, []int64) ([]uuid.UUID, error)
	Delete(context.Context, string) ([]uuid.UUID, error)
	Shutdown()
}

func New(name string) (Repository, error) {
	ur1, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	switch ur1.Scheme {
	case "postgres":
		return postgres.New(ur1)
	default:
		return nil, errors.New("invalid repository scheme")
	}
}
