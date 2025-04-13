package pool

import (
	"net"
	"sync"
	"time"
)

type PoolManager struct {
	mu    sync.RWMutex
	pools map[string]*ConnPool
}

func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]*ConnPool),
	}
}

func (pm *PoolManager) GetPool(addr string) *ConnPool {
	pm.mu.RLock()
	if pool, exists := pm.pools[addr]; exists {
		pm.mu.RUnlock()
		return pool
	}
	pm.mu.RUnlock()

	// 创建连接池
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pool, exists := pm.pools[addr]; exists {
		return pool
	}

	pool := NewConnPool(addr, 10, 2, 5, 60*time.Second, 5*time.Minute,
		func(address string) (net.Conn, error) {
			
			return net.DialTimeout("tcp", address, 2*time.Second)
		})
	pm.pools[addr] = pool
	return pool
}
