package server

import (
	"errors"
	"net"
	"strings"

	"github.com/apernet/hysteria/extras/v2/outbounds"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func serverConfigOutboundDirectToOutbound(c serverConfigOutboundDirect) (outbounds.PluggableOutbound, error) {
	var mode outbounds.DirectOutboundMode
	switch strings.ToLower(c.Mode) {
	case "", "auto":
		mode = outbounds.DirectOutboundModeAuto
	case "64":
		mode = outbounds.DirectOutboundMode64
	case "46":
		mode = outbounds.DirectOutboundMode46
	case "6":
		mode = outbounds.DirectOutboundMode6
	case "4":
		mode = outbounds.DirectOutboundMode4
	default:
		return nil, configError{Field: "outbounds.direct.mode", Err: errors.New("unsupported mode")}
	}
	bindIP := len(c.BindIPv4) > 0 || len(c.BindIPv6) > 0
	bindDevice := len(c.BindDevice) > 0
	if bindIP && bindDevice {
		return nil, configError{Field: "outbounds.direct", Err: errors.New("cannot bind both IP and device")}
	}
	if bindIP {
		ip4, ip6 := net.ParseIP(c.BindIPv4), net.ParseIP(c.BindIPv6)
		if len(c.BindIPv4) > 0 && ip4 == nil {
			return nil, configError{Field: "outbounds.direct.bindIPv4", Err: errors.New("invalid IPv4 address")}
		}
		if len(c.BindIPv6) > 0 && ip6 == nil {
			return nil, configError{Field: "outbounds.direct.bindIPv6", Err: errors.New("invalid IPv6 address")}
		}
		return outbounds.NewDirectOutboundBindToIPs(mode, ip4, ip6)
	}
	if bindDevice {
		return outbounds.NewDirectOutboundBindToDevice(mode, c.BindDevice)
	}
	return outbounds.NewDirectOutboundSimple(mode), nil
}

func serverConfigOutboundSOCKS5ToOutbound(c serverConfigOutboundSOCKS5) (outbounds.PluggableOutbound, error) {
	if c.Addr == "" {
		return nil, configError{Field: "outbounds.socks5.addr", Err: errors.New("empty socks5 address")}
	}
	return outbounds.NewSOCKS5Outbound(c.Addr, c.Username, c.Password), nil
}

func serverConfigOutboundHTTPToOutbound(c serverConfigOutboundHTTP) (outbounds.PluggableOutbound, error) {
	if c.URL == "" {
		return nil, configError{Field: "outbounds.http.url", Err: errors.New("empty http address")}
	}
	return outbounds.NewHTTPOutbound(c.URL, c.Insecure)
}

func geoDownloadFunc(filename, url string) {
	log.Info("downloading database", zap.String("filename", filename), zap.String("url", url))
}

func geoDownloadErrFunc(err error) {
	if err != nil {
		log.Error("failed to download database", zap.Error(err))
	}
}
