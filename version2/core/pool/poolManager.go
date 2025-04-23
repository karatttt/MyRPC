package pool

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	// 全局唯一的 PoolManager 实例
	globalPoolManager *PoolManager
	// 用于确保 PoolManager 只被初始化一次
	poolManagerOnce sync.Once
)

type PoolManager struct {
	mu       sync.RWMutex
	pools    map[string]*ConnPool
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown chan struct{}
	sigChan  chan os.Signal
}

// GetPoolManager 获取全局唯一的 PoolManager 实例
func GetPoolManager() *PoolManager {
	poolManagerOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		globalPoolManager = &PoolManager{
			pools:    make(map[string]*ConnPool),
			ctx:      ctx,
			cancel:   cancel,
			shutdown: make(chan struct{}),
			sigChan:  make(chan os.Signal, 1),
		}

		// 启动信号处理
		go globalPoolManager.handleSignals()
	})
	return globalPoolManager
}

// handleSignals 处理系统信号
func (pm *PoolManager) handleSignals() {
	// 注册信号
	signal.Notify(pm.sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-pm.sigChan
	fmt.Printf("\nReceived signal: %v\n", sig)

	// 创建一个带超时的上下文用于关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭连接池
	fmt.Println("Shutting down connection pools...")
	if err := pm.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
	}
	fmt.Println("Connection pools shut down successfully")

	// 通知关闭完成
	close(pm.shutdown)
}

// GetPool 获取指定地址的连接池，如果不存在则创建
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

	// 双重检查，防止其他goroutine已经创建
	if pool, exists := pm.pools[addr]; exists {
		return pool
	}

	pool := NewConnPool(
		addr,           // 服务器地址
		200,             // 最大活跃连接数
		200,              // 最小空闲连接数
		60*time.Second, // 空闲连接超时时间
		60*time.Second, // 建立连接最大生命周期
		func(address string) (net.Conn, error) {
			return net.DialTimeout("tcp", address, 60*time.Second)
		},
	)
	pm.pools[addr] = pool
	return pool
}

// Shutdown 优雅关闭所有连接池
func (pm *PoolManager) Shutdown(ctx context.Context) error {
	// 发送关闭信号
	pm.cancel()

	// 等待所有连接池关闭
	done := make(chan struct{})
	go func() {
		pm.mu.Lock()
		for _, pool := range pm.pools {
			pool.Close()
		}
		pm.pools = make(map[string]*ConnPool)
		pm.mu.Unlock()
		close(done)
	}()

	// 等待关闭完成或超时
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForShutdown 等待关闭完成
func (pm *PoolManager) WaitForShutdown() {
	<-pm.shutdown
}
