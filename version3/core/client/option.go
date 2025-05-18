package client

import (
	"MyRPC/core/transport"
	"MyRPC/core/codec"
	"time"
)
// 赋值options
type Option func(*Options)

// 客户端的option，在请求发送出去后生命周期结束
type Options struct {
	target string
	// 这个里面的配置实际上和该options相似，但是如果直接在transport中复用这里的Option去使用其属性时，会导致循环依赖
	// 当然也可以在send之前显示的new一个这样的结构，再拷贝
	// 顺序是先创建opt，再创建transportOpt，再with属性
	ClientTransportOption *transport.ClientTransportOption
	// 这里使用接口，方便自定义transport
	ClientTransport transport.ClientTransport
	// 编码器
	Codec codec.Codec

	// 超时时间
	Timeout time.Duration
	// 重试次数（异步模式下，超时或者err进行重试）
	RetryTimes int
}

var DefaultOptions = NewOptions()

func NewOptions() *Options {
	
	return &Options{
		ClientTransportOption: &transport.ClientTransportOption{
			Codec: codec.DefaultClientCodec,
		},
		// transport用默认值，TODO 这里的trans实际上也可以用map映射来复用，避免创建多个实例
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

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
		// 这里的timeout是针对transport的超时时间
		o.ClientTransportOption.Timeout = timeout
	}
}

// WithRetryTimes 设置重试次数
func WithRetryTimes(retryTimes int) Option {
	return func(o *Options) {
		o.RetryTimes = retryTimes
		o.ClientTransportOption.RetryTimes = retryTimes
	}
}