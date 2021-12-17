package service

import (
	"encoding/json"
	"fmt"
	"github.com/xflash-panda/server-trojan/internal/pkg/api"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

//InboundBuilder build Inbound config for different protocol
func InboundBuilder(config *Config, nodeInfo *api.NodeInfo) (*core.InboundHandlerConfig, error) {
	var (
		streamSetting *conf.StreamConfig
		jsonSetting   json.RawMessage
	)

	inboundDetourConfig := &conf.InboundDetourConfig{}
	// Build Port
	portList := &conf.PortList{
		Range: []conf.PortRange{{From: uint32(nodeInfo.ServerPort), To: uint32(nodeInfo.ServerPort)}},
	}
	inboundDetourConfig.PortList = portList

	// Build Tag
	inboundDetourConfig.Tag = fmt.Sprintf("%s_%d", protocol, nodeInfo.ServerPort)
	// SniffingConfig
	sniffingConfig := &conf.SniffingConfig{
		Enabled: false,
	}
	inboundDetourConfig.SniffingConfig = sniffingConfig

	var setting *conf.TrojanServerConfig
	setting = &conf.TrojanServerConfig{}

	jsonSetting, err := json.Marshal(setting)
	if err != nil {
		return nil, fmt.Errorf("marshal setting %s config fialed: %s", protocol, err)
	}

	// Build streamSettings
	streamSetting = new(conf.StreamConfig)
	transportProtocol := conf.TransportProtocol(TCP)
	_, err = transportProtocol.Build()
	if err != nil {
		return nil, fmt.Errorf("convert TransportProtocol failed: %s", err)
	}

	streamSetting.Network = &transportProtocol
	streamSetting.Security = TLS
	tlsSettings := &conf.TLSConfig{
		ServerName: nodeInfo.ServerName,
		Insecure:   nodeInfo.AllowInsecure != 0,
	}

	if config.Cert.CertFile != "" && config.Cert.KeyFile != "" {
		certFile, keyFile, err := getCertFile(config.Cert)
		if err != nil {
			return nil, err
		}
		tlsSettings.Certs = append(tlsSettings.Certs, &conf.TLSCertConfig{CertFile: certFile, KeyFile: keyFile, OcspStapling: 3600})
	}

	streamSetting.TLSSettings = tlsSettings
	inboundDetourConfig.Protocol = protocol
	inboundDetourConfig.StreamSetting = streamSetting
	inboundDetourConfig.Settings = &jsonSetting
	return inboundDetourConfig.Build()
}

//getCertFile
func getCertFile(certConfig *CertConfig) (certFile string, keyFile string, err error) {
	if certConfig.CertFile == "" || certConfig.KeyFile == "" {
		return "", "", fmt.Errorf("cert file path or key file path not exist")
	}
	return certConfig.CertFile, certConfig.KeyFile, nil
}
