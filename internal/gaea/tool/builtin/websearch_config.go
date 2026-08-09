package builtin

import (
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/netclient"
)

// searchCfg holds the runtime search configuration injected by boot.
// nil means all API-based engines are disabled; only public SearXNG works.
var searchCfg *config.SearchConfig

// searchProxy holds the runtime network proxy injected by boot. web_search
// requests honour the same auto/env/custom/off proxy modes as web_fetch.
var searchProxy netclient.ProxySpec

// SetSearchConfig injects the search configuration from boot assembly.
// Call once before any web_search tool execution.
func SetSearchConfig(cfg config.SearchConfig) {
	cp := cfg
	searchCfg = &cp
}

// SetSearchProxy injects the network proxy used by web_search requests.
func SetSearchProxy(spec netclient.ProxySpec) {
	searchProxy = spec
}
