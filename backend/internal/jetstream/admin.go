package jetstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type StreamInfo struct {
	Name     string   `json:"name"`
	Subjects []string `json:"subjects"`
	Messages uint64   `json:"messages"`
	Bytes    uint64   `json:"bytes"`
}

type ConsumerInfo struct {
	Name           string `json:"name"`
	Stream         string `json:"stream"`
	NumPending     uint64 `json:"num_pending"`
	NumAckPending  int    `json:"num_ack_pending"`
	NumRedelivered int    `json:"num_redelivered"`
	DeliveredStreamSeq  uint64 `json:"delivered_stream_seq"`
	AckFloorStreamSeq   uint64 `json:"ack_floor_stream_seq"`
	Created string `json:"created"`
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
	reseed    func(ctx context.Context, id uuid.UUID) error
}

func NewAdmin(mgr *Manager, getTenant func(ctx context.Context, id uuid.UUID) (string, string, error), reseed func(ctx context.Context, id uuid.UUID) error) *Admin {
	return &Admin{mgr: mgr, getTenant: getTenant, reseed: reseed}
}

func (a *Admin) js(ctx context.Context, tenantID uuid.UUID) (nats.JetStreamContext, error) {
	jwtStr, seed, err := a.getTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant creds: %w", err)
	}
	js, err := a.mgr.GetJS(tenantID, jwtStr, seed)
	if err != nil {
		if !errors.Is(err, nats.ErrAuthorization) || a.reseed == nil {
			return nil, err
		}
		if reseedErr := a.reseed(ctx, tenantID); reseedErr != nil {
			return nil, fmt.Errorf("auth violation (reseed failed: %v): %w", reseedErr, err)
		}
		return a.mgr.GetJS(tenantID, jwtStr, seed)
	}
	return js, nil
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

func (a *Admin) ListConsumers(ctx context.Context, tenantID uuid.UUID, stream string) ([]ConsumerInfo, error) {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	ch := js.ConsumersInfo(stream)
	var out []ConsumerInfo
	for ci := range ch {
		if ci == nil {
			break
		}
		out = append(out, ConsumerInfo{
			Name:                ci.Name,
			Stream:              ci.Stream,
			NumPending:          ci.NumPending,
			NumAckPending:       ci.NumAckPending,
			NumRedelivered:      ci.NumRedelivered,
			DeliveredStreamSeq:  ci.Delivered.Stream,
			AckFloorStreamSeq:   ci.AckFloor.Stream,
			Created:             ci.Created.Format(time.RFC3339),
		})
	}
	return out, nil
}

func (a *Admin) GetConsumer(ctx context.Context, tenantID uuid.UUID, stream, consumer string) (*ConsumerInfo, error) {
	js, err := a.js(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	ci, err := js.ConsumerInfo(stream, consumer)
	if err != nil {
		return nil, fmt.Errorf("consumer info: %w", err)
	}
	return &ConsumerInfo{
		Name:                ci.Name,
		Stream:              ci.Stream,
		NumPending:          ci.NumPending,
		NumAckPending:       ci.NumAckPending,
		NumRedelivered:      ci.NumRedelivered,
		DeliveredStreamSeq:  ci.Delivered.Stream,
		AckFloorStreamSeq:   ci.AckFloor.Stream,
		Created:             ci.Created.Format(time.RFC3339),
	}, nil
}
