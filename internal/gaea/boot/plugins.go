package boot

import (
	"context"
	"io"
	"os"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/plugin"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// pluginsOut carries the artifacts from starting plugins and LSP.
type pluginsOut struct {
	host    *plugin.Host
	cleanup func()
}

// startPlugins initialises CodeGraph (if enabled), Context7 (if key set),
// configured MCP servers, and LSP tools. It returns a cleanup function that
// shuts down all spawned subprocesses.
// S1.3-B：space 为装配空间（""=space.mode=off 全注册现状）。MCP 热插拔绕过
// 构建期过滤，因此启动挂载与 control.connectMCPSpec 的热插补挂载走同一规则
// （builtin.AllowsSpace，MCP 工具缺省 shared）——spec 层过滤，不改运行时。
func startPlugins(ctx context.Context, cfg *config.Config, reg *tool.Registry, sink event.Sink, stderrPath io.Writer, space string) *pluginsOut {
	out := &pluginsOut{}
	pluginHost := plugin.NewHost()
	specs := PluginSpecs(cfg.AutoStartPlugins())

	if key := os.Getenv("CONTEXT7_API_KEY"); key != "" {
		specs = append(specs, plugin.Spec{
			Name:    "context7",
			Type:    "http",
			URL:     "https://mcp.context7.com/mcp",
			Headers: map[string]string{"Authorization": "Bearer " + key},
		})
	}
	if len(specs) > 0 {
		if stderrPath != nil {
			for i := range specs {
				specs[i].Stderr = stderrPath
			}
		}
		host, ptools := plugin.StartAvailable(ctx, specs)
		pluginHost = host
		for _, t := range ptools {
			if builtin.AllowsSpace(t, space) {
				reg.Add(t)
			}
		}
		if text, ok := MCPStartupNotice(host.Failures()); ok {
			sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: text})
		}
	}
	out.host = pluginHost
	out.cleanup = pluginHost.Close

	return out
}
