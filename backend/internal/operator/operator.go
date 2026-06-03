package operator

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

type Operator struct {
	mu  sync.RWMutex
	kp  nkeys.KeyPair
	pub string
}

var ErrNoOperatorSeed = errors.New("OPERATOR_SEED is not set")

func LoadFromEnv() (*Operator, error) {
	seed := os.Getenv("OPERATOR_SEED")
	if seed == "" {
		return nil, ErrNoOperatorSeed
	}
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, fmt.Errorf("invalid OPERATOR_SEED: %w", err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return nil, err
	}
	return &Operator{kp: kp, pub: pub}, nil
}

func (o *Operator) PublicKey() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.pub
}

func (o *Operator) SignAccountClaims(claims *jwt.AccountClaims) (string, error) {
	o.mu.RLock()
	kp := o.kp
	o.mu.RUnlock()
	return claims.Encode(kp)
}

func NewAccountClaims(pubKey, name string) *jwt.AccountClaims {
	now := time.Now().UTC()
	c := jwt.NewAccountClaims(pubKey)
	c.Name = name
	c.IssuedAt = now.Unix()
	c.Expires = now.Add(365 * 24 * time.Hour).Unix()
	return c
}
