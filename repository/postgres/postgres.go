package postgres

import (
	"context"
	"net/url"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/pshvedko/nocopy/repository/block"
)

const (
	getFileQuery  = `insert into files (path) values ($1) on conflict (path) do update set version = files.version+1 returning id`
	addChainQuery = `insert into chains select returning id`
	addBlockQuery = `insert into blocks (id, hash, size) values ($1, $2, $3)`
	addLinkQuery  = `insert into links (chain_id, block_id, ordinal) values ($1, $2, $3)`
	setChainID    = `update files set chain_id = $2 from (select id, chain_id from files where id = $1 for update) as oldies where files.id = oldies.id returning oldies.chain_id`
)

type Repository struct {
	*url.URL
	*sqlx.DB
}

func (r Repository) SetFileID(ctx context.Context, id uuid.UUID, blocks []block.Block) ([]uuid.UUID, error) {
	var chainIDs [2]uuid.UUID
	err := r.GetContext(ctx, &chainIDs[0], addChainQuery)
	if err != nil {
		return nil, err
	}
	for i, part := range blocks {
		_, err = r.ExecContext(ctx, addBlockQuery, part.ID, part.Hash, part.Size)
		if err != nil {
			return nil, err
		}
		_, err = r.ExecContext(ctx, addLinkQuery, chainIDs[0], part.ID, i)
		if err != nil {
			return nil, err
		}
	}
	err = r.GetContext(ctx, &chainIDs[1], setChainID, id, chainIDs[0])
	return chainIDs[:], err
}

func (r Repository) GetFileID(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.GetContext(ctx, &id, getFileQuery, name)
	return id, err
}

func New(ur1 *url.URL) (*Repository, error) {
	db, err := sqlx.Open("pgx", ur1.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(8)
	return &Repository{
		URL: ur1,
		DB:  db,
	}, nil
}
