package pool

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Connection 表示一个连接
type Connection struct {
	conn     net.Conn
	lastUsed time.Time
	inUse    bool
}

// ConnectionPool 连接池
type ConnectionPool struct {
	mu          sync.Mutex
	connections []*Connection
	maxSize     int
	minSize     int
	address     string
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(address string, minSize, maxSize int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make([]*Connection, 0, maxSize),
		maxSize:     maxSize,
		minSize:     minSize,
		address:     address,
	}
	
	// 初始化最小连接数
	for i := 0; i < minSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			fmt.Printf("Failed to create initial connection: %v\n", err)
			continue
		}
		pool.connections = append(pool.connections, &Connection{
			conn:     conn,
			lastUsed: time.Now(),
			inUse:    false,
		})
	}
	
	return pool
}

// Get 获取一个连接
func (p *ConnectionPool) Get() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. 尝试获取空闲连接
	for _, conn := range p.connections {
		if !conn.inUse {
			conn.inUse = true
			conn.lastUsed = time.Now()
			return conn.conn, nil
		}
	}

	// 2. 如果没有空闲连接且未达到最大连接数，创建新连接
	if len(p.connections) < p.maxSize {
		conn, err := p.createConnection()
		if err != nil {
			return nil, err
		}
		newConn := &Connection{
			conn:     conn,
			lastUsed: time.Now(),
			inUse:    true,
		}
		p.connections = append(p.connections, newConn)
		return conn, nil
	}

	// 3. 如果达到最大连接数，等待并重试
	return nil, fmt.Errorf("connection pool is full")
}

// Put 归还连接
func (p *ConnectionPool) Put(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, c := range p.connections {
		if c.conn == conn {
			c.inUse = false
			c.lastUsed = time.Now()
			return
		}
	}
	// 如果找不到连接，说明是外部创建的，直接关闭
	conn.Close()
}

// createConnection 创建新连接
func (p *ConnectionPool) createConnection() (net.Conn, error) {
	return net.Dial("tcp", p.address)
}

// Close 关闭连接池
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.conn.Close()
	}
	p.connections = nil
} 