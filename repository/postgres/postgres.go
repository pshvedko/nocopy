package postgres

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/pshvedko/nocopy/repository/block"
)

const (
	getFileQuery     = `insert into files (path) values ($1) on conflict (path) do update set version = files.version + 1 returning id`
	addChainQuery    = `insert into chains select returning id`
	addBlockQuery    = `insert into blocks (id, hash, size) values ($1, $2, $3)`
	addLinkQuery     = `insert into links (chain_id, block_id, ordinal) values ($1, $2, $3)`
	setChainID       = `update files set chain_id = $2 from (select id, chain_id from files where id = $1 for update) as oldies where files.id = oldies.id returning oldies.chain_id`
	getBlockQuery    = `select block_id, ordinal, size, mime, created from files join chains on chains.id = files.chain_id join links on chains.id = links.chain_id join blocks on blocks.id = links.block_id where path = $1`
	deleteFileQuery  = `delete from files where path = $1 returning chain_id`
	deleteLinkQuery  = `delete from links where chain_id = $1 returning block_id`
	setBlockQuery    = `update blocks set refer = refer - 1 where id = $1 returning refer`
	deleteBlockQuery = `delete from blocks where id = $1 and refer = 0`
	deleteChainQuery = `delete from chains where id = $1`
)

type Repository struct {
	*url.URL
	*sqlx.DB
}

func (r Repository) DeleteBlockID(ctx context.Context, name string) (blockIDs []uuid.UUID, err error) {
	var blockID uuid.UUID
	var chainID uuid.UUID
	err = r.GetContext(ctx, &chainID, deleteFileQuery, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		return
	}
	rows, err := r.QueryContext(ctx, deleteLinkQuery, chainID)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		err = rows.Scan(&blockID)
		if err != nil {
			break
		}
		blockIDs = append(blockIDs, blockID)
	}
	err = rows.Err()
	if err != nil {
		return
	}
	var i int
	var refer int
	for i, blockID = range blockIDs {
		err = r.GetContext(ctx, &refer, setBlockQuery, blockID)
		if err != nil {
			return
		}
		if refer != 0 {
			blockIDs[i] = uuid.Nil
			continue
		}
		_, err = r.ExecContext(ctx, deleteBlockQuery, blockID)
		if err != nil {
			return
		}
	}
	_, err = r.ExecContext(ctx, deleteChainQuery, chainID)
	return
}

func (r Repository) Shutdown(context.Context) error {
	return r.Close()
}

func (r Repository) GetBlockID(ctx context.Context, name string) (mime string, date time.Time, length int64, blockIDs []uuid.UUID, err error) {
	rows, err := r.QueryContext(ctx, getBlockQuery, name)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
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
	err = rows.Err()
	if err != nil {
		return
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
