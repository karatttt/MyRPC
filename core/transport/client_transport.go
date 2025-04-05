package transport

import (
	"bufio"
	"context"
	"io"
	"net"
)

type clientTransport struct{}

// 默认的clientTransport，需要实现接口方法，才能被外部的参数类型适配，因为外部用接口来接受这个实例
// 小写类名，可以类比为java中的私有构造方法，要么提供默认值，要么提供单例，要么提供工厂方法
var DefaultClientTransport = NewClientTransport()

func NewClientTransport() ClientTransport {
	return &clientTransport{}
}

type ClientTransport interface {
	Send(ctx context.Context, reqBody interface{}, opt *ClientTransportOption) (rspBody interface{}, err error)
}

// 实现Send方法
func (c *clientTransport) Send(ctx context.Context, reqBody interface{}, opt *ClientTransportOption) (rspBody interface{}, err error) {
	// 获取连接
	// TODO 这里的连接后续可以优化从连接池获取
	conn, err := net.Dial("tcp", opt.Address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 将reqbody编码为二进制
	reqData, err := opt.Codec.Encode(ctx, reqBody)

	// 将req写到tcp帧中
	err = c.tcpWriteFrame(ctx, conn, reqData)
	if err != nil {
		return nil, err
	}

	// 读取tcp帧
	rspData, err := c.tcpReadFrame(ctx, conn)
	if err != nil {
		return nil, err
	}

	// 将rspData解码为rspBody
	rspBody, err = opt.Codec.Decode(ctx, rspData)
	if err != nil {
		return nil, err
	}
	return rspBody, nil
}

func (c *clientTransport) tcpWriteFrame(ctx context.Context, conn net.Conn, reqData []byte) error {
	// Send package in a loop.
	sentNum := 0
	num := 0
	var err error
	for sentNum < len(reqData) {
		num, err = conn.Write(reqData[sentNum:])
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				return err
			}
			return err
		}
		sentNum += num
	}
	return nil
}

func (c *clientTransport) tcpReadFrame(ctx context.Context, conn net.Conn) ([]byte, error) {
	// 读取TCP帧并存到缓冲区
	buf := bufio.NewReader(conn)

	// 读取帧头
	frameHeader := make([]byte, 4)
	_, err := io.ReadFull(buf, frameHeader)
	if err != nil {
		return nil, err
	}

	// 读取帧体
	frameBody := make([]byte, frameHeader[3])
	_, err = io.ReadFull(buf, frameBody)
	if err != nil {
		return nil, err
	}

	return frameBody, nil
}
