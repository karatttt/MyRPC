package transport

import (
	"context"
	"fmt"
	"net"
	"MyRPC/core/codec"
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
func (t *serverTransport)ListenAndServe(ctx context.Context, network, address string) error {

	// TODO 如果前面的service已经listen过了，这里实际上需要复用
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
func  (t *serverTransport) handleConnection(ctx context.Context, conn net.Conn) {
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

