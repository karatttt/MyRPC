// 我们尝试自己写一个HelloWorld的客户端和服务端

package pb

import (
	"MyRPC/common"
	"MyRPC/core/client"
	"MyRPC/core/codec"
	"MyRPC/core/internel"
	"MyRPC/core/server"
	"context"
	"fmt"
)

// 具体方法接口
type HelloServer interface {
	// SayHello is the method that will be called by the client.
	Hello(req *HelloRequest) (*HelloReply, error)
}

// HelloServer_Hello_Handler is the handler for the Hello method.
func HelloServer_Hello_Handler(srv interface{}, req []byte) (interface{}, error) {

	// 这里的srv是HelloServer的实现类
	// 通过类型断言将srv转换为HelloServer类型
	helloServer, ok := srv.(HelloServer)
	if !ok {
		return nil, fmt.Errorf("HelloServer_Hello_Handler: %v", "type assertion failed")
	}
	// 调用HelloServer的Hello方法
	// 将req反序列化
	reqBody := &HelloRequest{}
	err := codec.Unmarshal(req, reqBody)
	if err != nil {
		return nil, fmt.Errorf("HelloServer_Hello_Handler: %v", err)
	}
	reply, err := helloServer.Hello(reqBody)

	if err != nil {
		fmt.Printf("HelloServer_Hello_Handler: %v", err)
		return nil, err
	}
	return reply, nil

}

var HelloServer_ServiceDesc = server.ServiceDesc{
	ServiceName: "helloworld",
	HandlerType: (*HelloServer)(nil),
	Methods: []server.MethodDesc{
		{
			MethodName: "Hello",
			// 当接受到客户端调用的Hello方法时，server将会调用这个方法
			Func: HelloServer_Hello_Handler,
		},
	},
}

// 绑定方法
func RegisterHelloServer(s *server.Server, svr interface{}) error {

	if err := s.Register(&HelloServer_ServiceDesc, svr); err != nil {
		panic(fmt.Sprintf("Greeter register error:%v", err))
	}
	return nil
}

// 客户端代理接口
type HelloClientProxy interface {
	Hello(ctx context.Context, in *HelloRequest, opts ...client.Option) (*HelloReply, *common.RPCError)
}
type HelloAsyncClientProxy interface {
	HelloAsync(ctx context.Context, in *HelloRequest, opts ...client.Option) (*internel.Future, *common.RPCError)
}


// 客户端代理实现
type HelloClientProxyImpl struct {
	client client.Client
	opts   []client.Option
}

type HelloAsyncClientProxyImpl struct {
	client client.Client
	opts   []client.Option
}



// 创建客户端代理
func NewHelloClientProxy(opts ...client.Option) HelloClientProxy {
	return &HelloClientProxyImpl{
		client: client.DefaultClient,
		opts:   opts,
	}
}

// 创建客户端代理
func NewHelloAsyncClientProxy(opts ...client.Option) HelloAsyncClientProxy {
	return &HelloAsyncClientProxyImpl{
		client: client.DefaultClient,
		opts:   opts,
	}
}

// 实现Hello方法
func (c *HelloClientProxyImpl) Hello(ctx context.Context, req *HelloRequest, opts ...client.Option) (*HelloReply, *common.RPCError) {
	// 创建一个msg结构，存储service相关的数据，如serviceName等，并放到context中
	// 用msg结构可以避免在context中太多withValue传递过多的参数
	msg := internel.NewMsg()
	msg.WithServiceName("helloworld")
	msg.WithMethodName("Hello")
	ctx = context.WithValue(ctx, internel.ContextMsgKey, msg)

	rsp := &HelloReply{}
	// 这里需要将opts添加前面newProxy时传入的opts
	newOpts := append(c.opts, opts...)
	err := c.client.Invoke(ctx, req, rsp, newOpts...)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

// 实现HelloAsync方法
func (c *HelloAsyncClientProxyImpl) HelloAsync(ctx context.Context, req *HelloRequest, opts ...client.Option) (*internel.Future, *common.RPCError) {
	msg := internel.NewMsg()
	msg.WithServiceName("helloworld")
	msg.WithMethodName("Hello")
	ctx = context.WithValue(ctx, internel.ContextMsgKey, msg)

	rsp := &HelloReply{}
	// 这里需要将opts添加前面newProxy时传入的opts
	newOpts := append(c.opts, opts...)
	return c.client.InvokeAsync(ctx, req, rsp, newOpts...)
}


