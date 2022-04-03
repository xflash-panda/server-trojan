package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xflash-panda/server-trojan/internal/app/server"
	"github.com/xflash-panda/server-trojan/internal/pkg/api"
	"github.com/xflash-panda/server-trojan/internal/pkg/service"
	"github.com/xtls/xray-core/core"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	Name          = "trojan-node"
	Version       = "0.1.5"
	CopyRight     = "XFLASH-PANDA@2021"
	LogLevelDebug = "debug"
	LogLevelError = "error"
	LogLevelInfo  = "info"
)

func init() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}
	cli.ErrWriter = ioutil.Discard

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s xray.version=%s\n", Version, core.Version())
	}
}

func main() {
	var config server.Config
	var apiConfig api.Config
	var serviceConfig service.Config
	var certConfig service.CertConfig

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
			&cli.IntFlag{
				Name:        "node",
				Usage:       "Node ID",
				EnvVars:     []string{"X_PANDA_TROJAN_NODE", "NODE"},
				Required:    true,
				Destination: &apiConfig.NodeID,
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
			&cli.DurationFlag{
				Name:        "sys_interval",
				Usage:       "API request cycle, unit: second",
				EnvVars:     []string{"X_PANDA_TROJAN_SYS_INTERVAL", "SYS_INTERVAL"},
				Value:       time.Second * 60,
				DefaultText: "60",
				Required:    false,
				Destination: &serviceConfig.SysInterval,
			},

			&cli.StringFlag{
				Name:        "log_mode",
				Value:       LogLevelError,
				Usage:       "Log mode",
				EnvVars:     []string{"X_PANDA_TROJAN_LOG_LEVEL", "LOG_LEVEL"},
				Destination: &config.LogLevel,
				Required:    false,
			},
		},
		Before: func(c *cli.Context) error {
			log.SetFormatter(&log.TextFormatter{})
			if config.LogLevel == LogLevelDebug {
				log.SetFormatter(&log.TextFormatter{
					FullTimestamp: true,
				})
				log.SetLevel(log.DebugLevel)
				log.SetReportCaller(true)
			} else if config.LogLevel == LogLevelInfo {
				log.SetLevel(log.InfoLevel)
			} else if config.LogLevel == LogLevelError {
				log.SetLevel(log.ErrorLevel)
			} else {
				return fmt.Errorf("log mode %s not supported", config.LogLevel)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			if config.LogLevel != LogLevelDebug {
				defer func() {
					if r := recover(); r != nil {
						log.Fatal(r)
					}
				}()
			}
			serviceConfig.Cert = &certConfig
			serv := server.New(&config, &apiConfig, &serviceConfig)
			serv.Start()
			defer serv.Close()
			runtime.GC()
			{
				osSignals := make(chan os.Signal, 1)
				signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
				<-osSignals
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
