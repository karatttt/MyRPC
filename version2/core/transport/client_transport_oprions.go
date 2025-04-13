package transport

import(
	"MyRPC/core/codec"
)

type ClientTransportOption struct{
	Address string
	Codec codec.Codec
}