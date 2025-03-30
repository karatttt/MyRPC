package main

import (
	 "MyRPC/pb"
	 "MyRPC/core/server"
)

func main() {
	// Create a new server instance
	s := server.NewServer()

	// Register the HelloService with the server
	pb.RegisterHelloServer(s, &HelloServerImpl{})

	// Start the server on port 50051
	if err := s.Serve(":50051"); err != nil {
		panic(err)
	}
}


// 创建一个HelloServer的实现类
type HelloServerImpl struct{}
// 实现HelloServer接口的Hello方法
func (h *HelloServerImpl) Hello(req *pb.HelloRequest) (*pb.HelloReply, error) {
	// 这里可以实现具体的业务逻辑
	reply := &pb.HelloReply{
		Msg: "Hello " + req.Msg,
	}
	return reply, nil
}