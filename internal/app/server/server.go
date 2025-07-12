package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/xflash-panda/server-agent-proto/pkg"
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
	LogLevel  string
	AgentHost string
	AgentPort int
}

type Server struct {
	access        sync.Mutex
	instance      *core.Instance
	service       service.Service
	config        *Config
	serviceConfig *service.Config
	Running       bool
	extFileBytes  []byte
	ctx           context.Context
}

func New(config *Config, serviceConfig *service.Config, extFileBytes []byte) (*Server, error) {
	ctx := context.Background()
	return &Server{config: config, serviceConfig: serviceConfig, extFileBytes: extFileBytes, ctx: ctx}, nil
}

func (s *Server) Start(agentClient pb.AgentClient) error {
	s.access.Lock()
	defer s.access.Unlock()
	log.Infoln("server start")
	ctx, cancel := context.WithTimeout(context.Background(), service.DefaultTimeout)
	defer cancel()

	r, err := agentClient.Config(ctx, &pb.ConfigRequest{Params: &pb.CommonParams{NodeId: int32(s.serviceConfig.NodeID), NodeType: pb.NodeType_TROJAN}})
	if err != nil {
		return fmt.Errorf("get config eror: %v", err)
	}

	trojanConfig, err := api.UnmarshalTrojanConfig(r.GetRawData())
	if err != nil {
		return fmt.Errorf("failed to unmarshal trojan config: %s", err)
	}

	inBoundConfig, err := service.InboundBuilder(s.serviceConfig, trojanConfig)
	if err != nil {
		return fmt.Errorf("failed to build inbound config: %s", err)
	}

	outBoundConfig, err := service.OutboundBuilder(ctx, trojanConfig, s.extFileBytes)
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

	buildService := service.New(inBoundConfig.Tag, instance, s.serviceConfig, trojanConfig, agentClient)
	s.service = buildService
	if err := s.service.Start(); err != nil {
		return fmt.Errorf("failed to start build service: %s", err)
	}
	s.Running = true
	s.instance = instance
	log.Infoln("server is running")
	time.Sleep(1 * time.Minute)
	if err := s.service.StartMonitor(); err != nil {
		return fmt.Errorf("failed to start service monitor: %s", err)
	}
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

	// DNS Config
	dnsConfig := &conf.DNSConfig{}
	pbDnsConfig, _ := dnsConfig.Build()

	// Routing config
	routerConfig := &conf.RouterConfig{}
	pbRouteConfig, _ := routerConfig.Build()

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
			serial.ToTypedMessage(pbDnsConfig),
			serial.ToTypedMessage(&stats.Config{}),
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
			serial.ToTypedMessage(pbRouteConfig),
		},
		Outbound: outBoundConfigs,
		Inbound:  inboundConfigs,
	}
	instance, err := core.NewWithContext(ctx, pbCoreConfig)
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
