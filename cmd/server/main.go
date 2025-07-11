package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	api "github.com/xflash-panda/server-client/pkg"
	"github.com/xflash-panda/server-trojan/internal/app/server"
	"github.com/xflash-panda/server-trojan/internal/pkg/service"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"github.com/xtls/xray-core/core"
)

const (
	Name      = "trojan-node"
	Version   = "0.2.0-dev"
	CopyRight = "XFLASH-PANDA@2021"
)

func main() {
	var config server.Config
	var apiConfig api.Config
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
				Name:        "api",
				Usage:       "Server address",
				EnvVars:     []string{"X_PANDA_TROJAN_API", "API"},
				Required:    true,
				Destination: &apiConfig.APIHost,
			},
			&cli.StringFlag{
				Name:        "token",
				Usage:       "Token of server API",
				EnvVars:     []string{"X_PANDA_TROJAN_TOKEN", "TOKEN"},
				Required:    true,
				Destination: &apiConfig.Token,
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
				EnvVars:     []string{"X_PANDA_SS_FETCH_USER_INTERVAL", "FETCH_USER_INTERVAL"},
				Value:       time.Second * 60,
				DefaultText: "60",
				Required:    false,
				Destination: &serviceConfig.FetchUsersInterval,
			},
			&cli.DurationFlag{
				Name:        "report_traffics_interval, fui",
				Usage:       "API request cycle(report traffics), unit: second",
				EnvVars:     []string{"X_PANDA_SS_FETCH_USER_INTERVAL", "REPORT_TRAFFICS_INTERVAL"},
				Value:       time.Second * 80,
				DefaultText: "80",
				Required:    false,
				Destination: &serviceConfig.ReportTrafficsInterval,
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
			var extFileBytes []byte
			if extConfPath != "" {
				log.Infof("ext config: %s", extConfPath)
				// 读取文件的二进制流
				var err error
				extFileBytes, err = os.ReadFile(extConfPath)
				if err != nil {
					return fmt.Errorf("failed to read file binary stream: %w", err)
				}
			}

			serv, err := server.New(&config, &apiConfig, &serviceConfig, extFileBytes)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}
			if err := serv.Start(); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}
			defer serv.Close()
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
