package postgres

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	query01 = `insert into files (path) values ($1) on conflict (path) do update set version = files.version + 1 returning id`
	query02 = `insert into chains select returning id`
	query03 = `insert into blocks (id, hash, size) values ($1, $2, $3)`
	query04 = `insert into links (chain_id, block_id, ordinal) values ($1, $2, $3)`
	query05 = `update files set chain_id = $2 from (select id, chain_id from files where id = $1 for update) as oldies where files.id = oldies.id returning oldies.chain_id`
	query06 = `select block_id, ordinal, size, mime, created from files join chains on chains.id = files.chain_id join links on chains.id = links.chain_id join blocks on blocks.id = links.block_id where path = $1`
	query07 = `delete from files where path = $1 returning chain_id`
	query08 = `delete from links where chain_id = $1 returning block_id`
	query09 = `update blocks set refer = refer + $2 where id = $1 returning refer`
	query10 = `delete from blocks where id = $1 and refer = $2`
	query11 = `delete from chains where id = $1`
	query12 = `select id from blocks where hash = $1 and size = $2 and id <> $3;`
	query13 = `update links set block_id = $3 where chain_id = $1 and block_id = $2`
)

type Repository struct {
	*url.URL
	*sqlx.DB
}

func (r Repository) Refer(ctx context.Context, cid uuid.UUID, bid1 uuid.UUID, bid2 uuid.UUID) error {
	var refer int
	err := r.GetContext(ctx, &refer, query09, bid2, +1)
	if err != nil {
		return err
	}
	_, err = r.ExecContext(ctx, query13, cid, bid1, bid2)
	if err != nil {
		return err
	}
	_, err = r.ExecContext(ctx, query10, bid1, 1)
	return err
}

func (r Repository) Lookup(ctx context.Context, bid uuid.UUID, hash []byte, size int64) (blocks []uuid.UUID, err error) {
	rows, err := r.QueryContext(ctx, query12, hash, size, bid)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		err = rows.Scan(&bid)
		if err != nil {
			break
		}
		blocks = append(blocks, bid)
	}
	err = rows.Err()
	return
}

func (r Repository) Delete(ctx context.Context, name string) (blocks []uuid.UUID, err error) {
	var bid uuid.UUID
	var cid uuid.UUID
	err = r.GetContext(ctx, &cid, query07, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		return
	}
	rows, err := r.QueryContext(ctx, query08, cid)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		err = rows.Scan(&bid)
		if err != nil {
			break
		}
		blocks = append(blocks, bid)
	}
	err = rows.Err()
	if err != nil {
		return
	}
	var i int
	var refer int
	for i, bid = range blocks {
		err = r.GetContext(ctx, &refer, query09, bid, -1)
		if err != nil {
			return
		}
		if refer != 0 {
			blocks[i] = uuid.Nil
			continue
		}
		_, err = r.ExecContext(ctx, query10, bid, 0)
		if err != nil {
			return
		}
	}
	_, err = r.ExecContext(ctx, query11, cid)
	return
}

func (r Repository) Shutdown(context.Context) error {
	return r.Close()
}

func (r Repository) Get(ctx context.Context, name string) (mime string, date time.Time, length int64, blocks []uuid.UUID, err error) {
	rows, err := r.QueryContext(ctx, query06, name)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	var n int
	var size int64
	var ordinals []int
	for rows.Next() {
		blocks = append(blocks, uuid.UUID{})
		ordinals = append(ordinals, n)
		err = rows.Scan(&blocks[n], &ordinals[n], &size, &mime, &date)
		if err != nil {
			break
		}
		length += size
	}
	err = rows.Err()
	if err != nil {
		return
	}
	for i, j := range ordinals {
		if i > j {
			blocks[i], blocks[j] = blocks[j], blocks[i]
		}
	}
	return
}

func (r Repository) Update(ctx context.Context, fid uuid.UUID, blocks []uuid.UUID, hashes [][]byte, sizes []int64) ([]uuid.UUID, error) {
	var chains [2]uuid.UUID
	err := r.GetContext(ctx, &chains[0], query02)
	if err != nil {
		return nil, err
	}
	for i := range blocks {
		_, err = r.ExecContext(ctx, query03, blocks[i], hashes[i], sizes[i])
		if err != nil {
			return nil, err
		}
		_, err = r.ExecContext(ctx, query04, chains[0], blocks[i], i)
		if err != nil {
			return nil, err
		}
	}
	err = r.GetContext(ctx, &chains[1], query05, fid, chains[0])
	return chains[:], err
}

func (r Repository) Put(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.GetContext(ctx, &id, query01, name)
	return id, err
}

func New(ur1 *url.URL) (*Repository, error) {
	db, err := sqlx.Open("pgx", ur1.String())
	if err != nil {
		return nil, err
	}
	mo, err := strconv.Atoi(ur1.Query().Get("max_open"))
	if err != nil {
		mo = 16
	}
	mi, err := strconv.Atoi(ur1.Query().Get("max_idle"))
	if err != nil {
		mo = mo / 4 * 3
	}
	db.SetMaxOpenConns(mo)
	db.SetMaxIdleConns(mi)
	return &Repository{
		URL: ur1,
		DB:  db,
	}, nil
}
