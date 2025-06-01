//go:build windows
// +build windows

package poller

import (
    "net"
    "errors"
    "MyRPC/netx/connection"
)

type EventLoop interface {
    Serve(ln net.Listener) error
}

type eventLoop struct{}

func NewEventLoop(onRequest connection.OnRequest) (EventLoop, error) {
    return &eventLoop{}, nil
}

func (evl *eventLoop) Serve(ln net.Listener) error {
    return errors.New("event loop is not supported on Windows")
}
