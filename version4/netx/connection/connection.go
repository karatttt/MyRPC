
package connection
import (
	"net"
	"time"
	"MyRPC/netx/linkBuffer"
	"context"
)

type Connection interface {
	// Connection extends net.Conn, just for interface compatibility.
	// It's not recommended to use net.Conn API except for io.Closer.
	net.Conn

	// The recommended API for nocopy reading and writing.
	// Reader will return nocopy buffer data, or error after timeout which set by SetReadTimeout.
	Reader() linkBuffer.Reader
	// Writer will write data to the connection by NIO mode,
	// so it will return an error only when the connection isn't Active.
	Writer() linkBuffer.Writer

	// IsActive checks whether the connection is active or not.
	IsActive() bool

	// SetReadTimeout sets the timeout for future Read calls wait.
	// A zero value for timeout means Reader will not timeout.
	SetReadTimeout(timeout time.Duration) error

	// SetWriteTimeout sets the timeout for future Write calls wait.
	// A zero value for timeout means Writer will not timeout.
	SetWriteTimeout(timeout time.Duration) error

	// SetIdleTimeout sets the idle timeout of connections.
	// Idle connections that exceed the set timeout are no longer guaranteed to be active,
	// but can be checked by calling IsActive.
	SetIdleTimeout(timeout time.Duration) error

	// SetOnRequest can set or replace the OnRequest method for a connection, but can't be set to nil.
	// Although SetOnRequest avoids data race, it should still be used before transmitting data.
	// Replacing OnRequest while processing data may cause unexpected behavior and results.
	// Generally, the server side should uniformly set the OnRequest method for each connection via NewEventLoop,
	// which is set when the connection is initialized.
	// On the client side, if necessary, make sure that OnRequest is set before sending data.
	SetOnRequest(on OnRequest) error

	// // AddCloseCallback can add hangup callback for a connection, which will be called when connection closing.
	// // This is very useful for cleaning up idle connections. For instance, you can use callbacks to clean up
	// // the local resources, which bound to the idle connection, when hangup by the peer. No need another goroutine
	// // to polling check connection status.
	// AddCloseCallback(callback CloseCallback) error
}


type OnRequest func(ctx context.Context, connection Connection) error