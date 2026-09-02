package modelengine

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// ── GLM 目录远程热更新 v0 ────────────────────────────────────────
//
// config glm_catalog_url（~/.gaea_config.json）非空时启用：app 启动异步
// 拉取 + 每 24h 周期；GET 超时 10s；响应必须是 schema v2 对象（顶层
// version 存在——裸数组/无版本一律拒绝采纳），且版本与当前缓存不同才
// 写入缓存文件 glm_catalog_remote.json（与 engines.json 同目录，原子写）。
// 目录加载器按 mtime 感知缓存变化并合并，生效优先级：
// 覆盖文件(glm_catalog_path) > 远程缓存 > 内嵌（见 glm_catalog.go）。
//
// 安全边界：远程目录仅影响模型展示（ModelInfo 列表与能力/价格注记）与
// 费用估算（estimatePrice 的目录优先层），绝不影响请求路由、alias 判定
// （glm_alias.go 为编译期静态表）、鉴权或任何发往服务端的请求内容。
// 失败（网络/解析/version 缺失/写盘）静默回退现有目录，仅失败状态翻转
// 时 slog.Warn 一次，不重复刷日志。

const (
	// glmCatalogFetchTimeout 单次拉取 HTTP 超时。
	glmCatalogFetchTimeout = 10 * time.Second
	// glmCatalogRefreshPeriod 周期拉取间隔。
	glmCatalogRefreshPeriod = 24 * time.Hour
	// glmCatalogMaxBytes 响应体上限（目录量级远小于此，防异常响应撑爆内存）。
	glmCatalogMaxBytes = 1 << 20
)

// StartGLMCatalogRemote 注入远程缓存路径并启动拉取循环（app Startup 调用，
// 非绑定方法；照 SetGLMCatalogPath 先例）。缓存路径恒注入——缓存文件持久
// 存在，url 停用后已缓存的目录仍按优先级生效；url 为空 = 拉取禁用（不启
// 协程）。ctx 随 app 生命周期；stop 通道由 StopGLMCatalogRemote 在 app
// Shutdown 关闭。重复调用幂等（先停旧循环再起新循环）。
func (m *Manager) StartGLMCatalogRemote(ctx context.Context, url, cachePath string) {
	setGLMCatalogRemotePath(cachePath)

	m.mu.Lock()
	if m.catalogRemoteStop != nil {
		close(m.catalogRemoteStop)
	}
	stop := make(chan struct{})
	m.catalogRemoteStop = stop
	m.mu.Unlock()

	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	client := &http.Client{Timeout: glmCatalogFetchTimeout}
	go func() {
		fetchGLMCatalogRemote(client, url, cachePath)
		ticker := time.NewTicker(glmCatalogRefreshPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				fetchGLMCatalogRemote(client, url, cachePath)
			}
		}
	}()
}

// StopGLMCatalogRemote 停止远程目录拉取循环（app Shutdown 调用；未启动则
// 空操作）。
func (m *Manager) StopGLMCatalogRemote() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.catalogRemoteStop != nil {
		close(m.catalogRemoteStop)
		m.catalogRemoteStop = nil
	}
}

// glmRemoteFetchState 拉取成败状态（全局：目录加载器同为包级单例）。
// 仅在状态翻转时告警，24h 周期的持续失败不会重复刷日志。
var glmRemoteFetchState = struct {
	mu     sync.Mutex
	lastOK bool // 上次拉取是否成功（初值 true：首败即告警一次）
}{lastOK: true}

// fetchGLMCatalogRemote 拉取一次并按需写缓存；失败仅状态翻转时 Warn 一次。
func fetchGLMCatalogRemote(client *http.Client, rawURL, cachePath string) {
	ok := fetchGLMCatalogRemoteOnce(client, rawURL, cachePath)
	glmRemoteFetchState.mu.Lock()
	warn := !ok && glmRemoteFetchState.lastOK
	glmRemoteFetchState.lastOK = ok
	glmRemoteFetchState.mu.Unlock()
	if warn {
		slog.Warn("GLM 目录远程拉取失败，沿用现有目录（仅影响展示与费用估算，不影响请求路由/alias/鉴权）", "url", rawURL)
	}
}

// fetchGLMCatalogRemoteOnce 执行一次拉取：GET → 校验 v2（version 存在）→
// 版本与当前缓存一致则跳过写入，否则原子写缓存文件。
func fetchGLMCatalogRemoteOnce(client *http.Client, rawURL, cachePath string) bool {
	if strings.TrimSpace(cachePath) == "" {
		return false
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, glmCatalogMaxBytes))
	if err != nil {
		return false
	}
	doc, err := parseGLMCatalog(body)
	if err != nil || !doc.HasVersion {
		// 解析失败或顶层 version 缺失：拒绝采纳（防止把裸数组/异构数据
		// 误当远程目录写进缓存）。
		return false
	}
	if glmCatalogCacheVersion(cachePath) == doc.Version {
		return true // 版本一致：跳过写入（加载器无需感知）
	}
	if err := fileutil.AtomicWrite(cachePath, body, 0644); err != nil {
		return false
	}
	return true
}

// glmCatalogCacheVersion 读取缓存文件的顶层 version（不可读/解析失败/无
// 版本 → 0，视为「无有效缓存」）。
func glmCatalogCacheVersion(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	doc, err := parseGLMCatalog(data)
	if err != nil || !doc.HasVersion {
		return 0
	}
	return doc.Version
}
