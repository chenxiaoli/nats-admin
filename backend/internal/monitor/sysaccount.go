package monitor

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type ServerStats struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Connections int    `json:"connections"`
	Subscriptions int  `json:"subscriptions"`
	InMsgs      int64  `json:"in_msgs"`
	OutMsgs     int64  `json:"out_msgs"`
	InBytes     int64  `json:"in_bytes"`
	OutBytes    int64  `json:"out_bytes"`
	Uptime      string `json:"uptime"`
}

type AccountStats struct {
	AccountID   string `json:"account_id"`
	Connections int    `json:"connections"`
	TotalConns  int    `json:"total_conns"`
	InMsgs      int64  `json:"in_msgs"`
	OutMsgs     int64  `json:"out_msgs"`
	InBytes     int64  `json:"in_bytes"`
	OutBytes    int64  `json:"out_bytes"`
}

type Monitor struct {
	mu       sync.RWMutex
	servers  []ServerStats
	accounts map[string]AccountStats
	conn     *nats.Conn
	subs     []*nats.Subscription
}

func NewMonitor(sysConn *nats.Conn) *Monitor {
	return &Monitor{conn: sysConn, accounts: make(map[string]AccountStats)}
}

func (m *Monitor) Start(ctx context.Context) error {
	if m.conn == nil {
		return nil
	}

	// Subscribe to server stats. NATS 2.10 publishes these to
	// $SYS.SERVER.<sid>.STATSZ (note the trailing Z) on a server interval
	// (default 1m). Earlier doc that called the suffix "STATZ" was wrong.
	sub1, err := m.conn.Subscribe("$SYS.SERVER.*.STATSZ", func(msg *nats.Msg) {
		var raw struct {
			Server struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"server"`
			Statsz struct {
				Connections   int   `json:"connections"`
				Subscriptions int   `json:"subscriptions"`
				Received      struct {
					Msgs  int64 `json:"msgs"`
					Bytes int64 `json:"bytes"`
				} `json:"received"`
				Sent struct {
					Msgs  int64 `json:"msgs"`
					Bytes int64 `json:"bytes"`
				} `json:"sent"`
				Start time.Time `json:"start"`
			} `json:"statsz"`
		}
		if err := json.Unmarshal(msg.Data, &raw); err != nil {
			return
		}
		s := ServerStats{
			Name:         raw.Server.Name,
			ID:           raw.Server.ID,
			Connections:  raw.Statsz.Connections,
			Subscriptions: raw.Statsz.Subscriptions,
			InMsgs:       raw.Statsz.Received.Msgs,
			OutMsgs:      raw.Statsz.Sent.Msgs,
			InBytes:      raw.Statsz.Received.Bytes,
			OutBytes:     raw.Statsz.Sent.Bytes,
			Uptime:       time.Since(raw.Statsz.Start).Truncate(time.Second).String(),
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		found := false
		for i, sv := range m.servers {
			if sv.ID == s.ID {
				m.servers[i] = s
				found = true
				break
			}
		}
		if !found {
			m.servers = append(m.servers, s)
		}
	})
	if err != nil {
		return err
	}

	// Subscribe to per-account connection events. NATS publishes these to
	// $SYS.SERVER.ACCOUNT.<pubkey>.CONNS (not "STATZ") at the account-stat
	// interval (default ~10s). Each payload is a single account's snapshot.
	sub2, err := m.conn.Subscribe("$SYS.SERVER.ACCOUNT.*.CONNS", func(msg *nats.Msg) {
		var raw struct {
			Acc        string `json:"acc"`
			Conns      int    `json:"conns"`
			TotalConns int    `json:"total_conns"`
		}
		if err := json.Unmarshal(msg.Data, &raw); err != nil {
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.accounts[raw.Acc] = AccountStats{
			AccountID:   raw.Acc,
			Connections: raw.Conns,
			TotalConns:  raw.TotalConns,
		}
	})
	if err != nil {
		sub1.Unsubscribe()
		return err
	}

	m.subs = []*nats.Subscription{sub1, sub2}

	// Request initial server ping
	go func() {
		m.conn.Publish("$SYS.REQ.SERVER.PING", nil)
	}()

	return nil
}

func (m *Monitor) Stop() {
	for _, s := range m.subs {
		s.Unsubscribe()
	}
}

func (m *Monitor) Servers() []ServerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ServerStats, len(m.servers))
	copy(out, m.servers)
	return out
}

func (m *Monitor) Accounts() []AccountStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]AccountStats, 0, len(m.accounts))
	for _, a := range m.accounts {
		out = append(out, a)
	}
	return out
}

func (m *Monitor) Account(pubKey string) *AccountStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.accounts[pubKey]
	if !ok {
		return nil
	}
	return &a
}

func (m *Monitor) RequestServerStats(ctx context.Context) ([]ServerStats, error) {
	if m.conn == nil {
		return nil, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := m.conn.RequestWithContext(cctx, "$SYS.REQ.SERVER.PING", nil)
	if err != nil {
		return m.Servers(), nil
	}
	_ = resp
	return m.Servers(), nil
}

func (m *Monitor) RequestAccountStats(ctx context.Context, pubKey string) (*AccountStats, error) {
	if m.conn == nil {
		return nil, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	subj := "$SYS.REQ.ACCOUNT." + pubKey + ".STATZ"
	resp, err := m.conn.RequestWithContext(cctx, subj, nil)
	if err != nil {
		return m.Account(pubKey), nil
	}
	var stats AccountStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		return m.Account(pubKey), nil
	}
	m.mu.Lock()
	m.accounts[pubKey] = stats
	m.mu.Unlock()
	return &stats, nil
}
