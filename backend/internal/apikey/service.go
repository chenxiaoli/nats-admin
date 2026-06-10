package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

var (
	ErrNameConflict = errors.New("api key name already exists")
	ErrNotActive    = errors.New("api key not active")
)

const (
	prefix    = "nak_live_"
	randomLen = 32
)

// base62 alphabet
const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type Service struct {
	repo *PgRepository
}

func NewService(repo *PgRepository) *Service { return &Service{repo: repo} }

type Created struct {
	Key *Key
	Raw string
}

func (s *Service) Create(ctx context.Context, adminID uuid.UUID, name string) (*Created, error) {
	raw, err := generate()
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	hash := HashKey(raw)
	k := &Key{
		AdminID:   adminID,
		Name:      name,
		KeyPrefix: raw[:8],
		KeyHash:   hash,
	}
	if err := s.repo.Insert(ctx, k); err != nil {
		// Treat unique violation as conflict. Caller checks pg err code in tests if needed.
		return nil, err
	}
	return &Created{Key: k, Raw: raw}, nil
}

func (s *Service) List(ctx context.Context, adminID uuid.UUID) ([]*Key, error) {
	return s.repo.ListByAdmin(ctx, adminID)
}

func (s *Service) Revoke(ctx context.Context, adminID, keyID uuid.UUID) error {
	if err := s.repo.Get(ctx, keyID, adminID); err != nil {
		return err
	}
	ok, err := s.repo.Revoke(ctx, keyID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotActive
	}
	return nil
}

// Validate looks up a raw key by hash. Returns the api key id and admin id.
func (s *Service) Validate(ctx context.Context, raw string) (keyID, adminID uuid.UUID, err error) {
	hash := HashKey(raw)
	l, err := s.repo.LookupByHash(ctx, hash)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	s.repo.TouchLastUsed(ctx, l.KeyID)
	return l.KeyID, l.AdminID, nil
}

func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func generate() (string, error) {
	out := make([]byte, randomLen)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62))))
		if err != nil {
			return "", err
		}
		out[i] = base62[n.Int64()]
	}
	return prefix + string(out), nil
}
