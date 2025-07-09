package server

import (
	"fmt"
	"time"
)

const (
	LogLevelDebug = "debug"
	LogLevelError = "error"
	LogLevelInfo  = "info"
)

type ConnectionConfig struct {
	Handshake    uint32 `mapstructure:"handshake"`
	ConnIdle     uint32 `mapstructure:"connIdle"`
	UplinkOnly   uint32 `mapstructure:"uplinkOnly"`
	DownlinkOnly uint32 `mapstructure:"downlinkOnly"`
	BufferSize   int32  `mapstructure:"bufferSize"`
}

var defaultConnectionConfig *ConnectionConfig = &ConnectionConfig{
	Handshake:    4,
	ConnIdle:     30,
	UplinkOnly:   2,
	DownlinkOnly: 4,
	BufferSize:   64,
}

type ExtConfig struct {
	Outbounds []serverConfigOutboundEntry `mapstructure:"outbounds"`
	ACL       serverConfigACL             `mapstructure:"acl"`
	Sniff     serverConfigSniff           `mapstructure:"sniff"`
}

type serverConfigACL struct {
	Inline []string `mapstructure:"inline"`
}

type serverConfigOutboundEntry struct {
	Name   string                     `mapstructure:"name"`
	Type   string                     `mapstructure:"type"`
	Direct serverConfigOutboundDirect `mapstructure:"direct"`
	SOCKS5 serverConfigOutboundSOCKS5 `mapstructure:"socks5"`
	HTTP   serverConfigOutboundHTTP   `mapstructure:"http"`
}

type serverConfigSniff struct {
	Enable        bool          `mapstructure:"enable"`
	Timeout       time.Duration `mapstructure:"timeout"`
	RewriteDomain bool          `mapstructure:"rewriteDomain"`
	TCPPorts      string        `mapstructure:"tcpPorts"`
	UDPPorts      string        `mapstructure:"udpPorts"`
}

type serverConfigOutboundDirect struct {
	Mode       string `mapstructure:"mode"`
	BindIPv4   string `mapstructure:"bindIPv4"`
	BindIPv6   string `mapstructure:"bindIPv6"`
	BindDevice string `mapstructure:"bindDevice"`
}

type serverConfigOutboundSOCKS5 struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type serverConfigOutboundHTTP struct {
	URL      string `mapstructure:"url"`
	Insecure bool   `mapstructure:"insecure"`
}

// configError 表示配置错误
type configError struct {
	Field string
	Err   error
}

// Error 实现 error 接口
func (e configError) Error() string {
	return fmt.Sprintf("config error in field %s: %v", e.Field, e.Err)
}

// Unwrap 返回原始错误
func (e configError) Unwrap() error {
	return e.Err
}
