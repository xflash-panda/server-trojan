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
	registerId    int32
	agentClient   pb.AgentClient
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

	r, err := agentClient.Config(ctx, &pb.ConfigRequest{NodeId: int32(s.serviceConfig.NodeID), NodeType: pb.NodeType_TROJAN})
	if err != nil {
		return fmt.Errorf("fetch trojan config from agent failed: %w", err)
	}

	trojanConfig, err := api.UnmarshalTrojanConfig(r.GetRawData())
	if err != nil {
		return fmt.Errorf("unmarshal trojan config failed: %w", err)
	}

	//获取完配置，调用注册接口
	_, err = agentClient.Register(ctx, &pb.RegisterRequest{
		NodeId:   int32(s.serviceConfig.NodeID),
		NodeType: pb.NodeType_TROJAN,
	})
	if err != nil {
		return fmt.Errorf("register to agent failed: %w", err)
	}
	// 新版协议不返回 register_id，使用 NodeID 作为后续交互的 register_id
	s.registerId = int32(s.serviceConfig.NodeID)
	s.agentClient = agentClient

	inBoundConfig, err := service.InboundBuilder(s.serviceConfig, trojanConfig)
	if err != nil {
		return fmt.Errorf("build inbound config failed: %w", err)
	}

	outBoundConfig, err := service.OutboundBuilder(ctx, trojanConfig, s.extFileBytes)
	if err != nil {
		return fmt.Errorf("build outbound config failed: %w", err)
	}

	instance, err := s.loadCore(s.ctx, inBoundConfig, outBoundConfig)
	if err != nil {
		return fmt.Errorf("load xray core failed: %w", err)
	}

	if err := instance.Start(); err != nil {
		return fmt.Errorf("start xray instance failed: %w", err)
	}

	buildService := service.New(inBoundConfig.Tag, instance, s.serviceConfig, trojanConfig, agentClient, s.registerId)
	s.service = buildService
	if err := s.service.Start(); err != nil {
		return fmt.Errorf("start service failed: %w", err)
	}
	s.Running = true
	s.instance = instance
	log.Infoln("server is running")
	time.Sleep(1 * time.Minute)
	if err := s.service.StartMonitor(); err != nil {
		return fmt.Errorf("start service monitor failed: %w", err)
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
		return nil, fmt.Errorf("create xray instance failed: %w", err)
	}
	return instance, nil
}

func (s *Server) Close() {
	s.access.Lock()
	defer s.access.Unlock()

	// 在关闭服务前先尝试注销注册信息
	if s.agentClient != nil && s.registerId != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), service.DefaultTimeout)
		defer cancel()
		if _, err := s.agentClient.Unregister(ctx, &pb.UnregisterRequest{NodeType: pb.NodeType_TROJAN, RegisterId: s.registerId}); err != nil {
			log.Warnf("unregister failed: %v", err)
		}
	}
	err := s.service.Close()
	if err != nil {
		log.Fatalf("server close failed: %s", err)
	}
	log.Infoln("server close")
}
