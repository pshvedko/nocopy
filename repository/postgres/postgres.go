package postgres

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/pshvedko/nocopy/api"
)

type Repository struct {
	db *sqlx.DB
}

func (r *Repository) Put(ctx context.Context, path string) (fid uuid.UUID, err error) {
	err = r.db.GetContext(ctx, &fid, "select * from file_insert($1)", path)
	return
}

func (r *Repository) Link(ctx context.Context, cid uuid.UUID, bid1 uuid.UUID, bid2 uuid.UUID) (err error) {
	_, err = r.db.ExecContext(ctx, "call block_update($1, $2, $3)", cid, bid1, bid2)
	return
}

func (r *Repository) Update(ctx context.Context, fid uuid.UUID, blocks []uuid.UUID, hashes []api.Hash, sizes []int64) (chains []uuid.UUID, err error) {
	err = r.db.SelectContext(ctx, &chains, "select * from block_insert($1, $2, $3, $4)", fid, blocks, hashes, sizes)
	return
}

func (r *Repository) Lookup(ctx context.Context, hash api.Hash, size int64) (blocks []uuid.UUID, err error) {
	err = r.db.SelectContext(ctx, &blocks, "select * from block_select($1, $2)", hash, size)
	return
}

func (r *Repository) Break(ctx context.Context, cid uuid.UUID) (blocks []uuid.UUID, err error) {
	err = r.db.SelectContext(ctx, &blocks, "select * from block_delete($1)", cid)
	return
}

func (r *Repository) Delete(ctx context.Context, path string) (blocks []uuid.UUID, err error) {
	err = r.db.SelectContext(ctx, &blocks, "select * from file_delete($1)", path)
	return
}

func (r *Repository) Get(ctx context.Context, name string) (
	mime string,
	date time.Time,
	size int64,
	blocks []uuid.UUID,
	sizes []int64,
	err error) {
	rows, err := r.db.QueryContext(ctx, "select * from file_select($1)", name)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	var n int
	for rows.Next() {
		sizes = append(sizes, size)
		blocks = append(blocks, uuid.UUID{})
		err = rows.Scan(&blocks[n], &sizes[n], &mime, &date)
		if err != nil {
			break
		}
		size += sizes[n]
		n++
	}
	err = rows.Err()
	return
}

func (r *Repository) Shutdown() {
	if r == nil {
		return
	}
	_ = r.db.Close()
}

func New(u *url.URL) (*Repository, error) {
	db, err := sqlx.Open("pgx", u.String())
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	mo, err := strconv.Atoi(u.Query().Get("max_open"))
	if err != nil {
		mo = 16
	}
	mi, err := strconv.Atoi(u.Query().Get("max_idle"))
	if err != nil {
		mo = mo / 4 * 3
	}
	db.SetMaxOpenConns(mo)
	db.SetMaxIdleConns(mi)
	return &Repository{
		db: db,
	}, nil
}
