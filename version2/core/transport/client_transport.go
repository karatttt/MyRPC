package transport

import (
	"MyRPC/core/codec"
	"MyRPC/core/internel"
	"MyRPC/core/pool"
	"context"
	"fmt"
	"net"
)

type clientTransport struct{}

// 默认的clientTransport，需要实现接口方法，才能被外部的参数类型适配，因为外部用接口来接受这个实例
// 小写类名，可以类比为java中的私有构造方法，要么提供默认值，要么提供单例，要么提供工厂方法
var DefaultClientTransport = NewClientTransport()

func NewClientTransport() ClientTransport {
	return &clientTransport{}
}

// 实现Send方法
func (c *clientTransport) Send(ctx context.Context, reqBody interface{}, rspBody interface{}, opt *ClientTransportOption) error {
	// 获取连接
	pool := pool.GetPoolManager().GetPool(opt.Address)
	conn, err := pool.Get()
	if err != nil {
		return err
	}
	defer pool.Put(conn)

	// reqbody序列化
	reqData, err := codec.Marshal(reqBody)
	if err != nil {
		return err
	}

	// reqbody编码，返回请求帧
	framedata, err := opt.Codec.Encode(ctx, reqData)
	if err != nil {
		return err
	}

	// 写数据到连接中
	err = c.tcpWriteFrame(ctx, conn, framedata)
	if err != nil {
		return err
	}

	// 读取tcp帧
	rspDataBuf, err := c.tcpReadFrame(ctx, conn)
	if err != nil {
		return err
	}
	// 获取msg
	ctx, msg := internel.GetMessage(ctx)
	// rspDataBuf解码，提取响应体数据
	rspData, err := opt.Codec.Decode(msg, rspDataBuf)
	if err != nil {
		return err
	}

	// 将rspData反序列化为rspBody
	err = codec.Unmarshal(rspData, rspBody)
	if err != nil {
		return err
	}
	return nil
}

func (c *clientTransport) tcpWriteFrame(ctx context.Context, conn net.Conn, frame []byte) error {

	// 写入tcp
	_, err := conn.Write(frame)
	if err != nil {
		return fmt.Errorf("write frame error: %v", err)
	}
	return nil
}

func (c *clientTransport) tcpReadFrame(ctx context.Context, conn net.Conn) ([]byte, error) {
	return codec.ReadFrame(conn)
}
