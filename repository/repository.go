package repository

import (
	"context"
	"errors"
	"net/url"

	"github.com/google/uuid"

	"github.com/pshvedko/nocopy/repository/block"
	"github.com/pshvedko/nocopy/repository/postgres"
)

type Repository interface {
	AddFileID(context.Context, string) (uuid.UUID, error)
	GetBlockID(context.Context, string) (*string, int64, []uuid.UUID, error)
	SetChainID(context.Context, uuid.UUID, []block.Block) ([]uuid.UUID, error)
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
