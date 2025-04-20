package client

import (
	"MyRPC/core/transport"
	"MyRPC/core/codec"
)
// 赋值options
type Option func(*Options)

// 客户端的option，在请求发送出去后生命周期结束
type Options struct {
	target string
	// 这个里面的配置实际上和该options相似，但是如果直接在transport中复用这里的Option去使用其属性时，会导致循环依赖
	// 当然也可以在send之前显示的new一个这样的结构，再拷贝
	ClientTransportOption *transport.ClientTransportOption
	// 这里使用接口，方便自定义transport
	ClientTransport transport.ClientTransport
	// 编码器
	Codec codec.Codec
}

var DefaultOptions = NewOptions()

func NewOptions() *Options {
	
	return &Options{
		ClientTransportOption: &transport.ClientTransportOption{
			Codec: codec.DefaultClientCodec,
		},
		ClientTransport: transport.DefaultClientTransport,
		Codec: codec.DefaultClientCodec,
	}
}


// WithTarget 设置目标地址
func WithTarget(target string) Option {
	return func(o *Options) {
		o.target = target
		o.ClientTransportOption.Address = target
	}
	
}

// WithCodec 设置编码器
func WithCodec(codec codec.Codec) Option {
	return func(o *Options) {
		o.Codec = codec
		o.ClientTransportOption.Codec = codec
	}
}

