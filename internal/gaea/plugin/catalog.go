// catalog.go 是 MCP 目录缓存与启动 deadline（codex 蒸馏 B8 折算，第四批 P9）。
//
// 机制同构 codex MCP 客户端的「目录缓存 32 条 / TTL 30min / 身份指纹键 +
// 启动期 deadline 解耦」，折算到 gaea 的 plugin host：
//   - 目录缓存缓存的是**目录形状**（mcpTool 原始清单）而非工具对象——
//     remoteTool 绑定当时的连接，跨重建不可复用；命中缓存免一次
//     tools/list 往返，用缓存目录对新连接重建包装；
//   - 身份指纹 = spec 的 type+command+args+env+url+headers 规范化哈希：
//     同配置跨 controller 重建（reload/换配置）命中，改配置自然换键；
//   - 启动 deadline：start+目录拉取总预算 15s——慢/挂的 MCP 服务器只让
//     自己失败（既有 fail-soft：其余服务器照常），不再拖住 controller 构建。
package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// 上限常量（上游生产值同源：32 条 / 30 分钟）。
const (
	MCPStartTimeout  = 15 * time.Second
	catalogTTL       = 30 * time.Minute
	catalogMaxEntries = 32
)

// SpecFingerprint 返回服务器身份指纹（SHA256 前 16 hex）。args 保序、env 与
// headers 键序归一——语义相同的 spec 命中同一指纹，改任何一项即换键。
func SpecFingerprint(s Spec) string {
	h := sha256.New()
	writeField := func(k, v string) { fmt.Fprintf(h, "%s\x00%s\x00", k, v) }
	writeField("type", s.Type)
	writeField("command", s.Command)
	writeField("url", s.URL)
	writeField("dir", s.Dir)
	for i, a := range s.Args {
		writeField(fmt.Sprintf("arg%d", i), a)
	}
	for _, k := range sortedKeys(s.Env) {
		writeField("env."+k, s.Env[k])
	}
	for _, k := range sortedKeys(s.Headers) {
		writeField("header."+k, s.Headers[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// catalogEntry 是一份缓存目录。
type catalogEntry struct {
	tools []mcpTool
	at    time.Time
}

// catalog 是进程级目录缓存（跨 controller 重建复用；容量 32，满逐出最旧）。
var catalog = struct {
	mu sync.Mutex
	m  map[string]catalogEntry
}{m: map[string]catalogEntry{}}

func catalogGet(fp string) ([]mcpTool, bool) {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	e, ok := catalog.m[fp]
	if !ok || time.Since(e.at) > catalogTTL {
		if ok {
			delete(catalog.m, fp) // 过期即清，不留死键
		}
		return nil, false
	}
	return e.tools, true
}

func catalogPut(fp string, tools []mcpTool) {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if len(catalog.m) >= catalogMaxEntries {
		// 容量满：逐出最旧（缓存是加速面，逐出策略无须精巧）。
		var oldestKey string
		var oldestAt time.Time
		for k, e := range catalog.m {
			if oldestKey == "" || e.at.Before(oldestAt) {
				oldestKey, oldestAt = k, e.at
			}
		}
		delete(catalog.m, oldestKey)
	}
	catalog.m[fp] = catalogEntry{tools: tools, at: time.Now()}
}

// CatalogStats 供测试与诊断：条目数。
func CatalogStats() int {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	return len(catalog.m)
}

// catalogReset 清空缓存（测试用）。
func catalogReset() {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	catalog.m = map[string]catalogEntry{}
}

// listToolsCached 带缓存的目录拉取：命中免 tools/list 往返（用缓存目录对新
// 连接重建 remoteTool 包装）；未命中走 listTools 并入缓存。
func (c *Client) listToolsCached(ctx context.Context, fp string) ([]tool.Tool, error) {
	if cached, ok := catalogGet(fp); ok {
		return c.toolsFromCatalog(cached), nil
	}
	ts, err := c.listTools(ctx)
	if err != nil {
		return nil, err
	}
	catalogPut(fp, c.lastRawTools)
	return ts, nil
}

// toolsFromCatalog 用缓存目录形状对新连接重建工具包装（绑定当前 client）。
func (c *Client) toolsFromCatalog(raw []mcpTool) []tool.Tool {
	tools := make([]tool.Tool, 0, len(raw))
	infos := make([]ToolInfo, 0, len(raw))
	for _, t := range raw {
		infos = append(infos, ToolInfo{Name: t.Name, Description: t.Description})
		tools = append(tools, &remoteTool{
			client:   c,
			name:     toolName(c.name, t.Name),
			rawName:  t.Name,
			desc:     t.Description,
			schema:   canonicalizeSchema(t.InputSchema),
			readOnly: t.Annotations != nil && t.Annotations.ReadOnlyHint,
		})
	}
	sort.SliceStable(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	c.tools = infos
	return sortToolsByName(tools)
}

// startWithDeadline 是 start + 目录拉取的 deadline 包装（B8 启动解耦）：
// 超时返回错误（该服务器失败，其余照常——fail-soft 语义由调用方既有路径保持）。
func startWithDeadline(ctx context.Context, s Spec) (*Client, []tool.Tool, error) {
	dctx, cancel := context.WithTimeout(ctx, MCPStartTimeout)
	defer cancel()
	c, err := start(dctx, s)
	if err != nil {
		return nil, nil, err
	}
	ts, err := c.listToolsCached(dctx, SpecFingerprint(s))
	if err != nil {
		c.close()
		return nil, nil, err
	}
	return c, ts, nil
}
