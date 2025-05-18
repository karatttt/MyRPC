package mutilpath

import (
	"MyRPC/core/codec"
	"container/list"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)
const (
    minFrameHeaderSize = 16 // FrameHeader的固定长度
)


// 实现net.Conn接口的结构体，保证适配连接池的get和put
// 实际上也是一个连接，只是多了reqID从而可以派生出多个流，区分达到多路复用的目的
type MuxConn struct {
	conn        net.Conn // 原始连接
	pending     map[uint64]*pendingRequest // 每一个reqID（流）对应的等待通道
	buffer      *frameBuffer // 总的frame的buffer
	closeChan   chan struct{}
	readerDone  chan struct{}
	writeLock   sync.Mutex
	reqIDCounter uint64 // 分配递增的请求ID
	mu          sync.RWMutex
}

// 对实际的帧数据做了一个封装，方便处理
type MuxFrame struct {
	data []byte
}

type pendingRequest struct {
	ch      chan MuxFrame
	timeout time.Time
}



func NewMuxConn(conn net.Conn, bufferSize int) *MuxConn {
	mc := &MuxConn{
		conn:       conn,
		pending:    make(map[uint64]*pendingRequest),
		buffer:     newFrameBuffer(bufferSize),
		closeChan:  make(chan struct{}),
		readerDone: make(chan struct{}),
	}
	go mc.readLoop()
	return mc
}

func newFrameBuffer(maxSize int) *frameBuffer {
	return &frameBuffer{
		frames: make(map[uint64]*list.List),
		maxLen: maxSize,
	}
}

func (mc *MuxConn) NextRequestID() uint64 {
	return atomic.AddUint64(&mc.reqIDCounter, 1)
}

func (mc *MuxConn) readLoop() {
	defer close(mc.readerDone)

	for {
		select {
		case <-mc.closeChan:
			return
		default:
		}

		frame, err := codec.ReadFrame(mc.conn)
		if err != nil {
			// 协议错误处理
			fmt.Println("读取帧错误：", err)
			break
		}
		mc.dispatchFrame(frame)
	}
}


func (mc *MuxConn) dispatchFrame(frame []byte) {
	mc.mu.RLock()
	// 截取流序号
	sequenceID := binary.BigEndian.Uint32(frame[4:8])
	pr, exists := mc.pending[uint64(sequenceID)]
	mc.mu.RUnlock()

	frameStruct := MuxFrame{
		data: frame,
	}
	if exists {
		select {
		case pr.ch <- frameStruct:
			// 成功发送到等待通道
		default:
			// 通道已满，丢弃帧
			fmt.Println("丢弃帧 %s：通道已满", frame)
		}
	} else {
		// 存储未匹配帧
		if err := mc.buffer.store(frameStruct, uint64(sequenceID)); err != nil {
			// 缓冲区已满，记录日志
		}
	}
}



func (w *MuxConn) Read(b []byte) (n int, err error) {
	return w.conn.Read(b)
}

func (w *MuxConn) Write(b []byte) (n int, err error) {
	return w.conn.Write(b)
}

func (mc *MuxConn) Close() error {
	close(mc.closeChan)
	err := mc.conn.Close()
	<-mc.readerDone
	return err
}

func (w *MuxConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *MuxConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *MuxConn) SetDeadline(t time.Time) error {
	return w.conn.SetDeadline(t)
}

func (w *MuxConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *MuxConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}