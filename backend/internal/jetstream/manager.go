package jetstream

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type Manager struct {
	mu     sync.RWMutex
	conns  map[uuid.UUID]*nats.Conn
	url    string
	onClose func(uuid.UUID)
}

func NewManager(url string) *Manager {
	return &Manager{conns: make(map[uuid.UUID]*nats.Conn), url: url}
}

func (m *Manager) SetOnClose(fn func(uuid.UUID)) {
	m.onClose = fn
}

func (m *Manager) GetConn(tenantID uuid.UUID, jwtStr, seed string) (*nats.Conn, error) {
	m.mu.RLock()
	conn, ok := m.conns[tenantID]
	m.mu.RUnlock()
	if ok && conn.IsConnected() {
		return conn, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok = m.conns[tenantID]
	if ok && conn.IsConnected() {
		return conn, nil
	}

	nc, err := nats.Connect(m.url,
		nats.UserJWTAndSeed(jwtStr, seed),
		nats.ClosedHandler(func(c *nats.Conn) {
			m.mu.Lock()
			delete(m.conns, tenantID)
			m.mu.Unlock()
			if m.onClose != nil {
				m.onClose(tenantID)
			}
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	m.conns[tenantID] = nc
	return nc, nil
}

func (m *Manager) GetJS(tenantID uuid.UUID, jwtStr, seed string) (nats.JetStreamContext, error) {
	conn, err := m.GetConn(tenantID, jwtStr, seed)
	if err != nil {
		return nil, err
	}
	return conn.JetStream()
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, conn := range m.conns {
		conn.Close()
		delete(m.conns, id)
	}
}
