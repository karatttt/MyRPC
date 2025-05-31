//go:build linux
// +build linux

package poller

import (
	"syscall"
	"unsafe"
)

// 事件类型宏定义
const (
	PollReadable = 1 << 0 // 可读
	PollWritable = 1 << 1 // 可写
	PollHup      = 1 << 2 // 挂起/关闭
	PollDetach   = 1 << 3 // 注销
	PollR2RW     = 1 << 4 // 读转读写
	PollRW2R     = 1 << 5 // 读写转读
)

const EPOLLET = 1 << 31 // 兼容 Windows 下开发，Linux 下会被系统常量覆盖

type PollEvent = int

type Poll interface{
	// Control controls the file descriptor operator with the specified event.
	Control(operator *FDOperator, event PollEvent) error
	
	Wait() error // Wait blocks until an event occurs on the file descriptor.
}

type defaultPoll struct {
	fd        int
	operators map[int]*FDOperator
}

func (p *defaultPoll) setOperator(ptr unsafe.Pointer, op *FDOperator) {
	if p.operators == nil {
		p.operators = make(map[int]*FDOperator)
	}
	p.operators[op.FD] = op
}

func (p *defaultPoll) delOperator(op *FDOperator) {
	delete(p.operators, op.FD)
}

func EpollCtl(epfd, op, fd int, event *syscall.EpollEvent) error {
	return syscall.EpollCtl(epfd, op, fd, event)
}

// Control implements Poll.
func (p *defaultPoll) Control(operator *FDOperator, event PollEvent) error {
	fd := operator.FD
	var op int
	var evt syscall.EpollEvent
	p.setOperator(unsafe.Pointer(&evt.Fd), operator)
	switch event {
	case PollReadable: // server accept a new connection and wait read
		op, evt.Events = syscall.EPOLL_CTL_ADD, syscall.EPOLLIN|syscall.EPOLLRDHUP|syscall.EPOLLERR
	case PollWritable: // client create a new connection and wait connect finished
		op, evt.Events = syscall.EPOLL_CTL_ADD, EPOLLET|syscall.EPOLLOUT|syscall.EPOLLRDHUP|syscall.EPOLLERR
	case PollDetach: // deregister
		p.delOperator(operator)
		op, evt.Events = syscall.EPOLL_CTL_DEL, syscall.EPOLLIN|syscall.EPOLLOUT|syscall.EPOLLRDHUP|syscall.EPOLLERR
	case PollR2RW: // connection wait read/write
		op, evt.Events = syscall.EPOLL_CTL_MOD, syscall.EPOLLIN|syscall.EPOLLOUT|syscall.EPOLLRDHUP|syscall.EPOLLERR
	case PollRW2R: // connection wait read
		op, evt.Events = syscall.EPOLL_CTL_MOD, syscall.EPOLLIN|syscall.EPOLLRDHUP|syscall.EPOLLERR
	}
	evt.Fd = int32(fd)
	return EpollCtl(p.fd, op, fd, &evt)
}
