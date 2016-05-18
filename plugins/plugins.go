package plugins

import "github.com/namely/broadway/instance"

// Plugins interface declares methods that external plugins will implement
type Plugins interface {
	EmitInstanceEvent(instance.Instance)
}

// GRPCPlugins implements Plugins interface with external GRPC services
type GRPCPlugins struct {
	plugins []*grpcPlugin
}

// EmitInstanceEvent allows callers to emit an instance event
func (p *GRPCPlugins) EmitInstanceEvent(i instance.Instance) {
	for _, p := range p.plugins {
		p.emitInstanceEvent(i)
	}
}

// PluginConfig contains the necessary configuration for setting up  a plugin
type PluginConfig struct {
	Address string
	Name    string
}

// NewGRPCPlugins creates a GRPCPlugins object and returns with Plugins interface
func NewGRPCPlugins(pluginConfigs []PluginConfig) (Plugins, error) {

	ps := []*grpcPlugin{}

	for _, c := range pluginConfigs {
		ps = append(ps, grpcPlugin{config: c})
	}

	plugins := &GRPCPlugins{plugins: ps}

	return plugins, nil
}

type grpcPlugin struct {
	config PluginConfig
}

func (p grpcPlugin) emitInstanceEvent(i instance.Instance) {
}
