package service

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/xflash-panda/server-client/pkg"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

// OutboundBuilder build monster outbund config for addoutbound
func OutboundBuilder(ctx context.Context, nodeInfo *api.TrojanConfig, extConfigBytes []byte) (*core.OutboundHandlerConfig, error) {
	outboundDetourConfig := &conf.OutboundDetourConfig{}
	outboundDetourConfig.Protocol = "monster"
	outboundDetourConfig.Tag = fmt.Sprintf("monster_%d", nodeInfo.ServerPort)
	proxySetting := &conf.MonsterConfig{
		ACLYamlContent: extConfigBytes,
	}
	var setting json.RawMessage
	setting, err := json.Marshal(proxySetting)
	if err != nil {
		return nil, fmt.Errorf("marshal proxy Mmonster config failed: %s", err)
	}

	outboundDetourConfig.Settings = &setting

	return outboundDetourConfig.Build()
}
