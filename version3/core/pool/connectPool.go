package pool

import (
	"MyRPC/core/mutilpath"
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
)

type PooledConn struct {
	conn      net.Conn
	createdAt time.Time
	lastUsed  time.Time
}

type ConnPool struct {
	mu          sync.RWMutex
	cond        *sync.Cond
	addr        string
	maxActive   int
	maxIdle     int
	idleTimeout time.Duration
	maxWait     time.Duration
	dialFunc    func(string) (net.Conn, error)
	idleConns   []*PooledConn
	activeCount int32          // 当前活跃连接数
	closing     bool           // 正在关闭标志
	closed      bool           // 完全关闭标志
	wg          sync.WaitGroup // 等待活跃连接完成
	cleanerOnce sync.Once
	stats       struct {
		Hits     uint64
		Misses   uint64
		Timeouts uint64
		Errors   uint64
	}

	// 多路复用相关字段
	isMux       bool                               // 是否启用多路复用
	muxConns    map[*PooledConn]*mutilpath.MuxConn // 多路复用连接映射，每一个连接对应一个MuxConn，实际上MuxConn也是conn的一个封装
	maxStreams  int                                // 每个muxconn连接最大流数
	streamCount map[*PooledConn]int                // 连接对应的muxConn连接的当前流数

}

var MuxConn2SequenceIDMap = make(map[net.Conn]uint64) // 记录每个MuxConn目前分配至的ID

func NewConnPool(addr string, maxActive, maxIdle int, idleTimeout, maxWait time.Duration, dialFunc func(string) (net.Conn, error), isMux bool) *ConnPool {
	p := &ConnPool{
		addr:        addr,
		maxActive:   maxActive,
		maxIdle:     maxIdle,
		idleTimeout: idleTimeout,
		maxWait:     maxWait,
		dialFunc:    dialFunc,
		isMux:       isMux,
	}
	p.cond = sync.NewCond(&p.mu)
	p.startCleaner()
	return p
}
func (p *ConnPool) Get() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// 记录开始等待时间
	startWait := time.Now()

	// 设置等待超时
	var timer *time.Timer
	if p.maxWait > 0 {
		timer = time.AfterFunc(p.maxWait, func() {
			p.cond.Broadcast()
		})
		defer timer.Stop()
	}

	for {
		// 1. 优先检查空闲连接
		if len(p.idleConns) > 0 {
			conn := p.idleConns[len(p.idleConns)-1]
			p.idleConns = p.idleConns[:len(p.idleConns)-1]
			atomic.AddUint64(&p.stats.Hits, 1)

			// 从空闲变为活跃
			atomic.AddInt32(&p.activeCount, 1)
			p.wg.Add(1)

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

			// 多路复用处理
			if p.isMux {
				if muxConn, exists := p.muxConns[conn]; exists {
					if p.streamCount[conn] < p.maxStreams {
						p.streamCount[conn]++
						MuxConn2SequenceIDMap[muxConn] = muxConn.NextRequestID()
						return muxConn, nil
					}
				}
				// 如果不是多路复用连接或已达最大流数，回退到普通连接
			}

			p.mu.Unlock()
			return &pooledConnWrapper{conn, p}, nil
		}

		// 2. 检查是否可以创建新连接
		if int(atomic.LoadInt32(&p.activeCount)) < p.maxActive {
			atomic.AddInt32(&p.activeCount, 1)
			p.wg.Add(1)
			atomic.AddUint64(&p.stats.Misses, 1)
			p.mu.Unlock()

			rawConn, err := p.dialFunc(p.addr)
			if err != nil {
				atomic.AddInt32(&p.activeCount, -1)
				p.wg.Done()
				atomic.AddUint64(&p.stats.Errors, 1)
				p.mu.Lock()
				p.cond.Signal()
				p.mu.Unlock()
				return nil, err
			}

			pooledConn := &PooledConn{
				conn:      rawConn,
				createdAt: time.Now(),
				lastUsed:  time.Now(),
			}

			p.mu.Lock()
			// 多路复用连接初始化
			if p.isMux {
				if p.muxConns == nil {
					p.muxConns = make(map[*PooledConn]*mutilpath.MuxConn)
					p.streamCount = make(map[*PooledConn]int)
				}
				muxConn := mutilpath.NewMuxConn(rawConn, 1000)
				p.muxConns[pooledConn] = muxConn
				p.streamCount[pooledConn] = 1 // 新连接默认1个流
				MuxConn2SequenceIDMap[muxConn] = muxConn.NextRequestID()
				return muxConn, nil
			}
			p.mu.Unlock()

			return &pooledConnWrapper{pooledConn, p}, nil
		}

		// 3. 无空闲且活跃连接达到最大数，检查活跃连接的多路复用能力（仅在多路复用模式下）
		if p.isMux {
			for pc, muxConn := range p.muxConns {
				count := p.streamCount[pc]
				if count < p.maxStreams {
					p.streamCount[pc]++
					atomic.AddInt32(&p.activeCount, 1)
					pc.lastUsed = time.Now()
					MuxConn2SequenceIDMap[muxConn] = muxConn.NextRequestID()
					return p.muxConns[pc], nil
				}
			}
		}

		// 4. 处理连接池已满的情况
		if p.maxWait > 0 {
			// 检查是否已超时
			if timer != nil && !timer.Stop() {
				waitTime := time.Since(startWait)
				return nil, fmt.Errorf("connection pool exhausted, wait timeout after %v", waitTime)
			}
			// 记录超时统计
			atomic.AddUint64(&p.stats.Timeouts, 1)
			// 等待连接释放
			p.cond.Wait()
		} else {
			// 无等待时间，直接返回错误
			return nil, ErrPoolExhausted
		}
	}
}

// 连接池获取连接
// 对于多路复用连接muxConn，不再使用Put方式归还连接，而是直接close
func (p *ConnPool) Put(conn net.Conn) {
	if p.isMux {
		p.PutMuxConn(conn)
		return
	}
	pc, ok := conn.(*pooledConnWrapper)
	if !ok {
		conn.Close()
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 减少活跃计数
	atomic.AddInt32(&p.activeCount, -1)
	p.wg.Done()

	// 如果连接池正在关闭或已关闭，直接关闭连接
	if p.closing || p.closed {
		pc.conn.conn.Close()
		p.cond.Signal()
		return
	}

	// 检查连接是否健康
	if !p.isHealthy(pc.conn) {
		pc.conn.conn.Close()
		p.cond.Signal()
		return
	}

	// 检查是否超过最大空闲连接数
	if len(p.idleConns) >= p.maxIdle {
		pc.conn.conn.Close()
		p.cond.Signal()
		return
	}

	pc.conn.lastUsed = time.Now()
	p.idleConns = append(p.idleConns, pc.conn)
	p.cond.Signal()
}

func (p *ConnPool) PutMuxConn(conn net.Conn) {
	// 多路复用模式
	muxConn, ok := conn.(*mutilpath.MuxConn)
	if !ok {
		conn.Close()
		return
	}
	// 找到对应的 PooledConn
	var pooled *PooledConn
	for pc, mc := range p.muxConns {
		if mc == muxConn {
			pooled = pc
			break
		}
	}
	if pooled == nil {
		muxConn.Close()
		return
	}
	// 归还一个流
	if p.streamCount[pooled] > 0 {
		p.streamCount[pooled]--
		atomic.AddInt32(&p.activeCount, -1)
		p.wg.Done()
	}
	// 如果流为0，放回空闲池
	if p.streamCount[pooled] == 0 {
		pooled.lastUsed = time.Now()
		if len(p.idleConns) < p.maxIdle && !p.closing && !p.closed {
			p.idleConns = append(p.idleConns, pooled)
		} else {
			muxConn.Close()
			delete(p.muxConns, pooled)
			delete(p.streamCount, pooled)
		}
	}
	p.cond.Signal()
}

func (p *ConnPool) isHealthy(pc *PooledConn) bool {
	pc.conn.SetReadDeadline(time.Now().Add(time.Millisecond))
	buf := make([]byte, 1)
	n, err := pc.conn.Read(buf)

	if err == nil || n > 0 {
		return false
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	return false
}

func (p *ConnPool) Shutdown(timeout time.Duration) error {
	p.mu.Lock()
	// 标记为正在关闭，不再接受新连接
	p.closing = true
	p.mu.Unlock()

	// 关闭所有空闲连接
	p.mu.Lock()
	for _, conn := range p.idleConns {
		conn.conn.Close()
	}
	p.idleConns = nil
	p.mu.Unlock()

	// 等待活跃连接完成或超时
	done := make(chan struct{})
	go func() {
		p.wg.Wait() // 等待所有活跃连接归还
		close(done)
	}()

	select {
	case <-done:
		// 所有活跃连接已完成
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		return nil
	case <-time.After(timeout):
		// 超时，强制关闭
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		return fmt.Errorf("连接池关闭超时，仍有 %d 个活跃连接", atomic.LoadInt32(&p.activeCount))
	}
}

func (p *ConnPool) Close() {
	// 默认给5秒超时
	if err := p.Shutdown(5 * time.Second); err != nil {
		fmt.Println("连接池关闭警告:", err)
	}
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
		if now.Sub(c.lastUsed) > p.idleTimeout {
			c.conn.Close()
			atomic.AddInt32(&p.activeCount, -1)
			p.wg.Done()
		} else {
			retained = append(retained, c)
		}
	}
	p.idleConns = retained
	p.cond.Broadcast()
}

func (p *ConnPool) GetSequenceIDByMuxConn(conn net.Conn) uint32 {
	return uint32(MuxConn2SequenceIDMap[conn])
}
