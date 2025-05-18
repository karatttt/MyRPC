package transport

import(
	"MyRPC/core/codec"
	"time"
)

type ClientTransportOption struct{
	Address string
	Codec codec.Codec
	// 异步模式下，超过这个时间就会触发重试
	Timeout time.Duration
	// 重试次数（异步模式）
	RetryTimes int
}