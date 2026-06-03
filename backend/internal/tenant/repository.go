package tenant

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("tenant not found")

type Tenant struct {
	ID                 uuid.UUID
	Name               string
	Slug               string
	AccountPublicKey   string
	AccountJWT         string
	JSMemoryStorage    int64
	JSDiskStorage      int64
	JSMaxStreams       int32
	JSMaxConsumers     int32
	MaxConnections     int32
	MaxSubscriptions   int32
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const insertTenantSQL = `
INSERT INTO tenants (
  name, slug, account_public_key, account_jwt,
  js_max_memory_storage, js_max_disk_storage, js_max_streams, js_max_consumers,
  max_connections, max_subscriptions
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, created_at, updated_at
`

func (r *Repository) Create(ctx context.Context, t *Tenant) error {
	return r.pool.QueryRow(ctx, insertTenantSQL,
		t.Name, t.Slug, t.AccountPublicKey, t.AccountJWT,
		t.JSMemoryStorage, t.JSDiskStorage, t.JSMaxStreams, t.JSMaxConsumers,
		t.MaxConnections, t.MaxSubscriptions,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

const selectTenantSQL = `
SELECT id, name, slug, account_public_key, account_jwt,
       js_max_memory_storage, js_max_disk_storage, js_max_streams, js_max_consumers,
       max_connections, max_subscriptions, status, created_at, updated_at
FROM tenants
WHERE id = $1 AND status <> 'deleted'
`

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	t := &Tenant{}
	err := r.pool.QueryRow(ctx, selectTenantSQL, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.AccountPublicKey, &t.AccountJWT,
		&t.JSMemoryStorage, &t.JSDiskStorage, &t.JSMaxStreams, &t.JSMaxConsumers,
		&t.MaxConnections, &t.MaxSubscriptions, &t.Status,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

const listTenantsSQL = `
SELECT id, name, slug, account_public_key, account_jwt,
       js_max_memory_storage, js_max_disk_storage, js_max_streams, js_max_consumers,
       max_connections, max_subscriptions, status, created_at, updated_at
FROM tenants
WHERE status <> 'deleted'
ORDER BY created_at DESC
`

func (r *Repository) List(ctx context.Context) ([]*Tenant, error) {
	rows, err := r.pool.Query(ctx, listTenantsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Tenant
	for rows.Next() {
		t := &Tenant{}
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.AccountPublicKey, &t.AccountJWT,
			&t.JSMemoryStorage, &t.JSDiskStorage, &t.JSMaxStreams, &t.JSMaxConsumers,
			&t.MaxConnections, &t.MaxSubscriptions, &t.Status,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

const updateTenantJWTAndLimitsSQL = `
UPDATE tenants SET
  account_jwt = $2,
  js_max_memory_storage = $3,
  js_max_disk_storage = $4,
  js_max_streams = $5,
  js_max_consumers = $6,
  max_connections = $7,
  max_subscriptions = $8,
  updated_at = NOW()
WHERE id = $1
RETURNING updated_at
`

func (r *Repository) UpdateJWTAndLimits(ctx context.Context, t *Tenant) error {
	return r.pool.QueryRow(ctx, updateTenantJWTAndLimitsSQL,
		t.ID, t.AccountJWT,
		t.JSMemoryStorage, t.JSDiskStorage, t.JSMaxStreams, t.JSMaxConsumers,
		t.MaxConnections, t.MaxSubscriptions,
	).Scan(&t.UpdatedAt)
}

const setStatusSQL = `UPDATE tenants SET status = $2, updated_at = NOW() WHERE id = $1`

func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, setStatusSQL, id, status)
	return err
}

const deleteTenantSQL = `UPDATE tenants SET status = 'deleted', updated_at = NOW() WHERE id = $1`

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, deleteTenantSQL, id)
	return err
}

const insertSeedSQL = `
INSERT INTO tenant_keys (tenant_id, encrypted_seed, nonce, key_version)
VALUES ($1, $2, $3, 1)
ON CONFLICT (tenant_id) DO UPDATE SET
  encrypted_seed = EXCLUDED.encrypted_seed,
  nonce          = EXCLUDED.nonce,
  key_version    = tenant_keys.key_version + 1
`

func (r *Repository) SaveEncryptedSeed(ctx context.Context, id uuid.UUID, enc, nonce []byte) error {
	_, err := r.pool.Exec(ctx, insertSeedSQL, id, enc, nonce)
	return err
}

const selectSeedSQL = `SELECT encrypted_seed, nonce FROM tenant_keys WHERE tenant_id = $1`

func (r *Repository) LoadEncryptedSeed(ctx context.Context, id uuid.UUID) (enc, nonce []byte, err error) {
	err = r.pool.QueryRow(ctx, selectSeedSQL, id).Scan(&enc, &nonce)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	return
}

// Pool exposes the underlying pool for direct SQL when service methods need it.
func (r *Repository) Pool() *pgxpool.Pool { return r.pool }

var _ = time.Now
