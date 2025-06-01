package connection

import (
	"net"
)

// 继承 net.Conn 接口，增加 SetOnRequest 方法
type Connection interface {
	// Connection extends net.Conn, just for interface compatibility.
	// It's not recommended to use net.Conn API except for io.Closer.
	net.Conn
	SetOnRequest(on OnRequest) error
}

type OnRequest func(conn net.Conn) error
