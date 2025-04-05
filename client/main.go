package main

import (
	"context"
	"MyRPC/pb"
	"MyRPC/core/client"
	"fmt"
)

func main() {
	c := pb.NewHelloClientProxy(client.WithTarget("ip://127.0.0.1:8000"))
	rsp, err := c.Hello(context.Background(), &pb.HelloRequest{Msg: "world"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(rsp.Msg)
}
