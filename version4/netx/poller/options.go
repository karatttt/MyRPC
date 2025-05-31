
package poller

import (
	"MyRPC/netx/connection"
)
// Option .
type Option struct {
	f func(*options)
}

type options struct {
	onRequest    connection.OnRequest
}
