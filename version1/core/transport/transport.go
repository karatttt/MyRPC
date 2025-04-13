package transport

import "context"

// ------SERVER
type ServerTransport interface {
	ListenAndServe(ctx context.Context, network, address string) error
	RegisterHandler(handler Handler)
}

// 定义一个Handler接口，service实现了这个接口
type Handler interface {
	Handle(ctx context.Context, frame []byte) (rsp []byte, err error)
}


// ------CLIENT
type ClientTransport interface {
	Send(ctx context.Context, reqBody interface{}, rspBody interface{}, opt *ClientTransportOption) error
}


