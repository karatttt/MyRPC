package main

import (
	"MyRPC/core/client"
	"MyRPC/pb"
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	success int64
	wg      sync.WaitGroup
)

func main() {
	// 启动 pprof 服务（用于可视化内存分析）
	go func() {
		fmt.Println("[pprof] listening on :6060")
		_ = http.ListenAndServe(":6060", nil)
	}()

	// 创建 RPC 客户端
	c := pb.NewHelloClientProxy(client.WithTarget("127.0.0.1:8001"))
	if c == nil {
		fmt.Println("Failed to create client")
		return
	}

	printMemStats("Before requests")

	const N = 10000
	start := time.Now()
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			

			rsp, err := c.Hello(context.Background(), &pb.HelloRequest{Msg: "world"})
			if err == nil && rsp != nil {
				atomic.AddInt64(&success, 1)
			} else {
				fmt.Printf("Request %d error: %v\n", i, err)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	printMemStats("After requests")

	fmt.Println("\n------ Benchmark Summary ------")
	fmt.Printf("Total requests: %d\n", N)
	fmt.Printf("Success count:  %d\n", success)
	fmt.Printf("Total time:     %v\n", elapsed)
	fmt.Printf("Avg per call:   %v\n", elapsed/time.Duration(N))

	// 休眠3s
	time.Sleep(3 * time.Second)
}

// 打印内存状态
func printMemStats(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\n=== %s ===\n", label)
	fmt.Printf("Alloc = %v KB\n", m.Alloc/1024)
	fmt.Printf("TotalAlloc = %v KB\n", m.TotalAlloc/1024)
	fmt.Printf("Sys = %v KB\n", m.Sys/1024)
	fmt.Printf("NumGC = %v\n", m.NumGC)
}
