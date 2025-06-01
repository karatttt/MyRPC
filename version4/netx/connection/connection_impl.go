package connection

import (
	"MyRPC/netx/linkBuffer"
	"fmt"
	"net"
	"time"
)

// connection is the implement of Connection
type connection struct {
	rawConn      net.Conn // The underlying net.Conn
	inputBuffer  *linkBuffer.Buffer
	outputBuffer *linkBuffer.Buffer

	maxSize      int           // The maximum size of data between two Release().
	bookSize     int           // The size of data that can be read at once.
	isActive     bool          // Whether the connection is active or not.
	readTimeout  time.Duration // The timeout for future Read calls.
	writeTimeout time.Duration // The timeout for future Write calls.
	idleTimeout  time.Duration // The idle timeout of connections.
	onRequest    OnRequest     // The method to handle requests.
}

// init initializes the connection with options
func InitConn(conn net.Conn) Connection {
	c := &connection{
		rawConn:  conn,
		maxSize:  1024 * 1024, // Default max size is 1MB
		bookSize: 4096,        // Default book size is 4KB
	}
	c.inputBuffer = linkBuffer.NewBuffer()
	c.outputBuffer = linkBuffer.NewBuffer()

	// Set the connection as active
	c.isActive = true

	c.outputBuffer.Start(conn)

	// Set default timeouts
	c.readTimeout = 0
	c.writeTimeout = 0
	c.idleTimeout = 0

	return c
}

func (c *connection) Read(b []byte) (n int, err error) {
	return c.rawConn.Read(b)
}

func (c *connection) Write(b []byte) (n int, err error) {
	_, writeErr := c.outputBuffer.Write(b)
	if writeErr != nil {
		return 0, writeErr
	}
	return 0, nil
}

func (c *connection) Close() error {
	return c.rawConn.Close()
}

func (c *connection) LocalAddr() net.Addr {
	return c.rawConn.LocalAddr()
}

func (c *connection) RemoteAddr() net.Addr {
	return c.rawConn.RemoteAddr()
}

func (c *connection) SetDeadline(t time.Time) error {
	return c.rawConn.SetDeadline(t)
}

func (c *connection) SetReadDeadline(t time.Time) error {
	return c.rawConn.SetReadDeadline(t)
}

func (c *connection) SetWriteDeadline(t time.Time) error {
	return c.rawConn.SetWriteDeadline(t)
}

func (c *connection) SetOnRequest(on OnRequest) error {
	if on == nil {
		return fmt.Errorf("OnRequest handler cannot be nil")
	}
	c.onRequest = on
	return nil
}