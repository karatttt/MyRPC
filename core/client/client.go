package client

import (
	"context"
	"errors"
)

type Client interface {
	// Invoke performs a unary RPC.
	Invoke(ctx context.Context, reqBody interface{}, rspBody interface{}, opt ...Option) error
}

var DefaultClient = New()

var New = func() Client {
	return &client{}
}

type client struct{}

func (c *client) Invoke(ctx context.Context, reqBody interface{}, rspBody interface{}, opt ...Option) error {
	
	// 先根据opt，更新options结构体
	opts := DefaultOptions
	for _, o := range opt {
		o(opts)
	}

	// 发送请求
	if opts.ClientTransport == nil {
		return errors.New("clientTransport is nil")
	}
	err := opts.ClientTransport.Send(ctx, reqBody, rspBody, opts.ClientTransportOption)
	if err != nil {
		return err
	}
	return nil

}

