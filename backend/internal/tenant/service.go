package tenant

import (
	"context"
	"fmt"
	"regexp"

	"github.com/chenxiaoli/nats-admin/internal/crypto"
	"github.com/chenxiaoli/nats-admin/internal/operator"
	"github.com/google/uuid"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,48}[a-z0-9])?$`)

type CreateRequest struct {
	Name             string
	Slug             string
	JSMemoryStorage  int64
	JSDiskStorage    int64
	JSMaxStreams     int32
	JSMaxConsumers   int32
	MaxConnections   int32
	MaxSubscriptions int32
}

type Service struct {
	repo     *Repository
	resolver *Resolver
	op       *operator.Operator
	master   []byte
}

func NewService(repo *Repository, res *Resolver, op *operator.Operator, master []byte) *Service {
	return &Service{repo: repo, resolver: res, op: op, master: master}
}

type Created struct {
	Tenant *Tenant
	Seed   string
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Created, error) {
	if !slugRe.MatchString(req.Slug) {
		return nil, fmt.Errorf("invalid slug: %q", req.Slug)
	}

	akp, err := nkeys.CreateAccount()
	if err != nil {
		return nil, fmt.Errorf("create account nkey: %w", err)
	}
	apub, _ := akp.PublicKey()
	aseed, _ := akp.Seed()

	claims := operator.NewAccountClaims(apub, req.Name)
	claims.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: req.JSMemoryStorage,
		DiskStorage:   req.JSDiskStorage,
		Streams:       int64(req.JSMaxStreams),
		Consumer:      int64(req.JSMaxConsumers),
	}
	if req.MaxConnections > 0 {
		claims.Limits.Conn = int64(req.MaxConnections)
	}
	if req.MaxSubscriptions > 0 {
		claims.Limits.Subs = int64(req.MaxSubscriptions)
	}
	operatorJWT, err := s.op.SignAccountClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("sign account jwt: %w", err)
	}

	t := &Tenant{
		Name:             req.Name,
		Slug:             req.Slug,
		AccountPublicKey: apub,
		AccountJWT:       operatorJWT,
		JSMemoryStorage:  req.JSMemoryStorage,
		JSDiskStorage:    req.JSDiskStorage,
		JSMaxStreams:     req.JSMaxStreams,
		JSMaxConsumers:   req.JSMaxConsumers,
		MaxConnections:   req.MaxConnections,
		MaxSubscriptions: req.MaxSubscriptions,
		Status:           "active",
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}

	enc, nonce, err := crypto.Encrypt(s.master, aseed)
	if err != nil {
		_ = s.repo.SoftDelete(ctx, t.ID)
		return nil, fmt.Errorf("encrypt account seed: %w", err)
	}
	if err := s.repo.SaveEncryptedSeed(ctx, t.ID, enc, nonce); err != nil {
		_ = s.repo.SoftDelete(ctx, t.ID)
		return nil, fmt.Errorf("save encrypted seed: %w", err)
	}

	if s.resolver != nil {
		if err := s.resolver.Push(ctx, operatorJWT); err != nil {
			_ = s.repo.SoftDelete(ctx, t.ID)
			return nil, fmt.Errorf("push account jwt: %w", err)
		}
	}

	return &Created{Tenant: t, Seed: string(aseed)}, nil
}

func (s *Service) UpdateLimits(ctx context.Context, id uuid.UUID, req CreateRequest) (*Tenant, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	claims, err := jwt.DecodeAccountClaims(t.AccountJWT)
	if err != nil {
		return nil, fmt.Errorf("decode current account jwt: %w", err)
	}
	claims.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: req.JSMemoryStorage,
		DiskStorage:   req.JSDiskStorage,
		Streams:       int64(req.JSMaxStreams),
		Consumer:      int64(req.JSMaxConsumers),
	}
	claims.Limits.Conn = int64(req.MaxConnections)
	claims.Limits.Subs = int64(req.MaxSubscriptions)

	newJWT, err := s.op.SignAccountClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("re-sign: %w", err)
	}
	t.AccountJWT = newJWT
	t.JSMemoryStorage = req.JSMemoryStorage
	t.JSDiskStorage = req.JSDiskStorage
	t.JSMaxStreams = req.JSMaxStreams
	t.JSMaxConsumers = req.JSMaxConsumers
	t.MaxConnections = req.MaxConnections
	t.MaxSubscriptions = req.MaxSubscriptions

	if err := s.repo.UpdateJWTAndLimits(ctx, t); err != nil {
		return nil, err
	}
	if s.resolver != nil {
		if err := s.resolver.Push(ctx, newJWT); err != nil {
			return nil, fmt.Errorf("push updated jwt (db now stale, manual reconcile required): %w", err)
		}
	}
	return t, nil
}

func (s *Service) Suspend(ctx context.Context, id uuid.UUID) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.resolver != nil {
		if err := s.resolver.Delete(ctx, t.AccountPublicKey); err != nil {
			return err
		}
	}
	return s.repo.SetStatus(ctx, id, "suspended")
}

func (s *Service) Activate(ctx context.Context, id uuid.UUID) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.resolver != nil {
		if err := s.resolver.Push(ctx, t.AccountJWT); err != nil {
			return err
		}
	}
	return s.repo.SetStatus(ctx, id, "active")
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.resolver != nil {
		_ = s.resolver.Delete(ctx, t.AccountPublicKey)
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]*Tenant, error) {
	return s.repo.List(ctx)
}

func (s *Service) AccountSeedForSigning(ctx context.Context, id uuid.UUID) (string, error) {
	enc, nonce, err := s.repo.LoadEncryptedSeed(ctx, id)
	if err != nil {
		return "", err
	}
	pt, err := crypto.Decrypt(s.master, enc, nonce)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// UpdateAccountJWT overwrites the tenants.account_jwt column.
func (s *Service) UpdateAccountJWT(ctx context.Context, id uuid.UUID, jwtStr string) error {
	_, err := s.repo.Pool().Exec(ctx, `UPDATE tenants SET account_jwt=$2, updated_at=NOW() WHERE id=$1`, id, jwtStr)
	return err
}

// PushAccountJWT pushes the given JWT string to the NATS resolver.
func (s *Service) PushAccountJWT(ctx context.Context, _ uuid.UUID, jwtStr string) error {
	if s.resolver == nil {
		return ErrPushUnreachable
	}
	return s.resolver.Push(ctx, jwtStr)
}
