package postgres

import (
	"context"
	"net/url"
	"time"

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
	getBlockQuery = `select block_id, ordinal, size, mime, created from files join chains on chains.id = files.chain_id join links on chains.id = links.chain_id join blocks on blocks.id = links.block_id where path = $1`
)

type Repository struct {
	*url.URL
	*sqlx.DB
}

func (r Repository) GetBlockID(ctx context.Context, name string) (mime string, date time.Time, length int64, blockIDs []uuid.UUID, err error) {
	var rows *sqlx.Rows
	rows, err = r.QueryxContext(ctx, getBlockQuery, name)
	if err != nil {
		return
	}
	defer func() {
		err2 := rows.Err()
		if err2 != nil {
			err = err2
		}
		err2 = rows.Close()
		if err2 != nil {
			err = err2
		}
	}()
	var ordinals []int
	for rows.Next() {
		var blockID uuid.UUID
		var ordinal int
		var size int64
		err = rows.Scan(&blockID, &ordinal, &size, &mime, &date)
		if err != nil {
			break
		}
		length += size
		blockIDs = append(blockIDs, blockID)
		ordinals = append(ordinals, ordinal)
	}
	for i, j := range ordinals {
		if i > j {
			blockIDs[i], blockIDs[j] = blockIDs[j], blockIDs[i]
		}
	}
	return
}

func (r Repository) SetChainID(ctx context.Context, id uuid.UUID, blocks []block.Block) ([]uuid.UUID, error) {
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

func (r Repository) AddFileID(ctx context.Context, name string) (uuid.UUID, error) {
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
