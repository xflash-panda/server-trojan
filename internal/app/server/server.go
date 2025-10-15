package server

import (
	"context"
	"fmt"
	"os"
	"sync"

	api "github.com/xflash-panda/server-client/pkg"
	_ "github.com/xflash-panda/server-trojan/internal/pkg/dep"
	"github.com/xflash-panda/server-trojan/internal/pkg/dispatcher"
	"github.com/xflash-panda/server-trojan/internal/pkg/service"

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
	extFileBytes  []byte
	ctx           context.Context
	registerId    int
	apiClient     *api.Client
}

func New(config *Config, apiConfig *api.Config, serviceConfig *service.Config, extFileBytes []byte) (*Server, error) {
	// 创建全局context
	ctx := context.Background()
	return &Server{config: config, apiConfig: apiConfig, serviceConfig: serviceConfig, extFileBytes: extFileBytes, ctx: ctx}, nil
}

func (s *Server) Start() error {
	s.access.Lock()
	defer s.access.Unlock()
	log.Infoln("server start")

	apiClient := api.New(s.apiConfig)
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %s", err)
	}
	registerId, nodeConf, err := apiClient.Register(api.NodeId(s.serviceConfig.NodeID), api.Trojan,
		hostname, s.serviceConfig.ServerPort, "")
	if err != nil {
		return fmt.Errorf("failed to get node inf :%s", err)
	}
	log.Infof("Registered with server, registerId: %d", registerId)

	trojanConfig, ok := nodeConf.(*api.TrojanConfig)
	if !ok {
		return fmt.Errorf("nodeConf type assertion to *api.TrojanConfig failed")
	}
	inBoundConfig, err := service.InboundBuilder(s.serviceConfig, trojanConfig)
	if err != nil {
		return fmt.Errorf("failed to build inbound config: %s", err)
	}

	outBoundConfig, err := service.OutboundBuilder(s.ctx, trojanConfig, s.extFileBytes)
	if err != nil {
		return fmt.Errorf("failed to build outbound config: %s", err)
	}

	instance, err := s.loadCore(s.ctx, inBoundConfig, outBoundConfig)
	if err != nil {
		return fmt.Errorf("failed to load core: %s", err)
	}

	if err := instance.Start(); err != nil {
		return fmt.Errorf("failed to start instance: %s", err)
	}

	buildService := service.New(inBoundConfig.Tag, instance, s.serviceConfig, trojanConfig, registerId, apiClient)
	s.service = buildService
	if err := s.service.Start(); err != nil {
		return fmt.Errorf("failed to start build service: %s", err)
	}
	s.Running = true
	s.instance = instance
	s.registerId = registerId
	s.apiClient = apiClient
	log.Infoln("server is running")
	return nil
}

func (s *Server) loadCore(ctx context.Context, inboundConfig *core.InboundHandlerConfig, outboundConfig *core.OutboundHandlerConfig) (*core.Instance, error) {
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
	// 使用传入的context替代s.ctx
	instance, err := core.NewWithContext(ctx, pbCoreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %s", err)
	}
	return instance, nil
}

func (s *Server) Close() {
	s.access.Lock()
	defer s.access.Unlock()

	// 先取消注册
	if s.apiClient != nil && s.registerId > 0 {
		log.Infof("unregistering node, registerId: %d", s.registerId)
		err := s.apiClient.Unregister(api.Trojan, s.registerId)
		if err != nil {
			log.Errorf("failed to unregister: %s", err)
		} else {
			log.Infoln("unregister success")
		}
	}

	// 关闭服务
	if s.service != nil {
		err := s.service.Close()
		if err != nil {
			log.Errorf("server close failed: %s", err)
		}
	}
	log.Infoln("server close")
}
