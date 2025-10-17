package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	pb "github.com/xflash-panda/server-agent-proto/pkg"
	"github.com/xflash-panda/server-trojan/internal/app/server"
	"github.com/xflash-panda/server-trojan/internal/pkg/service"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"github.com/xtls/xray-core/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	Name      = "trojan-agent-node"
	Version   = "0.1.0"
	CopyRight = "XFLASH-PANDA@2021"
)

func main() {
	var config server.Config
	var serviceConfig service.Config
	var certConfig service.CertConfig
	var extConfPath string

	app := &cli.App{
		Name:      Name,
		Version:   Version,
		Copyright: CopyRight,
		Usage:     "Provide trojan service for the v2Board(XFLASH-PANDA)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "server_host, sh",
				Value:       "127.0.0.1",
				Usage:       "server host(agent)",
				EnvVars:     []string{"X_PANDA_TROJAN_SERVER_AGENT_HOST", "SERVER_HOST"},
				Destination: &config.AgentHost,
			},
			&cli.IntFlag{
				Name:        "port, p",
				Value:       8082,
				Usage:       "server port(agent)",
				EnvVars:     []string{"X_PANDA_TROJAN_SERVER_AGENT_HOST", "SERVER_PORT"},
				Destination: &config.AgentPort,
			},
			&cli.StringFlag{
				Name:        "ext_conf_file",
				Usage:       "Extended profiles for ACL and Outbounds(.yaml format)",
				EnvVars:     []string{"X_PANDA_TROJAN_EXT_CONF_FILE", "EXT_CONF_FILE"},
				Required:    false,
				Destination: &extConfPath,
			},
			&cli.StringFlag{
				Name:        "cert_file",
				Usage:       "Cert file",
				EnvVars:     []string{"X_PANDA_TROJAN_CERT_FILE", "CERT_FILE"},
				Value:       "/root/.cert/server.crt",
				Required:    false,
				DefaultText: "/root/.cert/server.crt",
				Destination: &certConfig.CertFile,
			},
			&cli.StringFlag{
				Name:        "key_file",
				Usage:       "Key file",
				EnvVars:     []string{"X_PANDA_TROJAN_KEY_FILE", "KEY_FILE"},
				Value:       "/root/.cert/server.key",
				Required:    false,
				DefaultText: "/root/.cert/server.key",
				Destination: &certConfig.KeyFile,
			},
			&cli.IntFlag{
				Name:        "node",
				Usage:       "Node ID",
				EnvVars:     []string{"X_PANDA_TROJAN_NODE", "NODE"},
				Required:    true,
				Destination: &serviceConfig.NodeID,
			},
			&cli.DurationFlag{
				Name:        "fetch_users_interval, fui",
				Usage:       "API request cycle(fetch users), unit: second",
				EnvVars:     []string{"X_PANDA_TROJAN_FETCH_USER_INTERVAL", "FETCH_USER_INTERVAL"},
				Value:       time.Second * 60,
				DefaultText: "60",
				Required:    false,
				Destination: &serviceConfig.FetchUsersInterval,
			},
			&cli.DurationFlag{
				Name:        "report_traffics_interval, fui",
				Usage:       "API request cycle(report traffics), unit: second",
				EnvVars:     []string{"X_PANDA_SS_TROJAN_USER_INTERVAL", "REPORT_TRAFFICS_INTERVAL"},
				Value:       time.Second * 80,
				DefaultText: "80",
				Required:    false,
				Destination: &serviceConfig.ReportTrafficsInterval,
			},
			&cli.DurationFlag{
				Name:        "heartbeat_interval",
				Usage:       "API request cycle(heartbeat), unit: second",
				EnvVars:     []string{"X_PANDA_SS_HEARTBEAT_INTERVAL", "HEARTTBEAT_INTERVAL"},
				Value:       time.Second * 60,
				DefaultText: "60 seconds",
				Required:    false,
				Destination: &serviceConfig.HeartBeatInterval,
			},
			&cli.StringFlag{
				Name:        "log_mode",
				Value:       server.LogLevelError,
				Usage:       "Log mode",
				EnvVars:     []string{"X_PANDA_TROJAN_LOG_LEVEL", "LOG_LEVEL"},
				Destination: &config.LogLevel,
				Required:    false,
			},
		},
		Before: func(c *cli.Context) error {
			log.SetFormatter(&log.TextFormatter{})
			if config.LogLevel == server.LogLevelDebug {
				log.SetFormatter(&log.TextFormatter{
					FullTimestamp: true,
				})
				log.SetLevel(log.DebugLevel)
				log.SetReportCaller(true)
			} else if config.LogLevel == server.LogLevelInfo {
				log.SetLevel(log.InfoLevel)
			} else if config.LogLevel == server.LogLevelError {
				log.SetLevel(log.ErrorLevel)
			} else {
				return fmt.Errorf("log mode %s not supported", config.LogLevel)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			if config.LogLevel != server.LogLevelDebug {
				defer func() {
					if e := recover(); e != nil {
						panic(e)
					}
				}()
			}
			serviceConfig.Cert = &certConfig
			agentAddr := fmt.Sprintf("%s:%d", config.AgentHost, config.AgentPort)
			agentConn, err := grpc.NewClient(agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithKeepaliveParams(
				keepalive.ClientParameters{
					Time:                30 * time.Second, // 每30秒发送一次keepalive探测
					Timeout:             10 * time.Second, // 如果10秒内没有响应，则认为连接断开
					PermitWithoutStream: true,             // 允许即使没有活动流的情况下也发送探测
				}))
			if err != nil {
				panic(fmt.Errorf("connect agent server %s failed: %w", agentAddr, err))
			}
			agentClient := pb.NewAgentClient(agentConn)
			defer agentConn.Close()

			var extFileBytes []byte
			if extConfPath != "" {
				log.Infof("ext config: %s", extConfPath)
				// 读取文件的二进制流
				var err error
				extFileBytes, err = os.ReadFile(extConfPath)
				if err != nil {
					return fmt.Errorf("read ext config file %s failed: %w", extConfPath, err)
				}
			}

			serv, err := server.New(&config, &serviceConfig, extFileBytes)
			if err != nil {
				return fmt.Errorf("create server failed: %w", err)
			}
			if err := serv.Start(agentClient); err != nil {
				// Start失败时，需要调用Close进行清理（包括取消注册）
				serv.Close()
				return fmt.Errorf("start server failed: %w", err)
			}

			// 确保无论正常退出还是异常退出都会调用 Close
			defer func() {
				if e := recover(); e != nil {
					log.Errorf("panic: %v", e)
					// 打印堆栈信息
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					log.Errorf("stack trace:\n%s", buf[:n])
					// 调用 Close 进行清理
					serv.Close()
					os.Exit(1)
				} else {
					// 正常退出时也调用 Close
					serv.Close()
				}
			}()

			runtime.GC()
			{
				osSignals := make(chan os.Signal, 1)
				signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
				<-osSignals
			}
			return nil
		},
	}

	app.Commands = []*cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show version information",
			Action: func(c *cli.Context) error {
				fmt.Printf("version=%s xray.version=%s\n", Version, core.Version())
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
