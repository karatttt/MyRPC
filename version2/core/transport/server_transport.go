package transport

import (
	"MyRPC/core/codec"
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
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	return t.serveTCP(ctx, ln)
}

// serveTCP 处理 TCP 连接
func (t *serverTransport) serveTCP(ctx context.Context, ln net.Listener) error {
	fmt.Print("开始监听TCP连接")
	for {
		conn, err := ln.Accept()
		// 设置保活机制SetKeepAlive

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			if err := tcpConn.SetKeepAlive(true); err != nil {
				fmt.Printf("tcp conn set keepalive error:%v", err)
			}
			if t.opts.KeepAlivePeriod > 0 {
				if err := tcpConn.SetKeepAlivePeriod(t.opts.KeepAlivePeriod); err != nil {
					fmt.Printf("tcp conn set keepalive period error: %v\n", err)
				}
			}
		}
		if err != nil {
			select {
			case <-ctx.Done():
				return nil // 退出监听
			default:
				fmt.Println("accept error:", err)
			}
			continue
		}

		go t.handleConnection(ctx, conn)
	}
}

// handleConnection 处理单个连接
func (t *serverTransport) handleConnection(ctx context.Context, conn net.Conn) {
	// 设置连接超时
	idleTimeout := 30 * time.Second
	if t.opts != nil && t.opts.IdleTimeout > 0 {
		idleTimeout = t.opts.IdleTimeout
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	// 处理连接
	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	// 循环读取请求，即长连接
	for {
		select {
		// 1. 如果上下文被取消，则关闭连接
		case <-ctx.Done():
			fmt.Printf("Context cancelled, closing connection from %s\n", conn.RemoteAddr())
			return
		default:
			frame, err := codec.ReadFrame(conn)
			if err != nil {
				// 2. 如果读取帧失败，如客户端断开连接，则关闭连接
				if err == io.EOF {
					fmt.Printf("Client %s disconnected normally\n", conn.RemoteAddr())
					return
				}
				// 3. 如果连接超时，超过设置的idletime，关闭连接
				if e, ok := err.(net.Error); ok && e.Timeout() {
					fmt.Printf("Connection from %s timed out after %v\n", conn.RemoteAddr(), idleTimeout)
					return
				}
				// 4. 处理强制关闭的情况
				if strings.Contains(err.Error(), "forcibly closed") {
					fmt.Printf("Client %s forcibly closed the connection\n", conn.RemoteAddr())
					return
				}
				fmt.Printf("Read error from %s: %v\n", conn.RemoteAddr(), err)
				return
			}

			// 重置读取超时
			conn.SetReadDeadline(time.Now().Add(idleTimeout))

			// 使用协程池处理请求
			frameCopy := frame // 创建副本避免闭包问题
			err = t.pool.Submit(func() {
				// 处理请求
				response, err := t.ConnHandler.Handle(context.Background(), frameCopy)
				if err != nil {
					fmt.Printf("Handle error for %s: %v\n", conn.RemoteAddr(), err)
					return
				}

				// 发送响应
				if _, err := conn.Write(response); err != nil {
					fmt.Printf("Write response error for %s: %v\n", conn.RemoteAddr(), err)
				}
			})

			if err != nil {
				fmt.Printf("Submit task to pool error for %s: %v\n", conn.RemoteAddr(), err)
				// 协程池提交失败，直接处理
				response, err := t.ConnHandler.Handle(ctx, frame)
				if err != nil {
					fmt.Printf("Handle error for %s: %v\n", conn.RemoteAddr(), err)
					continue
				}

				if _, err := conn.Write(response); err != nil {
					fmt.Printf("Write response error for %s: %v\n", conn.RemoteAddr(), err)
					return
				}
			}
		}
	}
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
