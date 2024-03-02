// utils/shared.go
package utils

import "sync"

type ConnectedUsers struct {
	mu    sync.Mutex
	Users map[string]bool
}

func NewConnectedUsers() *ConnectedUsers {
	return &ConnectedUsers{
		Users: make(map[string]bool),
	}
}

func (cu *ConnectedUsers) Set(name string, value bool) {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	cu.Users[name] = value
}

func (cu *ConnectedUsers) Get(name string) bool {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	return cu.Users[name]
}
