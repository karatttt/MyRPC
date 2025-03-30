// 我们尝试自己写一个HelloWorld的客户端和服务端

package pb

import (
	"MyRPC/core/server"
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

	// TODO 这里的req先是一个HelloRequest的实例，实际使用中应该是从客户端传过来的请求
	reply, err := helloServer.Hello(&HelloRequest{
		Msg: "Hello World"})
	
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
				Func:    HelloServer_Hello_Handler,
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