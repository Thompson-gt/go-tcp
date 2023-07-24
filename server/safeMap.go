package main

import (
	"errors"
	"net"
	"sync"
)

type ServerMapper interface {
	Get(key string) (net.Conn, error)
	Add(key string, val net.Conn)
	Remove(key string)
	AllConns() map[string]net.Conn
}

type SafeMap struct {
	Conns map[string]net.Conn
	lock  sync.RWMutex
}

func NewSafeMap() SafeMap {
	m := make(map[string]net.Conn)
	return SafeMap{Conns: m}
}

// get a value from the safe map
func (sm *SafeMap) Get(key string) (net.Conn, error) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	if result, ok := sm.Conns[key]; ok {
		return result, nil
	}
	return nil, errors.New("connection not found in SafeMap")
}

// add a key value pair to the safe map
func (sm *SafeMap) Add(key string, val net.Conn) {
	sm.lock.Lock()
	sm.Conns[key] = val
	sm.lock.Unlock()
}

// return the whole map at once
func (sm *SafeMap) AllConns() map[string]net.Conn {
	return sm.Conns
}

// remove the given key from the safe map
func (sm *SafeMap) Remove(key string) {
	sm.lock.Lock()
	delete(sm.Conns, key)
	sm.lock.Unlock()
}
