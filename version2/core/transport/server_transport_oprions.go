package transport

import(
	"time"
)

type ServerTransportOption struct{
	KeepAlivePeriod time.Duration
	IdleTimeout time.Duration
}