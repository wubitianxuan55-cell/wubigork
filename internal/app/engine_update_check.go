package app

// engine_update_check.go — 受管本地引擎官方新版检查（Strata / llama / WHIRL）。
//
// 三引擎各自的本地版本号与官方发布通道：
//   - WHIRL: whirl-server.exe --version → "0.2.5.1"；官方 tsaipifong/whirl-llm
//   - llama: llama-server.exe --version → "0.8.0-dev (build 1 …)"；官方 ggml-org/llama.cpp
//   - Strata: engine\strata.exe --version → "0.1.56"；官方 Niko1221/Strata
//
// 官方侧统一走 GitHub releases/latest（免鉴权 60 次/小时，按钮触发+30 分钟
// 内存缓存足够）。版本比较取首个点分数字段（剥 v 前缀/-dev 后缀）逐段比。
// 本刀只做「告知」不做升级——下载/替换安装保持用户手动（引擎在跑、DLL 同驻
// 约束各异，自动升级风险大于收益）。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// engineUpdateSpec 单引擎的版本探针与官方通道。
type engineUpdateSpec struct {
	// repo GitHub releases/latest 的 owner/name。
	repo string
	// binPath 本地可执行（--version 探针目标），函数形态便于取各引擎安装根。
	binPath func(a *App) string
	// assetVersionRe 非 nil 时：官方版本不取 release tag，而从 release 资产名
	// 抓组件版本（strata 的 release tag 0.2.x 是控制端版本，本地 strata.exe
	// 报的是 engine 组件 0.1.x——两序列不同，直接比 tag 会永远显示有新版本）。
	assetVersionRe *regexp.Regexp
}

func strataBinPath(a *App) string {
	return filepath.Join(a.strataProjectPath(), "engine", "strata.exe")
}

var strataEngineAssetRe = regexp.MustCompile(`strata-engine-([0-9]+\.[0-9]+\.[0-9]+)-windows`)

var engineUpdateSpecs = map[string]engineUpdateSpec{
	"whirl":  {repo: "tsaipifong/whirl-llm", binPath: func(a *App) string { return whirlBinPath(a) }},
	"llama":  {repo: "ggml-org/llama.cpp", binPath: func(*App) string { return voiceV2LLMBinPath() }},
	"strata": {repo: "Niko1221/Strata", binPath: strataBinPath, assetVersionRe: strataEngineAssetRe},
}

var (
	versionNumRe = regexp.MustCompile(`\bv?(\d+(?:\.\d+){1,3})\b`)
	// llama.cpp 旧版形态 "version: 4559 (3efa77d)"——version: 前缀行允许裸整数。
	versionBareRe = regexp.MustCompile(`version:\s*v?(\d+)\b`)

	// 探针 seam：测试注入（绝不真 exec / 真联网——app 测试隔离铁律）。
	engineVersionProbe = func(bin string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "--version")
		proc.HideWindow(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}
	engineUpdateFetchLatest = func(repo string) (tag, name, url string, assets []string, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo), nil)
		if err != nil {
			return "", "", "", nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", "", "", nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return "", "", "", nil, fmt.Errorf("GitHub API HTTP %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if err != nil {
			return "", "", "", nil, err
		}
		var payload struct {
			TagName string `json:"tag_name"`
			Name    string `json:"name"`
			HTMLURL string `json:"html_url"`
			Assets  []struct {
				Name string `json:"name"`
			} `json:"assets"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return "", "", "", nil, fmt.Errorf("GitHub 响应解析失败: %w", err)
		}
		if payload.TagName == "" {
			return "", "", "", nil, fmt.Errorf("GitHub 响应缺 tag_name")
		}
		for _, a := range payload.Assets {
			assets = append(assets, a.Name)
		}
		return payload.TagName, payload.Name, payload.HTMLURL, assets, nil
	}
)

// engineUpdateCache 30 分钟内存缓存（GitHub 免鉴权限流 + 按钮防抖）。
var (
	engineUpdateCacheMu sync.Mutex
	engineUpdateCache   = map[string]engineUpdateResult{}
	engineUpdateCacheAt = map[string]time.Time{}
)

const engineUpdateCacheTTL = 30 * time.Minute

// ResetEngineUpdateCacheForTests 清缓存（测试隔离）。
func ResetEngineUpdateCacheForTests() {
	engineUpdateCacheMu.Lock()
	defer engineUpdateCacheMu.Unlock()
	engineUpdateCache = map[string]engineUpdateResult{}
	engineUpdateCacheAt = map[string]time.Time{}
}

type engineUpdateResult struct {
	Engine        string `json:"engine"`
	LocalVersion  string `json:"local_version"`
	LatestVersion string `json:"latest_version"`
	LatestName    string `json:"latest_name"`
	ReleaseURL    string `json:"release_url"`
	HasUpdate     bool   `json:"has_update"`
	CheckedAt     string `json:"checked_at"`
	Error         string `json:"error,omitempty"`
}

// parseFirstVersion 从 --version 输出抓首个版本号（剥 v 前缀）。逐行扫描：
// 每行先找点分号（whirl-server 0.2.5.1 / version: 0.8.0-dev / strata 0.1.56），
// 整行无点分且带 version: 前缀时接受裸整数（llama.cpp 旧 b-number 形态）——
// 防止 MSVC 19.44 之类构建器尾注被误抓。
func parseFirstVersion(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := versionNumRe.FindStringSubmatch(line); m != nil {
			return m[1]
		}
		if m := versionBareRe.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// compareVersions 点分逐段比较；段数不足补零（0.2.5 == 0.2.5.0）。
// 返回 -1/0/1。非数字段按 0（-dev 后缀已在抓取时剥除——正则只吃数字段）。
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(as) {
			ai, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(bs[i])
		}
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

// EngineUpdateCheck 检查指定受管引擎的官方最新版本。支持 whirl / llama / strata。
func (a *App) EngineUpdateCheck(engineID string) map[string]interface{} {
	spec, ok := engineUpdateSpecs[engineID]
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf("引擎 %s 不支持版本检查（仅 strata/llama/whirl）", engineID)}
	}

	engineUpdateCacheMu.Lock()
	if at, hit := engineUpdateCacheAt[engineID]; hit && time.Since(at) < engineUpdateCacheTTL {
		res := engineUpdateCache[engineID]
		engineUpdateCacheMu.Unlock()
		return mapToIface(res)
	}
	engineUpdateCacheMu.Unlock()

	res := engineUpdateResult{Engine: engineID, CheckedAt: time.Now().Format("2006-01-02 15:04:05")}

	bin := spec.binPath(a)
	if bin == "" {
		res.Error = "引擎可执行路径未配置"
		return mapToIface(res)
	}
	if _, err := os.Stat(bin); err != nil {
		res.Error = fmt.Sprintf("引擎可执行不存在（%s）", bin)
		return mapToIface(res)
	}
	out, err := engineVersionProbe(bin)
	if err != nil {
		res.Error = fmt.Sprintf("本地版本探测失败: %v", err)
		return mapToIface(res)
	}
	res.LocalVersion = parseFirstVersion(out)
	if res.LocalVersion == "" {
		res.Error = fmt.Sprintf("无法从 --version 输出解析版本号（%s）", firstLine(out))
		return mapToIface(res)
	}

	tag, name, url, assets, err := engineUpdateFetchLatest(spec.repo)
	if err != nil {
		res.Error = fmt.Sprintf("官方版本查询失败: %v", err)
		// 本地版本已知、官方侧失败：照样返回（不算全失败）
		cacheEngineUpdate(engineID, res)
		return mapToIface(res)
	}
	res.LatestVersion = parseFirstVersion(tag)
	// 组件版本口径（strata）：release tag 是控制端版本序列，本地 --version 报
	// engine 组件序列——从资产名抓 engine 版本作比较基准。
	if spec.assetVersionRe != nil {
		for _, an := range assets {
			if m := spec.assetVersionRe.FindStringSubmatch(an); m != nil {
				res.LatestVersion = m[1]
				break
			}
		}
	}
	res.LatestName = name
	res.ReleaseURL = url
	res.HasUpdate = res.LatestVersion != "" && compareVersions(res.LocalVersion, res.LatestVersion) < 0

	cacheEngineUpdate(engineID, res)
	return mapToIface(res)
}

func cacheEngineUpdate(engineID string, res engineUpdateResult) {
	engineUpdateCacheMu.Lock()
	defer engineUpdateCacheMu.Unlock()
	engineUpdateCache[engineID] = res
	engineUpdateCacheAt[engineID] = time.Now()
}

func mapToIface(res engineUpdateResult) map[string]interface{} {
	return map[string]interface{}{
		"engine":         res.Engine,
		"local_version":  res.LocalVersion,
		"latest_version": res.LatestVersion,
		"latest_name":    res.LatestName,
		"release_url":    res.ReleaseURL,
		"has_update":     res.HasUpdate,
		"checked_at":     res.CheckedAt,
		"error":          res.Error,
	}
}

// EngineUpdateStatus 引擎卡聚合展示：三引擎一次全查（各走缓存）。
func (a *App) EngineUpdateStatus() map[string]interface{} {
	out := map[string]interface{}{}
	for id := range engineUpdateSpecs {
		out[id] = a.EngineUpdateCheck(id)
	}
	return out
}
