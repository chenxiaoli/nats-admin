package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("api key not found")

type Key struct {
	ID         uuid.UUID
	AdminID    uuid.UUID
	Name       string
	KeyPrefix  string
	KeyHash    string
	LastUsedAt *time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

const insertSQL = `
INSERT INTO api_keys (admin_id, name, key_prefix, key_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at
`

func (r *PgRepository) Insert(ctx context.Context, k *Key) error {
	return r.pool.QueryRow(ctx, insertSQL, k.AdminID, k.Name, k.KeyPrefix, k.KeyHash).
		Scan(&k.ID, &k.CreatedAt)
}

const lookupHashSQL = `
SELECT id, admin_id FROM api_keys
WHERE key_hash = $1 AND revoked_at IS NULL
`

type Lookup struct {
	KeyID  uuid.UUID
	AdminID uuid.UUID
}

func (r *PgRepository) LookupByHash(ctx context.Context, hash string) (*Lookup, error) {
	l := &Lookup{}
	err := r.pool.QueryRow(ctx, lookupHashSQL, hash).Scan(&l.KeyID, &l.AdminID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return l, err
}

const touchSQL = `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`

func (r *PgRepository) TouchLastUsed(ctx context.Context, id uuid.UUID) {
	_, _ = r.pool.Exec(ctx, touchSQL, id)
}

const listSQL = `
SELECT id, admin_id, name, key_prefix, last_used_at, created_at, revoked_at
FROM api_keys WHERE admin_id = $1 ORDER BY created_at DESC
`

func (r *PgRepository) ListByAdmin(ctx context.Context, adminID uuid.UUID) ([]*Key, error) {
	rows, err := r.pool.Query(ctx, listSQL, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Key
	for rows.Next() {
		k := &Key{}
		if err := rows.Scan(&k.ID, &k.AdminID, &k.Name, &k.KeyPrefix,
			&k.LastUsedAt, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

const getSQL = `
SELECT id, admin_id FROM api_keys WHERE id = $1 AND admin_id = $2
`

func (r *PgRepository) Get(ctx context.Context, id, adminID uuid.UUID) error {
	var gotID, gotAdmin uuid.UUID
	err := r.pool.QueryRow(ctx, getSQL, id, adminID).Scan(&gotID, &gotAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

const revokeSQL = `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`

func (r *PgRepository) Revoke(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, revokeSQL, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
