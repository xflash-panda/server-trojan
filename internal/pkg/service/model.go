package service

import "time"

const (
	protocol       = "trojan"
	TLS            = "tls"
	TCP            = "tcp"
	WS             = "ws"
	GRPC           = "grpc"
	DefaultTimeout = 15 * time.Second
)

// Service is the interface of all the services running in the panel
type Service interface {
	Start() error
	StartMonitor() error
	Close() error
}

type CertConfig struct {
	CertFile string
	KeyFile  string
}

type FallBackConfig struct {
	Host string
	Port int
}
