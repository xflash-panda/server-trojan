package server

import (
	"context"
	"fmt"
	"strings"
	"sync"

	api "github.com/xflash-panda/server-client/pkg"
	_ "github.com/xflash-panda/server-trojan/internal/pkg/dep"
	"github.com/xflash-panda/server-trojan/internal/pkg/dispatcher"
	"github.com/xflash-panda/server-trojan/internal/pkg/service"

	C "github.com/apernet/hysteria/core/v2/server"
	"github.com/apernet/hysteria/extras/v2/outbounds"
	log "github.com/sirupsen/logrus"
	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/app/stats"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

type Config struct {
	LogLevel string
}

type Server struct {
	access        sync.Mutex
	instance      *core.Instance
	service       service.Service
	config        *Config
	apiConfig     *api.Config
	serviceConfig *service.Config
	Running       bool
	pOutbound     C.Outbound
	ctx           context.Context
}

func buildOutboundFromExtConfig(extConfig *ExtConfig) (C.Outbound, error) {
	var obs []outbounds.OutboundEntry
	var uOb outbounds.PluggableOutbound

	if len(extConfig.Outbounds) == 0 {
		obs = []outbounds.OutboundEntry{{
			Name:     "default",
			Outbound: outbounds.NewDirectOutboundSimple(outbounds.DirectOutboundModeAuto),
		}}
	} else {
		obs = make([]outbounds.OutboundEntry, len(extConfig.Outbounds))
		for i, entry := range extConfig.Outbounds {
			if entry.Name == "" {
				return nil, fmt.Errorf("empty outbound name")
			}
			var ob outbounds.PluggableOutbound
			var err error
			switch strings.ToLower(entry.Type) {
			case "direct":
				ob, err = serverConfigOutboundDirectToOutbound(entry.Direct)
			case "socks5":
				ob, err = serverConfigOutboundSOCKS5ToOutbound(entry.SOCKS5)
			case "http":
				ob, err = serverConfigOutboundHTTPToOutbound(entry.HTTP)
			default:
				return nil, fmt.Errorf("unsupported outbound type")
			}
			if err != nil {
				return nil, fmt.Errorf("failed to create outbound: %s", err)
			}
			obs[i] = outbounds.OutboundEntry{Name: entry.Name, Outbound: ob}
		}

	}
	gLoader := &GeoLoader{
		GeoIPFilename:   "",
		GeoSiteFilename: "",
		UpdateInterval:  geoDefaultUpdateInterval,
		DownloadFunc:    geoDownloadFunc,
		DownloadErrFunc: geoDownloadErrFunc,
	}

	if len(extConfig.ACL.Inline) > 0 {
		aclOutbound, err := outbounds.NewACLEngineFromString(strings.Join(extConfig.ACL.Inline, "\n"), obs, gLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to create acl outbound: %s", err)
		}
		uOb = aclOutbound
	} else {
		uOb = obs[0].Outbound
	}

	return &outbounds.PluggableOutboundAdapter{PluggableOutbound: uOb}, nil
}

func New(config *Config, apiConfig *api.Config, serviceConfig *service.Config, extConfig *ExtConfig) (*Server, error) {
	var pOutbound C.Outbound
	var err error
	if extConfig != nil {
		pOutbound, err = buildOutboundFromExtConfig(extConfig)
		if err != nil {
			return nil, err
		}
	} else {
		pOutbound = &outbounds.PluggableOutboundAdapter{PluggableOutbound: outbounds.NewDirectOutboundSimple(outbounds.DirectOutboundModeAuto)}
	}

	// 创建全局context
	ctx := context.Background()
	return &Server{config: config, apiConfig: apiConfig, serviceConfig: serviceConfig, pOutbound: pOutbound, ctx: ctx}, nil
}

func (s *Server) Start() error {
	s.access.Lock()
	defer s.access.Unlock()
	log.Infoln("server start")

	apiClient := api.New(s.apiConfig)
	nodeConf, err := apiClient.Config(api.NodeId(s.serviceConfig.NodeID), api.Trojan)
	if err != nil {
		return fmt.Errorf("failed to get node inf :%s", err)
	}

	trojanConfig, ok := nodeConf.(*api.TrojanConfig)
	if !ok {
		return fmt.Errorf("nodeConf type assertion to *api.TrojanConfig failed")
	}
	inBoundConfig, err := service.InboundBuilder(s.serviceConfig, trojanConfig)
	if err != nil {
		return fmt.Errorf("failed to build inbound config: %s", err)
	}

	outBoundConfig, err := service.OutboundBuilder(s.ctx, trojanConfig, s.pOutbound)
	if err != nil {
		return fmt.Errorf("failed to build outbound config: %s", err)
	}

	instance, err := s.loadCore(inBoundConfig, outBoundConfig)
	if err != nil {
		return fmt.Errorf("failed to load core: %s", err)
	}

	if err := instance.Start(); err != nil {
		return fmt.Errorf("failed to start instance: %s", err)
	}

	buildService := service.New(inBoundConfig.Tag, instance, s.serviceConfig, trojanConfig,
		apiClient.Users, apiClient.Submit)
	s.service = buildService
	if err := s.service.Start(); err != nil {
		return fmt.Errorf("failed to start build service: %s", err)
	}
	s.Running = true
	s.instance = instance
	log.Infoln("server is running")
	return nil
}

func (s *Server) loadCore(inboundConfig *core.InboundHandlerConfig, outboundConfig *core.OutboundHandlerConfig) (*core.Instance, error) {
	// Log Config
	logConfig := &conf.LogConfig{}
	logConfig.LogLevel = s.config.LogLevel
	if s.config.LogLevel != LogLevelDebug {
		logConfig.AccessLog = "none"
		logConfig.ErrorLog = "none"
		logConfig.DNSLog = false
	}
	pbLogConfig := logConfig.Build()

	// InboundConfig
	inboundConfigs := make([]*core.InboundHandlerConfig, 1)
	inboundConfigs[0] = inboundConfig

	// OutBound config
	outBoundConfigs := make([]*core.OutboundHandlerConfig, 1)
	outBoundConfigs[0] = outboundConfig

	// PolicyConfig
	policyConfig := &conf.PolicyConfig{}
	pbPolicy := &conf.Policy{
		StatsUserUplink:   true,
		StatsUserDownlink: true,
		StatsUserOnline:   false,
		Handshake:         &defaultConnectionConfig.Handshake,
		ConnectionIdle:    &defaultConnectionConfig.ConnIdle,
		UplinkOnly:        &defaultConnectionConfig.UplinkOnly,
		DownlinkOnly:      &defaultConnectionConfig.DownlinkOnly,
		BufferSize:        &defaultConnectionConfig.BufferSize,
	}
	policyConfig.Levels = map[uint32]*conf.Policy{0: pbPolicy}
	pbPolicyConfig, _ := policyConfig.Build()
	pbCoreConfig := &core.Config{
		App: []*serial.TypedMessage{
			serial.ToTypedMessage(pbLogConfig),
			serial.ToTypedMessage(pbPolicyConfig),
			serial.ToTypedMessage(&stats.Config{}),
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		},
		Outbound: outBoundConfigs,
		Inbound:  inboundConfigs,
	}
	// 使用NewWithContext替代New，传入全局context
	instance, err := core.NewWithContext(s.ctx, pbCoreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %s", err)
	}
	return instance, nil
}

func (s *Server) Close() {
	s.access.Lock()
	defer s.access.Unlock()

	err := s.service.Close()
	if err != nil {
		log.Fatalf("server close failed: %s", err)
	}
	log.Infoln("server close")
}
