package server

import (
	"context"
	"fmt"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/app/stats"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"

	pb "github.com/xflash-panda/server-agent-proto/pkg"
	api "github.com/xflash-panda/server-client/pkg"
	_ "github.com/xflash-panda/server-trojan/internal/pkg/dep"
	"github.com/xflash-panda/server-trojan/internal/pkg/dispatcher"
	"github.com/xflash-panda/server-trojan/internal/pkg/service"
)

type Config struct {
	LogLevel  string
	AgentHost string
	AgentPort int
	DataDir   string
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
	registerId    string
	agentClient   pb.AgentClient
	closeOnce     sync.Once
}

func New(config *Config, serviceConfig *service.Config, extFileBytes []byte) (*Server, error) {
	ctx := context.Background()

	return &Server{
		config:        config,
		serviceConfig: serviceConfig,
		extFileBytes:  extFileBytes,
		ctx:           ctx,
	}, nil
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

	// 先尝试从文件加载state
	state, err := LoadState(s.config.DataDir)
	if err != nil {
		log.Warnf("load state failed: %v", err)
	}

	needRegister := false
	// 如果有保存的registerId，先验证其有效性
	if state != nil && state.RegisterId != "" {
		log.Infof("found saved state: register_id=%s, node_id=%d, hostname=%s",
			state.RegisterId, state.NodeID, state.Hostname)
		log.Infoln("verifying register_id...")

		// 调用Verify验证registerId是否有效
		_, err := agentClient.Verify(ctx, &pb.VerifyRequest{
			NodeType:   pb.NodeType_TROJAN,
			RegisterId: state.RegisterId,
		})
		if err != nil {
			// 验证失败，清空state并重新注册
			log.Warnf("verify register_id failed: %v, will re-register", err)
			needRegister = true
			if err := ClearState(s.config.DataDir); err != nil {
				log.Warnf("clear state failed: %v", err)
			}
		} else {
			// 验证成功，使用保存的registerId
			log.Infoln("register_id verified successfully")
			s.registerId = state.RegisterId
		}
	} else {
		log.Infoln("no saved state found")
		needRegister = true
	}

	// 如果需要注册，则调用注册接口
	if needRegister {
		log.Infoln("registering to agent...")
		hostname, _ := os.Hostname()
		registerResp, err := agentClient.Register(ctx, &pb.RegisterRequest{
			NodeId:   int32(s.serviceConfig.NodeID),
			NodeType: pb.NodeType_TROJAN,
			HostName: hostname,
			Port:     fmt.Sprintf("%d", trojanConfig.ServerPort),
			Ip:       "",
		})
		if err != nil {
			return fmt.Errorf("register to agent failed: %w", err)
		}
		s.registerId = registerResp.GetRegisterId()

		// 保存state到文件
		state = &State{
			RegisterId: s.registerId,
			NodeID:     s.serviceConfig.NodeID,
			Hostname:   hostname,
		}
		if err := SaveState(s.config.DataDir, state); err != nil {
			log.Warnf("save state failed: %v", err)
		}
		log.Infof("registration successful, register_id=%s", s.registerId)
	}
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
	// 立即启动监控，不再等待 1 分钟
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

func (s *Server) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		// 在关闭服务前先尝试注销注册信息
		if s.agentClient != nil && s.registerId != "" {
			ctx, cancel := context.WithTimeout(context.Background(), service.DefaultTimeout)
			defer cancel()
			if _, err := s.agentClient.Unregister(ctx, &pb.UnregisterRequest{NodeType: pb.NodeType_TROJAN, RegisterId: s.registerId}); err != nil {
				log.Warnf("unregister failed: %v", err)
				closeErr = fmt.Errorf("unregister failed: %w", err)
			} else {
				// 注销成功后清空state文件
				if err := ClearState(s.config.DataDir); err != nil {
					log.Warnf("clear state failed: %v", err)
				}
			}
		}
		// 仅当 service 已初始化时才执行关闭，避免空指针
		if s.service != nil {
			if err := s.service.Close(); err != nil {
				log.Errorf("service close failed: %s", err)
				if closeErr != nil {
					closeErr = fmt.Errorf("%v; service close failed: %w", closeErr, err)
				} else {
					closeErr = fmt.Errorf("service close failed: %w", err)
				}
			}
		}
		// 关闭 xray instance
		if s.instance != nil {
			if err := s.instance.Close(); err != nil {
				log.Errorf("xray instance close failed: %s", err)
				if closeErr != nil {
					closeErr = fmt.Errorf("%v; xray instance close failed: %w", closeErr, err)
				} else {
					closeErr = fmt.Errorf("xray instance close failed: %w", err)
				}
			} else {
				log.Infoln("xray instance closed")
			}
		}
		log.Infoln("server close")
	})
	return closeErr
}
