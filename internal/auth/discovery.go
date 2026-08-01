package auth

import (
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"time"
)

// OIDCDiscovery xAI OIDC 端点配置
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// DiscoverEndpoints 通过 OIDC Discovery 获取 xAI 的授权和 token 端点
//
// 端点摘自 hermes-agent 的 _xai_oauth_discovery()：
//
//	GET https://auth.x.ai/.well-known/openid-configuration
//	→ { authorization_endpoint, token_endpoint }
func DiscoverEndpoints() (*OIDCDiscovery, error) {
	client := netclient.NewSimpleClient(15 * time.Second)
	resp, err := client.Get("https://auth.x.ai/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("OIDC Discovery 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OIDC Discovery 返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, fmt.Errorf("读取 OIDC Discovery 响应失败: %w", err)
	}

	var disc OIDCDiscovery
	if err := json.Unmarshal(body, &disc); err != nil {
		return nil, fmt.Errorf("解析 OIDC Discovery 响应失败: %w\n原始: %s", err, string(body))
	}

	if disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" {
		return nil, fmt.Errorf("OIDC Discovery 响应缺少必要字段 (authorization_endpoint / token_endpoint)")
	}

	return &disc, nil
}
