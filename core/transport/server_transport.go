package transport

import (
	"context"
	"fmt"
	"net"
	"encoding/json"
)

type Transport interface {
	ListenAndServe(ctx context.Context, network, address string) error
}

// 定义一个Handler接口，service实现了这个接口
type Handler interface {
	Handle(req []byte) (rsp []byte, err error)
}


type ServerTransport struct {
	// 这个就是Service
	ConnHandler Handler
}

type Request struct {
	Method string `json:"method"` // 请求的方法名
	Body   []byte `json:"body"`   // 请求的参数
}


// ListenAndServe 监听并处理 TCP 连接
func (t *ServerTransport)ListenAndServe(ctx context.Context, network, address string) error {

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
func (t *ServerTransport) serveTCP(ctx context.Context, ln net.Listener) error {
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
		
		go t.handleConnection(conn)
	}
}

// handleConnection 处理单个连接
func  (t *ServerTransport) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("New connection from", conn.RemoteAddr())
	
	//读取请求头
	header := make([]byte, 1024)
	n, err := conn.Read(header)
	if err != nil {
		fmt.Println("read header error:", err)
		return
	}

	// 读取请求体
	body := make([]byte, n)
	n, err = conn.Read(body)
	if err != nil {
		fmt.Println("read body error:", err)
		return
	}

	// 获取请求体中的methodName字段的值
	request := &Request{}
	err = json.Unmarshal(body, request)
	if err != nil {
		fmt.Println("unmarshal request error:", err)
		return
	}
	// 获取Handler执行结果
	response, err := t.ConnHandler.Handle(request.Body)
	if err != nil {
		fmt.Println("handle error:", err)
		return
	}

	// 发送响应
	responseBody, err := json.Marshal(response)
	if err != nil {
		fmt.Println("marshal response error:", err)
		return
	}
	conn.Write(responseBody)	
}
