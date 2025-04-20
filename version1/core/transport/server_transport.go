package transport

import (
	"MyRPC/core/codec"
	"context"
	"fmt"
	"net"
)

// 实现ServerTransport接口
type serverTransport struct {
	// 这个就是Service
	ConnHandler Handler
}

var DefaultServerTransport = NewServerTransport()

func NewServerTransport() ServerTransport {
	return &serverTransport{}
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
	//TODO 这里可以做一个处理业务逻辑的协程池
	// 实际上每个连接一个协程，同时负责读取请求并直接处理业务逻辑也是可行的，读取请求时如果阻塞，Go调度器会自动切换到其他协程执行
	// 但是协程池可以限制同时处理业务逻辑的协程数量，避免请求量大时，过多协程导致的资源消耗

	// 这里是处理完一个请求就释放连接，后续可以考虑长连接
	defer conn.Close()
	fmt.Println("New connection from", conn.RemoteAddr())
	// 读取帧
	frame, err := codec.ReadFrame(conn)
	if err != nil {
		fmt.Println("read frame error:", err)
		return
	}

	// 调用service的Handler执行结果
	response, err := t.ConnHandler.Handle(ctx, frame)
	if err != nil {
		fmt.Println("handle error:", err)
		return
	}

	// 发送响应，此时已经是完整帧
	conn.Write(response)
}

// 将Handler注册到ServerTransport中
func (t *serverTransport) RegisterHandler(handler Handler) {
	t.ConnHandler = handler
}
