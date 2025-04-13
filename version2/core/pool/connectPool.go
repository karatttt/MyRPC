package pool

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

)


var (
	ErrPoolClosed    = errors.New("connection pool is closed")
	ErrPoolExhausted = errors.New("connection pool exhausted")
	ErrConnTimeout   = errors.New("connection timeout")
)

type PooledConn struct {
	conn      net.Conn
	createdAt time.Time
	lastUsed  time.Time
	mu        sync.Mutex
}

type ConnPool struct {
	addr             string
	maxActive        int
	minIdle          int
	maxIdle          int
	idleTimeout      time.Duration
	connTTL          time.Duration
	maxWait          time.Duration
	dialFunc         func(addr string) (net.Conn, error)
	mu               sync.RWMutex
	idleConns        []*PooledConn
	activeCount      int32
	cond             *sync.Cond
	cleanerOnce      sync.Once
	closed           bool
	// 用于统计连接池的情况
	stats            struct {
		Hits     uint64
		Misses   uint64
		Timeouts uint64
		Errors   uint64
	}
}

func NewConnPool(addr string, maxActive, minIdle, maxIdle int, idleTimeout, connTTL, maxWait time.Duration, dialFunc func(string) (net.Conn, error)) *ConnPool {
	p := &ConnPool{
		addr:        addr,
		maxActive:   maxActive,
		minIdle:     minIdle,
		maxIdle:     maxIdle,
		idleTimeout: idleTimeout,
		connTTL:     connTTL,
		maxWait:     maxWait,
		dialFunc:    dialFunc,
	}
	p.cond = sync.NewCond(&p.mu)
	p.startCleaner()
	return p
}

func (p *ConnPool) Get() (net.Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}

	// 设置等待超时
	var timer *time.Timer
	if p.maxWait > 0 {
		timer = time.AfterFunc(p.maxWait, func() {
			p.cond.Broadcast()
		})
		defer timer.Stop()
	}

	for {
		// 检查空闲连接
		if len(p.idleConns) > 0 {
			conn := p.idleConns[len(p.idleConns)-1]
			p.idleConns = p.idleConns[:len(p.idleConns)-1]
			atomic.AddUint64(&p.stats.Hits, 1)

			// 连接健康检查
			if !p.isHealthy(conn) {
				conn.conn.Close()
				atomic.AddInt32(&p.activeCount, -1)
				continue
			}

			conn.lastUsed = time.Now()
			p.mu.Unlock()
			return &pooledConnWrapper{conn, p}, nil
		}

		// 检查是否可以创建新连接
		if int(atomic.LoadInt32(&p.activeCount)) < p.maxActive {
			atomic.AddInt32(&p.activeCount, 1)
			atomic.AddUint64(&p.stats.Misses, 1)
			p.mu.Unlock()

			conn, err := p.dialFunc(p.addr)
			if err != nil {
				atomic.AddInt32(&p.activeCount, -1)
				atomic.AddUint64(&p.stats.Errors, 1)
				p.cond.Signal()
				return nil, err
			}

			pooledConn := &PooledConn{
				conn:      conn,
				createdAt: time.Now(),
				lastUsed:  time.Now(),
			}
			return &pooledConnWrapper{pooledConn, p}, nil
		}

		// 等待连接释放
		if p.maxWait > 0 {
			atomic.AddUint64(&p.stats.Timeouts, 1)
			p.cond.Wait()
			if timer != nil && !timer.Stop() {
				p.mu.Unlock()
				return nil, ErrConnTimeout
			}
		} else {
			p.cond.Wait()
		}
	}
}

func (p *ConnPool) Put(conn net.Conn) {
	pc, ok := conn.(*pooledConnWrapper)
	if !ok {
		conn.Close()
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		pc.conn.conn.Close()
		return
	}

	// 检查连接是否健康
	if !p.isHealthy(pc.conn) {
		pc.conn.conn.Close()
		atomic.AddInt32(&p.activeCount, -1)
		p.cond.Signal()
		return
	}

	// 检查是否超过最大空闲连接数
	if len(p.idleConns) >= p.maxIdle {
		pc.conn.conn.Close()
		atomic.AddInt32(&p.activeCount, -1)
		p.cond.Signal()
		return
	}

	pc.conn.lastUsed = time.Now()
	p.idleConns = append(p.idleConns, pc.conn)
	p.cond.Signal()
}

func (p *ConnPool) isHealthy(pc *PooledConn) bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 检查连接是否超过TTL
	if time.Since(pc.createdAt) > p.connTTL {
		return false
	}

	// 设置读超时
	pc.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	defer pc.conn.SetReadDeadline(time.Time{})

	// 发送PING消息检查连接
	_, err := pc.conn.Write([]byte{0x01})
	if err != nil {
		return false
	}

	// 读取响应
	var one [1]byte
	_, err = pc.conn.Read(one[:])
	return err == nil
}

func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	for _, conn := range p.idleConns {
		conn.conn.Close()
	}
	p.idleConns = nil
	p.cond.Broadcast()
}

func (p *ConnPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"active":  atomic.LoadInt32(&p.activeCount),
		"idle":    len(p.idleConns),
		"hits":    atomic.LoadUint64(&p.stats.Hits),
		"misses":  atomic.LoadUint64(&p.stats.Misses),
		"timeouts": atomic.LoadUint64(&p.stats.Timeouts),
		"errors":  atomic.LoadUint64(&p.stats.Errors),
	}
}

// pooledConnWrapper 包装连接，确保正确返回到连接池
type pooledConnWrapper struct {
	conn *PooledConn
	pool *ConnPool
}

func (w *pooledConnWrapper) Read(b []byte) (n int, err error) {
	return w.conn.conn.Read(b)
}

func (w *pooledConnWrapper) Write(b []byte) (n int, err error) {
	return w.conn.conn.Write(b)
}

func (w *pooledConnWrapper) Close() error {
	w.pool.Put(w)
	return nil
}

func (w *pooledConnWrapper) LocalAddr() net.Addr {
	return w.conn.conn.LocalAddr()
}

func (w *pooledConnWrapper) RemoteAddr() net.Addr {
	return w.conn.conn.RemoteAddr()
}

func (w *pooledConnWrapper) SetDeadline(t time.Time) error {
	return w.conn.conn.SetDeadline(t)
}

func (w *pooledConnWrapper) SetReadDeadline(t time.Time) error {
	return w.conn.conn.SetReadDeadline(t)
}

func (w *pooledConnWrapper) SetWriteDeadline(t time.Time) error {
	return w.conn.conn.SetWriteDeadline(t)
} 

func (p *ConnPool) startCleaner() {
	p.cleanerOnce.Do(func() {
		go func() {
			for {
				time.Sleep(p.idleTimeout / 2)
				p.cleanIdle()
			}
		}()
	})
}

func (p *ConnPool) cleanIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	var retained []*PooledConn
	for _, c := range p.idleConns {
		if now.Sub(c.createdAt) > p.idleTimeout {
			c.conn.Close()
			p.activeCount--
		} else {
			retained = append(retained, c)
		}
	}
	p.idleConns = retained
	p.cond.Broadcast()
}
