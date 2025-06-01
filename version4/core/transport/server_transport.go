package transport

import (
	"MyRPC/core/codec"
	netxConn "MyRPC/netx/connection"
	"MyRPC/netx/poller"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/panjf2000/ants/v2"
)

// 实现ServerTransport接口
type serverTransport struct {
	// 这个就是Service
	ConnHandler Handler
	opts        *ServerTransportOption
	// 添加协程池
	pool *ants.Pool
}

var DefaultServerTransport = NewServerTransport()

func NewServerTransport() ServerTransport {
	// 创建一个容量为10000的协程池
	pool, err := ants.NewPool(10000, ants.WithPreAlloc(true))
	if err != nil {
		fmt.Printf("create ants pool error: %v\n", err)
		return nil
	}

	return &serverTransport{
		pool: pool,
		opts: &ServerTransportOption{},
	}
}

type Request struct {
	Method string `json:"method"` // 请求的方法名
	Body   []byte `json:"body"`   // 请求的参数
}

// ListenAndServe 监听并处理 TCP 连接
func (t *serverTransport) ListenAndServe(ctx context.Context, network, address string) error {

	ln, err := net.Listen(network, address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	return t.serveTCP(ctx, ln)
}

// serveTCP 处理 TCP 连接
func (t *serverTransport) serveTCP(ctx context.Context, ln net.Listener) error {
	//初始化事件循环
	eventLoop, err := poller.NewEventLoop(t.OnRequest)
	if err != nil {
		return fmt.Errorf("failed to create event loop: %w", err)
	}
	err = eventLoop.Serve(ln)
	if err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}

// handleConnection 处理单个连接
func (t *serverTransport) OnRequest(conn net.Conn) error {
	// 设置连接超时
	idleTimeout := 30 * time.Second
	if t.opts != nil && t.opts.IdleTimeout > 0 {
		idleTimeout = t.opts.IdleTimeout
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	// 处理连接
	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	frame, err := codec.ReadFrame(conn)
	if err != nil {
		// 2. 如果读取帧失败，如客户端断开连接，则关闭连接
		if err == io.EOF {
			fmt.Printf("Client %s disconnected normally\n", conn.RemoteAddr())
			return err
		}
		// 3. 如果连接超时，超过设置的idletime，关闭连接
		if e, ok := err.(net.Error); ok && e.Timeout() {
			fmt.Printf("Connection from %s timed out after %v\n", conn.RemoteAddr(), idleTimeout)
			return err
		}
		// 4. 处理强制关闭的情况
		if strings.Contains(err.Error(), "forcibly closed") {
			fmt.Printf("Client %s forcibly closed the connection\n", conn.RemoteAddr())
			return err
		}
		fmt.Printf("Read error from %s: %v\n", conn.RemoteAddr(), err)
		return err
	}

	// 重置读取超时
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	// 使用协程池处理请求，适用于多路复用模式
	frameCopy := frame // 创建副本避免闭包问题
	err = t.pool.Submit(func() {
		// 处理请求
		response, err := t.ConnHandler.Handle(context.Background(), frameCopy)
		if err != nil {
			fmt.Printf("Handle error for %s: %v\n", conn.RemoteAddr(), err)
			return
		}

		// 发送响应
		conn = conn.(netxConn.Connection) // 确保conn实现了Connection接口，调用聚集发包的接口
		if _, err := conn.Write(response); err != nil {
			fmt.Printf("Write response error for %s: %v\n", conn.RemoteAddr(), err)
		}
	})

	if err != nil {
		fmt.Printf("Submit task to pool error for %s: %v\n", conn.RemoteAddr(), err)
		// 协程池提交失败，直接处理
		response, err := t.ConnHandler.Handle(context.Background(), frame)
		if err != nil {
			fmt.Printf("Handle error for %s: %v\n", conn.RemoteAddr(), err)
		}
		if _, err := conn.Write(response); err != nil {
			fmt.Printf("Write response error for %s: %v\n", conn.RemoteAddr(), err)
			return err
		}
	}
	return nil
}

// 将Handler注册到ServerTransport中
func (t *serverTransport) RegisterHandler(handler Handler) {
	t.ConnHandler = handler
}

// SetKeepAlivePeriod 设置保活时间
func (t *serverTransport) SetKeepAlivePeriod(time time.Duration) {
	if t.opts == nil {
		t.opts = &ServerTransportOption{}
	}
	t.opts.KeepAlivePeriod = time
}

// SetIdleTimeout 设置空闲超时
func (t *serverTransport) SetIdleTimeout(time time.Duration) {
	if t.opts == nil {
		t.opts = &ServerTransportOption{}
	}
	t.opts.IdleTimeout = time
}
