package jetstream

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type StreamInfo struct {
	Name     string   `json:"name"`
	Subjects []string `json:"subjects"`
	Messages uint64   `json:"messages"`
	Bytes    uint64   `json:"bytes"`
}

type CreateStreamReq struct {
	Name     string   `json:"name"`
	Subjects []string `json:"subjects"`
	MaxBytes int64    `json:"max_bytes"`
	MaxMsgs  int64    `json:"max_msgs"`
	Replicas int      `json:"replicas"`
}

type KVBucketInfo struct {
	Bucket  string `json:"bucket"`
	Values  uint64 `json:"values"`
	History int64  `json:"history"`
}

type CreateKVReq struct {
	Bucket   string `json:"bucket"`
	History  int    `json:"history"`
	MaxBytes int64  `json:"max_bytes"`
}

type Admin struct {
	mgr       *Manager
	getTenant func(ctx context.Context, id uuid.UUID) (jwtStr, seed string, err error)
}

func NewAdmin(mgr *Manager, getTenant func(ctx context.Context, id uuid.UUID) (string, string, error)) *Admin {
	return &Admin{mgr: mgr, getTenant: getTenant}
}

func (a *Admin) js(ctx context.Context, tenantID uuid.UUID) (nats.JetStreamContext, error) {
	jwtStr, seed, err := a.getTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant creds: %w", err)
	}
	return a.mgr.GetJS(tenantID, jwtStr, seed)
}

func (a *Admin) ListStreams(ctx context.Context, tenantID uuid.UUID) ([]StreamInfo, error) {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	ch := js.Streams()
	var out []StreamInfo
	for si := range ch {
		if si == nil {
			break
		}
		out = append(out, StreamInfo{
			Name:     si.Config.Name,
			Subjects: si.Config.Subjects,
			Messages: si.State.Msgs,
			Bytes:    si.State.Bytes,
		})
	}
	return out, nil
}

func (a *Admin) CreateStream(ctx context.Context, tenantID uuid.UUID, req CreateStreamReq) (*StreamInfo, error) {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	cfg := &nats.StreamConfig{
		Name:     req.Name,
		Subjects: req.Subjects,
		MaxBytes: req.MaxBytes,
		MaxMsgs:  req.MaxMsgs,
		Replicas: req.Replicas,
	}
	if cfg.Replicas == 0 {
		cfg.Replicas = 1
	}
	si, err := js.AddStream(cfg)
	if err != nil {
		return nil, fmt.Errorf("add stream: %w", err)
	}
	return &StreamInfo{
		Name:     si.Config.Name,
		Subjects: si.Config.Subjects,
		Messages: si.State.Msgs,
		Bytes:    si.State.Bytes,
	}, nil
}

func (a *Admin) DeleteStream(ctx context.Context, tenantID uuid.UUID, name string) error {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return err
	}
	return js.DeleteStream(name)
}

func (a *Admin) PurgeStream(ctx context.Context, tenantID uuid.UUID, name string) error {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return err
	}
	return js.PurgeStream(name)
}

func (a *Admin) ListKV(ctx context.Context, tenantID uuid.UUID) ([]KVBucketInfo, error) {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	ch := js.KeyValueStores()
	var out []KVBucketInfo
	for ks := range ch {
		if ks == nil {
			break
		}
		out = append(out, KVBucketInfo{
			Bucket:  ks.Bucket(),
			Values:  ks.Values(),
			History: ks.History(),
		})
	}
	return out, nil
}

func (a *Admin) CreateKV(ctx context.Context, tenantID uuid.UUID, req CreateKVReq) (*KVBucketInfo, error) {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	cfg := &nats.KeyValueConfig{
		Bucket:   req.Bucket,
		History:  uint8(req.History),
		MaxBytes: req.MaxBytes,
	}
	kv, err := js.CreateKeyValue(cfg)
	if err != nil {
		return nil, fmt.Errorf("create kv: %w", err)
	}
	ks, err := kv.Status()
	if err != nil {
		return nil, err
	}
	return &KVBucketInfo{
		Bucket:  ks.Bucket(),
		Values:  ks.Values(),
		History: ks.History(),
	}, nil
}

func (a *Admin) DeleteKV(ctx context.Context, tenantID uuid.UUID, bucket string) error {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return err
	}
	return js.DeleteKeyValue(bucket)
}
