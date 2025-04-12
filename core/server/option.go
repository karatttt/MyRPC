package server

import (
	"MyRPC/core/codec"
	"MyRPC/core/transport"
)

type Option func(*Options)

// 服务端的option，在服务结束后生命周期结束
type Options struct {
	ServerName string
	Address    string
	Transport  transport.ServerTransport
	Codec      codec.Codec
}

var DefaultOptions = NewOptions()

func NewOptions() *Options {
	return &Options{
		Transport: transport.DefaultServerTransport,
		Codec:     codec.DefaultServerCodec,
	}
}

func WithTransport(transport transport.ServerTransport) Option {
	return func(o *Options) {
		o.Transport = transport
	}
}

// WithAddress sets the server address
func WithAddress(address string) Option {
	return func(o *Options) {
		o.Address = address
	}
}
