package manager

import (
	"errors"
	"sync"

	"github.com/andrewmthomas87/von-neumann-web-server/game"
	"github.com/pion/webrtc/v3"
)

type Manager interface {
	List() []game.Server
	Register(server game.Server)
	Unregister(server game.Server)
	Connect(id string, sd *webrtc.SessionDescription) (*webrtc.SessionDescription, error)
}

type manager struct {
	servers map[string]game.Server
	lock    sync.RWMutex
}

func (m *manager) List() []game.Server {
	m.lock.RLock()
	defer m.lock.RUnlock()

	servers := make([]game.Server, len(m.servers))
	i := 0
	for _, s := range m.servers {
		servers[i] = s
		i++
	}
	return servers
}

func (m *manager) Register(server game.Server) {
	m.lock.Lock()
	m.servers[server.ID()] = server
	m.lock.Unlock()
}

func (m *manager) Unregister(server game.Server) {
	m.lock.Lock()
	delete(m.servers, server.ID())
	m.lock.Unlock()
}

var ErrInvalidServer = errors.New("invalid server")

func (m *manager) Connect(id string, sd *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	m.lock.RLock()
	s, ok := m.servers[id]
	m.lock.RUnlock()
	if !ok {
		return nil, ErrInvalidServer
	}

	return s.Connect(sd)
}

func New() Manager {
	return &manager{servers: make(map[string]game.Server)}
}
