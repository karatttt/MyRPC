package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"MyRPC/core/client"
	"MyRPC/pb"
)

func main() {
	// 创建一个带计时功能的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在单独的 goroutine 中处理信号
	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		cancel() // 取消上下文
	}()

	// 调用两次
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled, exiting...")
			return
		default:
			c := pb.NewHelloClientProxy(client.WithTarget("127.0.0.1:8001"))
			if c == nil {
				fmt.Println("Failed to create client")
				return
			}

			rsp, err := c.Hello(ctx, &pb.HelloRequest{Msg: "world"})
			if err != nil {
				fmt.Println("RPC call error:", err)
				return
			}
			fmt.Println("Response:", rsp.Msg)
		}
	}

	// 等待上下文超时
	<-ctx.Done()
	fmt.Println("Shutting down gracefully...")
}
