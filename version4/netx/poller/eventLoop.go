//go:build linux
// +build linux

package poller

import (
	"MyRPC/netx/connection"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
)

type EventLoop interface {
	Serve(ln net.Listener) error
}

// 实现EventLoop接口
type eventLoop struct {
	sync.Mutex
	stop chan error
	opts *options
	ln   net.Listener
}

// NewEventLoop
func NewEventLoop(onRequest connection.OnRequest) (EventLoop, error) {

	opts := &options{
		onRequest: onRequest,
	}
	return &eventLoop{
		opts: opts,
		stop: make(chan error, 1),
	}, nil
}

// Serve implements EventLoop.
func (evl *eventLoop) Serve(ln net.Listener) error {
	evl.Lock()
	evl.ln = ln
	fd, err := getListenerFD(ln)
	if err != nil {
		return err
	}
	operator := FDOperator{
		FD:     int(fd),
		OnRead: evl.ListenerOnRead,
        Type: ListenerType, // 标记为监听器类型
	}
	operator.poll = pollmanager.Pick()
	err = operator.Control(PollReadable)
	evl.Unlock()

	return err
}

func getListenerFD(ln net.Listener) (fd uintptr, err error) {
	// 以 TCPListener 为例
	tcpLn, ok := ln.(*net.TCPListener)
	if !ok {
		return 0, errors.New("listener is not *net.TCPListener")
	}
	file, err := tcpLn.File()
	if err != nil {
		return 0, err
	}
	resfd := file.Fd()
	syscall.SetNonblock(int(resfd), true) // 设置为非阻塞
	return resfd, nil
}

// 每一个事件循环中一定有listen连接的事件，当事件就绪的时候就调用这个函数
func (evl *eventLoop) ListenerOnRead(conn net.Conn) error {
	conn, err := evl.ln.Accept()
	if err != nil {
		// 非阻塞下 accept 没有新连接时返回
		if ne, ok := err.(net.Error); ok && ne.Temporary() {
			// 临时错误，继续等待
			return nil
		}
		fmt.Println("Accept error:", err)
		return err
	}
	fmt.Printf("Accepted new connection: %s\n", conn.RemoteAddr())
	// 选择 poller
	poller := pollmanager.Pick()
	if poller == nil {
		fmt.Println("No available poller")
		conn.Close()

	}
	rawConn, ok := conn.(syscall.Conn)
	if !ok {
		// 不是 syscall.Conn，不能获取 fd
	}
	var fd int
	sysRawConn, err := rawConn.SyscallConn()
	if err != nil {
		fmt.Println("Error getting syscall connection:", err)
	} else {
		err = sysRawConn.Control(func(f uintptr) {
			fd = int(f)
		})
		if err != nil {
			fmt.Println("Error getting file descriptor:", err)
		}
	}
	// 初始化连接
	OpConn := connection.InitConn(conn)
	fmt.Printf("Initialized connection with FD: %d\n", fd)
	// 创建 FDOperator 并注册到 poller
	newOp := &FDOperator{
        poll  : poller,
		Conn:   OpConn,
		FD:     fd,
		OnRead: evl.opts.onRequest, // 这里传入业务处理函数
        Type: ConnectionType, // 标记为连接类型
	}
	if err := poller.Control(newOp, PollReadable); err != nil {
		fmt.Println("Error registering connection:", err)
		conn.Close()
	}
	fmt.Printf("Registered new connection with FD: %d\n", fd)
	return nil
}


// 1. netpoll现在是客户端自己编解码，然后这样触发Onrequest：
// for {
// 	closedBy = c.status(closing)
// 	// close by user or not processable
// 	if closedBy == user || onRequest == nil || c.Reader().Len() == 0 {
// 		break
// 	}
// 	_ = onRequest(c.ctx, c)
// }

// 试着融入这套框架，也就是，在这个OnRequest之前把数据封装好，也就是，这个buffer是结构体的内存，已经转好了
// 2. InitConn 结合buffer完成

// 3.这个是trpc的conn的Onread，：func tcpOnRead(data any, ioData *iovec.IOData) error {
// data passed from desc to tcpOnRead must be of type *tcpconn.
// 以上的做好还要写好poller的整个流程（包括接受一个Onread，连接可读的时候调用，以及非连接的时候accept，我这里其实可以把这两种都抽象成OnRead），其实直接OnRead中直接读单个readFrame就行了（两个net都有写到缓冲buffer的过程，我不用，因为我依赖poller线程解码自然要拷贝到用户态，无所谓使用了readFrame，实在要优化就是从缓冲区拿到一批再循环ReadFrame，但是这样缓冲区就要做粘包逻辑，不好），还有就是，readFrame实际上也是poller线程在做，直到将单个Frame数据读完（也做了解码反序列化），不用循环是因为会影响其他的事件，而且循环停止的界限也不好控制，但是这里要保证业务的handler必须是异步
// 上面是poller要写的，还要写一个serverTransPort，启动这个net（即以上的所有poll）以及传入连接可读的onRead，即Handler那一套，
// 这个写完了，就开始写write，所以handler最后的写要写到缓冲区，批量发包
