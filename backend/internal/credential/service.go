package credential

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chenxiaoli/nats-admin/internal/crypto"
	"github.com/chenxiaoli/nats-admin/internal/operator"
	"github.com/chenxiaoli/nats-admin/internal/tenant"
	"github.com/google/uuid"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

var ErrTenantGone = errors.New("tenant not found")

type IssueRequest struct {
	Name      string
	PubAllow  []string
	SubAllow  []string
	ExpiresAt *time.Time
}

type Issued struct {
	ID        uuid.UUID
	Name      string
	PublicKey string
	Creds     string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type Service struct {
	repo    *PgRepository
	tenants *tenant.Service
	op      *operator.Operator
	master  []byte
}

func NewService(repo *PgRepository, tenants *tenant.Service, op *operator.Operator, master []byte) *Service {
	return &Service{repo: repo, tenants: tenants, op: op, master: master}
}

func (s *Service) Issue(ctx context.Context, tenantID uuid.UUID, req IssueRequest) (*Issued, error) {
	_, err := s.tenants.Get(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantGone
	}

	aseedStr, err := s.tenants.AccountSeedForSigning(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load account seed: %w", err)
	}
	defer zeroStr(aseedStr)
	akp, err := nkeys.FromSeed([]byte(aseedStr))
	if err != nil {
		return nil, fmt.Errorf("invalid account seed: %w", err)
	}
	apub, _ := akp.PublicKey()

	ukp, err := nkeys.CreateUser()
	if err != nil {
		return nil, err
	}
	upub, _ := ukp.PublicKey()
	useed, _ := ukp.Seed()
	defer zeroBytes(useed)

	claims := jwt.NewUserClaims(upub)
	claims.Name = req.Name
	claims.IssuerAccount = apub
	claims.IssuedAt = time.Now().UTC().Unix()
	if req.ExpiresAt != nil {
		claims.Expires = req.ExpiresAt.Unix()
	}
	if len(req.PubAllow) > 0 {
		claims.Permissions.Pub.Allow = req.PubAllow
	}
	if len(req.SubAllow) > 0 {
		claims.Permissions.Sub.Allow = req.SubAllow
	}
	userJWT, err := claims.Encode(akp)
	if err != nil {
		return nil, fmt.Errorf("sign user jwt: %w", err)
	}

	uenc, unonce, err := crypto.Encrypt(s.master, useed)
	if err != nil {
		return nil, err
	}

	cred := &Credential{
		TenantID:      tenantID,
		Name:          req.Name,
		UserPublicKey: upub,
		UserJWT:       userJWT,
		EncryptedSeed: uenc,
		Nonce:         unonce,
		PubAllow:      req.PubAllow,
		SubAllow:      req.SubAllow,
		ExpiresAt:     req.ExpiresAt,
	}
	if err := s.repo.InsertCredential(ctx, cred); err != nil {
		return nil, err
	}

	credsBody := fmt.Sprintf(
		"-----BEGIN NATS USER JWT-----\n%s\n------END NATS USER JWT------\n\n"+
			"-----BEGIN USER NKEY SEED-----\n%s\n------END USER NKEY SEED------\n",
		userJWT, string(useed),
	)
	return &Issued{
		ID:        cred.ID,
		Name:      cred.Name,
		PublicKey: upub,
		Creds:     credsBody,
		ExpiresAt: cred.ExpiresAt,
		CreatedAt: cred.CreatedAt,
	}, nil
}

func (s *Service) Revoke(ctx context.Context, tenantID, credentialID uuid.UUID) error {
	cred, err := s.repo.GetCredential(ctx, credentialID)
	if err != nil {
		return ErrNotFound
	}
	if cred.TenantID != tenantID {
		return ErrNotFound
	}
	if cred.RevokedAt != nil {
		return nil
	}

	_, err = s.tenants.Get(ctx, tenantID)
	if err != nil {
		return ErrTenantGone
	}

	// Get the current account JWT from the tenants table
	acctJWT := ""
	row := s.repo.Pool().QueryRow(ctx, `SELECT account_jwt FROM tenants WHERE id = $1`, tenantID)
	if err := row.Scan(&acctJWT); err != nil {
		return fmt.Errorf("get account jwt: %w", err)
	}

	claims, err := jwt.DecodeAccountClaims(acctJWT)
	if err != nil {
		return fmt.Errorf("decode account jwt: %w", err)
	}
	if claims.Revocations == nil {
		claims.Revocations = jwt.RevocationList{}
	}
	claims.Revocations.Revoke(cred.UserPublicKey, time.Now())
	newJWT, err := s.op.SignAccountClaims(claims)
	if err != nil {
		return fmt.Errorf("re-sign account jwt: %w", err)
	}

	if err := s.tenants.UpdateAccountJWT(ctx, tenantID, newJWT); err != nil {
		return err
	}
	if err := s.repo.RevokeCredential(ctx, credentialID, time.Now().UTC()); err != nil {
		return err
	}
	if err := s.tenants.PushAccountJWT(ctx, tenantID, newJWT); err != nil {
		return fmt.Errorf("push (db ok, resolver stale): %w", err)
	}
	return nil
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]*Credential, error) {
	return s.repo.ListCredentials(ctx, tenantID)
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func zeroStr(s string) {
	zeroBytes([]byte(s))
}
