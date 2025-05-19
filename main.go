package main

import (
	"github.com/argoproj-labs/rollouts-plugin-metric-coralogix/internal/plugin"
	rolloutsPlugin "github.com/argoproj/argo-rollouts/metricproviders/plugin/rpc"
	goPlugin "github.com/hashicorp/go-plugin"
	log "github.com/sirupsen/logrus"
)

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = goPlugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ARGO_ROLLOUTS_RPC_PLUGIN",
	MagicCookieValue: "metricprovider",
}

func main() {
	logCtx := *log.WithFields(log.Fields{"plugin": "coralogix"})

	rpcPluginImp := &plugin.RpcPlugin{
		LogCtx: logCtx,
	}
	// pluginMap is the map of plugins we can dispense.
	pluginMap := map[string]goPlugin.Plugin{
		"RpcMetricProviderPlugin": &rolloutsPlugin.RpcMetricProviderPlugin{Impl: rpcPluginImp},
	}

	logCtx.Debug("message from plugin", "ping", "pong")

	goPlugin.Serve(&goPlugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
