package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	ErrPushRejected   = errors.New("resolver rejected claim push")
	ErrPushUnreachable = errors.New("resolver unreachable")
)

type Resolver struct {
	sysConn *nats.Conn
	timeout time.Duration
}

func NewResolver(sysConn *nats.Conn) *Resolver {
	return &Resolver{sysConn: sysConn, timeout: 5 * time.Second}
}

func (r *Resolver) Push(ctx context.Context, accountJWT string) error {
	if r.sysConn == nil {
		return ErrPushUnreachable
	}
	cctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	resp, err := r.sysConn.RequestWithContext(cctx, "$SYS.REQ.CLAIMS.UPDATE", []byte(accountJWT))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPushUnreachable, err)
	}
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return fmt.Errorf("decode resolver reply: %w", err)
	}
	if out.Error != "" {
		return fmt.Errorf("%w: %s", ErrPushRejected, out.Error)
	}
	return nil
}

func (r *Resolver) Delete(ctx context.Context, accountPubKey string) error {
	if r.sysConn == nil {
		return ErrPushUnreachable
	}
	cctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	resp, err := r.sysConn.RequestWithContext(cctx, "$SYS.REQ.CLAIMS.DELETE", []byte(accountPubKey))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPushUnreachable, err)
	}
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return fmt.Errorf("decode resolver reply: %w", err)
	}
	if out.Error != "" {
		return fmt.Errorf("%w: %s", ErrPushRejected, out.Error)
	}
	return nil
}
