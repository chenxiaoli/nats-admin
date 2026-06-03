package credential

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("credential not found")

type Credential struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Name          string
	UserPublicKey string
	UserJWT       string
	EncryptedSeed []byte
	Nonce         []byte
	PubAllow      []string
	SubAllow      []string
	RevokedAt     *time.Time
	ExpiresAt     *time.Time
	CreatedAt     time.Time
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

func (r *PgRepository) Pool() *pgxpool.Pool { return r.pool }

const insertCredSQL = `
INSERT INTO user_credentials (
  tenant_id, name, user_public_key, user_jwt, encrypted_seed, nonce,
  pub_allow, sub_allow, expires_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING id, created_at
`

func (r *PgRepository) InsertCredential(ctx context.Context, c *Credential) error {
	return r.pool.QueryRow(ctx, insertCredSQL,
		c.TenantID, c.Name, c.UserPublicKey, c.UserJWT, c.EncryptedSeed, c.Nonce,
		c.PubAllow, c.SubAllow, c.ExpiresAt,
	).Scan(&c.ID, &c.CreatedAt)
}

const selectCredSQL = `
SELECT id, tenant_id, name, user_public_key, user_jwt, encrypted_seed, nonce,
       pub_allow, sub_allow, revoked_at, expires_at, created_at
FROM user_credentials WHERE id = $1
`

func (r *PgRepository) GetCredential(ctx context.Context, id uuid.UUID) (*Credential, error) {
	c := &Credential{}
	err := r.pool.QueryRow(ctx, selectCredSQL, id).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.UserPublicKey, &c.UserJWT, &c.EncryptedSeed, &c.Nonce,
		&c.PubAllow, &c.SubAllow, &c.RevokedAt, &c.ExpiresAt, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

const listCredSQL = `
SELECT id, tenant_id, name, user_public_key, user_jwt, encrypted_seed, nonce,
       pub_allow, sub_allow, revoked_at, expires_at, created_at
FROM user_credentials WHERE tenant_id = $1 ORDER BY created_at DESC
`

func (r *PgRepository) ListCredentials(ctx context.Context, tenantID uuid.UUID) ([]*Credential, error) {
	rows, err := r.pool.Query(ctx, listCredSQL, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Credential
	for rows.Next() {
		c := &Credential{}
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.UserPublicKey, &c.UserJWT, &c.EncryptedSeed, &c.Nonce,
			&c.PubAllow, &c.SubAllow, &c.RevokedAt, &c.ExpiresAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

const revokeCredSQL = `UPDATE user_credentials SET revoked_at = $2 WHERE id = $1`

func (r *PgRepository) RevokeCredential(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx, revokeCredSQL, id, at)
	return err
}
