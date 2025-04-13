package main

import (
	"MyRPC/core/client"
	"MyRPC/pb"
	"context"
	"fmt"
)

func main() {
	c := pb.NewHelloClientProxy(client.WithTarget("127.0.0.1:8000"))
	if c == nil {
		fmt.Println("Failed to create client")
		return
	}

	rsp, err := c.Hello(context.Background(), &pb.HelloRequest{Msg: "world"})
	if err != nil {
		fmt.Println("RPC call error:", err)
		return
	}
	fmt.Println("Response:", rsp.Msg)
}
