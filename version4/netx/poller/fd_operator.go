//go:build linux
// +build linux

package poller

import (
	"MyRPC/netx/buffer"
	"sync/atomic"
	"net"
)
const (
	ListenerType = iota // 0: listener
	ConnectionType        // 1: connection
)

// FDOperator is a collection of operations on file descriptors.
type FDOperator struct {
	 
	Conn net.Conn // Conn is the connection associated with the file descriptor.
	// FD is file descriptor, poll will bind when register.
	FD int

	Type int // listener or connection, used to distinguish the type of file descriptor.

	// The FDOperator provides three operations of reading, writing, and hanging.
	// The poll actively fire the FDOperator when fd changes, no check the return value of FDOperator.
	OnRead  func(conn net.Conn, ) error
	OnWrite func(op *FDOperator) error
	OnHup   func() error

	// The following is the required fn, which must exist when used, or directly panic.
	// Fns are only called by the poll when handles connection events.
	Input   *buffer.Buffer // Input is the input buffer, which is used to read data from the file descriptor.
	InputAck func(n int) (err error)

	// Outputs will locked if len(rs) > 0, which need unlocked by OutputAck.
	Output   *buffer.Buffer 
	OutputAck func(n int) (err error)

	// poll is the registered location of the file descriptor.
	poll Poll

	// protect only detach once
	detached int32

	// private, used by operatorCache
	next  *FDOperator
	state int32 // CAS: 0(unused) 1(inuse) 2(do-done)
	index int32 // index in operatorCache
}

func (op *FDOperator) Control(event PollEvent) error {
	if event == PollDetach && atomic.AddInt32(&op.detached, 1) > 1 {
		return nil
	}
	return op.poll.Control(op, event)
}