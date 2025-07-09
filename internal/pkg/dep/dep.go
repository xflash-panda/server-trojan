package dep

import (
	// 核心应用功能
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/metrics"
	_ "github.com/xtls/xray-core/app/policy"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	_ "github.com/xtls/xray-core/app/stats"

	// 配置加载
	_ "github.com/xtls/xray-core/main/json"

	// 代理协议 - 只保留必要的
	_ "github.com/xflash-panda/server-trojan/internal/pkg/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/trojan"

	// 传输层 - 只保留必要的
	_ "github.com/xtls/xray-core/transport/internet/headers/noop"
	_ "github.com/xtls/xray-core/transport/internet/headers/tls"
	_ "github.com/xtls/xray-core/transport/internet/tcp"
	_ "github.com/xtls/xray-core/transport/internet/tls"
	_ "github.com/xtls/xray-core/transport/internet/udp"
	_ "github.com/xtls/xray-core/transport/internet/websocket"
)
