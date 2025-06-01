package transport

import (
	"MyRPC/common"
	"MyRPC/core/codec"
	"MyRPC/core/internel"
	"MyRPC/core/mutilpath"
	"MyRPC/core/pool"
	"context"
	"fmt"
	"net"
	"time"
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
	// 创建一个done通道用于监听上下文取消
	var done chan struct{}
	// 判断当前是否设置了超时时间
	if opt.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		done = make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				close(done)
			}
		}()
		defer cancel()
	}

	// 获取连接
	conn, ctx, _, err := fetchConn(ctx, opt)
	//defer pool.Put(conn)
	if err != nil {
		return &common.RPCError{
			Code:    common.ErrCodeNetwork,
			Message: fmt.Sprintf("failed to get connection: %v", err),
		}
	}
	// reqbody序列化
	reqData, err := codec.Marshal(reqBody)
	if err != nil {
		return &common.RPCError{
			Code:    common.ErrCodeSerialization,
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	// reqbody编码，返回请求帧
	framedata, err := opt.Codec.Encode(ctx, reqData)
	if err != nil {
		return &common.RPCError{
			Code:    common.ErrCodeEncoding,
			Message: fmt.Sprintf("failed to encode request: %v", err),
		}
	}

	// 获取msg
	ctx, msg := internel.GetMessage(ctx)
	rspDataBuf := make([]byte, 0)

	if !opt.MuxOpen {
		// 正常模式
		// 写数据到连接中
		err = c.tcpWriteFrame(ctx, conn, framedata)
		if err != nil {
			return &common.RPCError{
				Code:    common.ErrCodeNetwork,
				Message: fmt.Sprintf("failed to write frame: %v", err),
			}
		}

		// 读取tcp帧
		rspDataBuf, err = c.tcpReadFrame(ctx, conn)
		if err != nil {
			// 先检查是否设置了超时时间，是超时错误则返回超时错误，否则返回网络错误
			if opt.Timeout > 0 {
				select {
				case <-done:
					return &common.RPCError{
						Code:    common.ErrCodeTimeout,
						Message: "request timeout",
					}
				}
			}
			return &common.RPCError{
				Code:    common.ErrCodeNetwork,
				Message: fmt.Sprintf("failed to read frame: %v", err),
			}
		}
	} else {
		// mux模式下，通过ch阻塞等待相应的流回包
		muxConn, _ := conn.(*mutilpath.MuxConn)
		seqID := msg.GetSequenceID()
		ch := muxConn.RegisterPending(seqID)
		defer muxConn.UnregisterPending(seqID)

		// 写数据
		err = c.tcpWriteFrame(ctx, conn, framedata)
		if err != nil {
			return &common.RPCError{
				Code:    common.ErrCodeNetwork,
				Message: fmt.Sprintf("failed to write frame: %v", err),
			}
		}

		// 读响应
		select {
		case frame := <-ch:
			rspDataBuf = frame.Data
		case <-ctx.Done():
			return &common.RPCError{
				Code:    common.ErrCodeNetwork,
				Message: fmt.Sprintf("failed to read frame: %v", err),
			}
		}
	}

	// rspDataBuf解码，提取响应体数据
	rspData, err := opt.Codec.Decode(msg, rspDataBuf)
	if err != nil {
		return &common.RPCError{
			Code:    common.ErrCodeDecoding,
			Message: fmt.Sprintf("failed to decode response: %v", err),
		}
	}

	// 将rspData反序列化为rspBody
	err = codec.Unmarshal(rspData, rspBody)
	if err != nil {
		return &common.RPCError{
			Code:    common.ErrCodeDeserialization,
			Message: fmt.Sprintf("failed to unmarshal response: %v", err),
		}
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

func fetchConn(ctx context.Context, opt *ClientTransportOption) (conn net.Conn, ctxRes context.Context, poolRes *pool.ConnPool, err error) {
	if !opt.MuxOpen {
		poolRes := pool.GetPoolManager().GetPool(opt.Address, 1000, 1000, 60*time.Second, 60*time.Second, false)
		conn, err := poolRes.Get()
		if err != nil {
			return nil, ctx, poolRes, err
		}
		return conn, ctx, poolRes, nil
	} else {
		poolRes := pool.GetPoolManager().GetPool(opt.Address, 8, 8, 60*time.Second, 60*time.Second, true)
		conn, err := poolRes.Get()
		if err != nil {
			return nil, ctx, poolRes, err
		}
		// 获取msg，开启mux，并设置sequenceID
		ctx, msg := internel.GetMessage(ctx)
		msg.WithMuxOpen(opt.MuxOpen)
		msg.WithSequenceID(poolRes.GetSequenceIDByMuxConn(conn))
		return conn, ctx, poolRes, nil
	}
}
