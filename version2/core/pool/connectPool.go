package pool

import (
	"errors"
	"fmt"
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
	mu          sync.RWMutex                   // 读写锁，用于保护连接池的并发访问
	cond        *sync.Cond                     // 条件变量，用于等待可用连接
	addr        string                         // 服务器地址，格式为 "host:port"
	maxActive   int                            // 连接池最大活跃连接数，当达到这个数量时，新的请求会等待
	maxIdle     int                            // 连接池最大空闲连接数，超过这个数量的空闲连接会被关闭
	idleTimeout time.Duration                  // 空闲连接的最大存活时间，超过这个时间的空闲连接会被清理
	maxWait     time.Duration                  // 获取连接的最大等待时间，超过这个时间还未获取到连接会返回超时错误
	dialFunc    func(string) (net.Conn, error) // 创建新连接的函数，可以自定义连接创建逻辑
	idleConns   []*PooledConn                  // 空闲连接列表
	activeCount int32                          // 当前活跃连接数（使用原子操作）
	cleanerOnce sync.Once                      // 确保清理器只启动一次
	closed      bool                           // 连接池是否已关闭
	// 用于统计连接池的情况
	stats struct {
		Hits     uint64
		Misses   uint64
		Timeouts uint64
		Errors   uint64
	}
}

func NewConnPool(addr string, maxActive, maxIdle int, idleTimeout, maxWait time.Duration, dialFunc func(string) (net.Conn, error)) *ConnPool {
	p := &ConnPool{
		addr:        addr,
		maxActive:   maxActive,
		maxIdle:     maxIdle,
		idleTimeout: idleTimeout,
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

			// 重置所有超时设置
			conn.conn.SetDeadline(time.Time{})
			conn.conn.SetReadDeadline(time.Time{})
			conn.conn.SetWriteDeadline(time.Time{})

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
			// 如果定时器触发了，说明是超时了
			// 如果定时器没有触发，说明是Put方法的唤醒了，有空闲新连接，继续获取
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
	pc.conn.SetReadDeadline(time.Now().Add(time.Millisecond)) // 设置超时时间为 1 毫秒
	buf := make([]byte, 1)                                    // 创建一个缓冲区
	n, err := pc.conn.Read(buf)                               // 尝试从连接读取数据到缓冲区

	// 如果没有错误或读取了数据，则说明发生了意外的读取
	if err == nil || n > 0 {
		return false
	}

	// 如果是超时错误，说明连接空闲，但没有读取到数据，认为是正常的超时
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true // 无错误，正常超时
	}

	// 处理其他连接错误，包括连接被关闭等
	return false
}

func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	fmt.Println("Closing now connection pool")
	p.closed = true
	for _, conn := range p.idleConns {
		// 打印关闭的连接
		fmt.Println("Closing idle connection")
		conn.conn.Close()
	}
	p.idleConns = nil
	p.cond.Broadcast()
}

func (p *ConnPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"active":   atomic.LoadInt32(&p.activeCount),
		"idle":     len(p.idleConns),
		"hits":     atomic.LoadUint64(&p.stats.Hits),
		"misses":   atomic.LoadUint64(&p.stats.Misses),
		"timeouts": atomic.LoadUint64(&p.stats.Timeouts),
		"errors":   atomic.LoadUint64(&p.stats.Errors),
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
