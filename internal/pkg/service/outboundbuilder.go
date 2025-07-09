package service

import (
	"context"
	"fmt"

	api "github.com/xflash-panda/server-client/pkg"
	"github.com/xflash-panda/server-trojan/internal/pkg/proxy/freedom"

	C "github.com/apernet/hysteria/core/v2/server"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

// OutboundBuilder build freedom outbund config for addoutbound
func OutboundBuilder(ctx context.Context, nodeInfo *api.TrojanConfig, pOutbound C.Outbound) (*core.OutboundHandlerConfig, error) {
	outboundDetourConfig := &conf.OutboundDetourConfig{}
	outboundDetourConfig.Protocol = "freedom"
	outboundDetourConfig.Tag = fmt.Sprintf("%s_%d", protocol, nodeInfo.ServerPort)

	// 使用传入的context创建带PluggableOutbound的context
	freedom.WithPluggableOutbound(ctx, pOutbound)

	// Freedom Protocol setting
	return outboundDetourConfig.Build()
}
