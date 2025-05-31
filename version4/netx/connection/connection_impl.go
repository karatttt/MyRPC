package connection

import(
	"net"
	"time"
	"MyRPC/netx/linkBuffer"
)


// connection is the implement of Connection
type connection struct {
	rawConn  	net.Conn // The underlying net.Conn
	inputBuffer     *linkBuffer.Buffer
	outputBuffer    *linkBuffer.Buffer

	maxSize         int       // The maximum size of data between two Release().
	bookSize        int       // The size of data that can be read at once.
	isActive        bool      // Whether the connection is active or not.
	readTimeout     time.Duration // The timeout for future Read calls.
	writeTimeout    time.Duration // The timeout for future Write calls.
	idleTimeout     time.Duration // The idle timeout of connections.
	onRequest       OnRequest // The method to handle requests.
}


// init initializes the connection with options
func InitConn(conn net.Conn, onRequest OnRequest) (inputBuffer *linkBuffer.Buffer, outputBuffer *linkBuffer.Buffer) {
	c := &connection{
		rawConn:   conn,
		maxSize:  1024 * 1024, // Default max size is 1MB
		bookSize:  4096,       // Default book size is 4KB
	}
	c.inputBuffer = linkBuffer.NewBuffer()
	c.outputBuffer = linkBuffer.NewBuffer()

	// Set the connection as active
	c.isActive = true

	// Set the OnRequest method
	if onRequest != nil {
		c.onRequest = onRequest
	} else {
		return nil, nil
	}

	c.outputBuffer.Start(conn)

	// Set default timeouts
	c.readTimeout = 0
	c.writeTimeout = 0
	c.idleTimeout = 0

	return nil, nil
}
