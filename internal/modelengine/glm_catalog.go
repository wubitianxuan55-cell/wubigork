package modelengine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── GLM 静态目录（内嵌 JSON + 覆盖文件/远程缓存热更新）─────────
//
// 智谱官方 API 无模型列表端点，目录锚定 docs.bigmodel.cn「模型概览」。
//
// schema v2（B 刀）：顶层 {"version":2,"updated":"...","models":[...]}；
// 条目 = id + kind(可选) + 能力/价格元数据（context_length/max_output/price_in/
// price_out/currency/unit/free/caps/price_note/points_*，全部可选）。
// 解析向后兼容：同时接受 v2 对象与 v1 裸数组（数组 = 无版本条目集）。
// kind 省略时由 ClassifyModelKind 统一分类兜底（glm-image 系为生图、
// glm-tts→tts、glm-asr→stt、embedding→embedding、glm-ocr→ocr，其余为 llm）。
//
// 数据来源与核实日期（写在这里——JSON 无注释）：
//   - 模型清单/上下文/最大输出/能力标记/免费档/alias：
//     docs.bigmodel.cn「模型概览」，2026-09-02 核实；
//   - 国内价（CNY，官方核实）：glm-ocr 输入输出 0.2 元/M tokens、
//     embedding-3 0.5 元/M tokens、glm-image 0.1 元/次、cogvideox-3 1 元/次、
//     glm-realtime-flash 0.18 元/分钟、glm-realtime-air 0.3 元/分钟；
//     glm-5.3-flash 官方仅给相对价（=glm-5.3 的 1/10，限时 1/20），
//     不填绝对价（price_note 记录口径，估算回退内置表）；其余付费模型
//     绝对价官方页动态渲染未查到——不填 price，估算回退 modelPricing；
//   - coding 套餐积分系数（积分=(输入×In+缓存命中×Cached+输出×Out)/10000，
//     仅 coding 端点支持的 glm-5.3 / glm-5.3-flash 有）：
//     glm-5.3 In 6.9 / Cached 1.7 / Out 24（高峰×3）；
//     glm-5.3-flash In 2.3 / Cached 0.56 / Out 8（高峰×1.2，非高峰 0.4）。
//
// 合并语义（生效优先级：覆盖文件 > 远程缓存 > 内嵌）：同 ID 替换——
// 字段显式给出（非零值/非空，照既有 Kind 处理模式泛化）才覆盖、否则保留
// 下层值；新 ID 追加尾部；读取/解析失败静默回退下层（只打日志）。

//go:embed glm_catalog.json
var glmCatalogJSON []byte

// glmCatalogEntry 目录条目（内嵌与覆盖/远程文件同 schema）。
// Free 用指针：区分「显式 free:false」与「未给出」（覆盖语义需要）。
type glmCatalogEntry struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind,omitempty"`
	ContextLength int      `json:"context_length,omitempty"`
	MaxOutput     int      `json:"max_output,omitempty"`
	PriceIn       float64  `json:"price_in,omitempty"`
	PriceOut      float64  `json:"price_out,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	Free          *bool    `json:"free,omitempty"`
	Caps          []string `json:"caps,omitempty"`
	PriceNote     string   `json:"price_note,omitempty"`
	PointsIn      float64  `json:"points_in,omitempty"`
	PointsCached  float64  `json:"points_cached,omitempty"`
	PointsOut     float64  `json:"points_out,omitempty"`
	PointsPeak    float64  `json:"points_peak,omitempty"`
}

// glmCatalogDoc parseGLMCatalog 的解析结果。
type glmCatalogDoc struct {
	Entries    []glmCatalogEntry
	Version    int  // schema 版本（裸数组 = 0）
	HasVersion bool // 顶层 version 是否存在（远程目录采纳的门控条件）
	Updated    string
}

// parseGLMCatalog 解析目录 JSON：schema v2 顶层对象（{version,updated,models}）
// 或 v1 裸数组（无版本条目集）。未知字段忽略（json.Unmarshal 默认）。
func parseGLMCatalog(data []byte) (glmCatalogDoc, error) {
	var raw struct {
		Version *int              `json:"version"`
		Updated string            `json:"updated"`
		Models  []glmCatalogEntry `json:"models"`
	}
	if err := json.Unmarshal(data, &raw); err == nil && raw.Models != nil {
		doc := glmCatalogDoc{Entries: raw.Models, Updated: raw.Updated}
		if raw.Version != nil {
			doc.Version, doc.HasVersion = *raw.Version, true
		}
		return doc, nil
	}
	var entries []glmCatalogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return glmCatalogDoc{}, err
	}
	return glmCatalogDoc{Entries: entries}, nil
}

// fileState 目录文件层的 mtime 状态（不存在时 exists=false，mtime 零值）。
type fileState struct {
	exists bool
	mod    time.Time
}

// sameAs 状态是否未变（mtime 用 time.Equal——忽略单调时钟位，与既有
// ModTime().Equal 口径一致）。
func (a fileState) sameAs(b fileState) bool {
	return a.exists == b.exists && a.mod.Equal(b.mod)
}

func statFileState(path string) fileState {
	if path == "" {
		return fileState{}
	}
	st, err := os.Stat(path)
	if err != nil {
		return fileState{}
	}
	return fileState{exists: true, mod: st.ModTime()}
}

// glmCatalogLoader 内嵌目录解析 + 覆盖文件/远程缓存 mtime 热更新状态。
// 任一层 mtime 未变化时直接复用上次合并结果（零磁盘 IO 除 stat 外）。
type glmCatalogLoader struct {
	mu          sync.Mutex
	base        []ModelInfo // 内嵌目录解析结果（只解析一次）
	baseVersion int         // 内嵌 schema 版本（2）
	baseUpdated string      // 内嵌 updated（"2026-09-02"）
	path        string      // 覆盖文件路径（glm_catalog_path；空=无该层）
	remotePath  string      // 远程缓存路径（glm_catalog_remote.json；空=无该层）
	merged      []ModelInfo // 上次合并结果缓存（nil=未算）
	effVersion  string      // 当前生效 schema 版本（透传 catalog_version）
	source      string      // 当前生效源描述（透传 catalog_source）
	ovState     fileState   // 覆盖文件上次加载状态
	rmState     fileState   // 远程缓存上次加载状态
}

var glmCatalog = &glmCatalogLoader{}

// setGLMCatalogPath 设置覆盖文件路径（app 启动按 ~/.gaea_config.json 的
// glm_catalog_path 注入一次；空=无覆盖层）。
func setGLMCatalogPath(path string) {
	p := strings.TrimSpace(path)
	glmCatalog.mu.Lock()
	defer glmCatalog.mu.Unlock()
	if p == glmCatalog.path {
		return
	}
	glmCatalog.path = p
	glmCatalog.merged = nil // 路径变化强制重算
	glmCatalog.ovState = fileState{}
}

// setGLMCatalogRemotePath 设置远程缓存文件路径（app 启动注入一次，与
// engines.json 同目录；空=无远程层。缓存文件持久存在，url 停用后仍生效，
// 由生效优先级 覆盖 > 远程 > 内嵌 约束）。
func setGLMCatalogRemotePath(path string) {
	p := strings.TrimSpace(path)
	glmCatalog.mu.Lock()
	defer glmCatalog.mu.Unlock()
	if p == glmCatalog.remotePath {
		return
	}
	glmCatalog.remotePath = p
	glmCatalog.merged = nil
	glmCatalog.rmState = fileState{}
}

// glmStaticModels 返回 GLM 静态模型目录（内嵌 + 远程缓存 + 覆盖文件按
// 优先级合并后的快照）。每次调用检查两层文件 mtime，变化才重读合并；
// 保持无参签名，TestConnection/fetchModels 的 GLM 分支调用点不变。
func glmStaticModels() []ModelInfo {
	return glmCatalog.models()
}

// models 返回目录快照。
func (l *glmCatalogLoader) models() []ModelInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.modelsLocked()
}

// modelsLocked 返回目录快照（需持锁调用）。两层文件状态任一变化即重算；
// 解析失败静默回退下层并记下状态，避免每次调用重放同一条错误日志。
func (l *glmCatalogLoader) modelsLocked() []ModelInfo {
	base := l.baseEntries()
	if l.path == "" && l.remotePath == "" {
		if l.merged == nil {
			l.merged = base
			l.effVersion, l.source = l.builtinInfo()
		}
		return append([]ModelInfo(nil), l.merged...)
	}
	ov := statFileState(l.path)
	rm := statFileState(l.remotePath)
	if l.merged != nil && ov.sameAs(l.ovState) && rm.sameAs(l.rmState) {
		return append([]ModelInfo(nil), l.merged...)
	}
	cur := base
	version := l.baseVersion
	source := fmt.Sprintf("builtin v%d (%s)", l.baseVersion, l.baseUpdated)
	if rm.exists {
		if doc, err := readGLMCatalogFile(l.remotePath); err != nil {
			slog.Warn("GLM 目录远程缓存读取失败，忽略该层", "path", l.remotePath, "error", err)
		} else {
			cur = mergeGLMEntries(cur, doc.Entries)
			if doc.HasVersion {
				version, source = doc.Version, "remote "+strconv.Itoa(doc.Version)
			}
		}
	}
	if ov.exists {
		if doc, err := readGLMCatalogFile(l.path); err != nil {
			slog.Warn("GLM 目录覆盖文件读取失败，忽略该层", "path", l.path, "error", err)
		} else {
			cur = mergeGLMEntries(cur, doc.Entries)
			if doc.HasVersion {
				version = doc.Version
			}
			source = "override"
		}
	}
	l.merged = cur
	l.effVersion, l.source = strconv.Itoa(version), source
	l.ovState, l.rmState = ov, rm
	return append([]ModelInfo(nil), cur...)
}

// builtinInfo 内嵌目录的（版本, 来源）描述。
func (l *glmCatalogLoader) builtinInfo() (string, string) {
	return strconv.Itoa(l.baseVersion), fmt.Sprintf("builtin v%d (%s)", l.baseVersion, l.baseUpdated)
}

// readGLMCatalogFile 读取并解析一个目录文件（覆盖/远程共用）。
func readGLMCatalogFile(path string) (glmCatalogDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return glmCatalogDoc{}, err
	}
	return parseGLMCatalog(data)
}

// baseEntries 懒解析内嵌 JSON（只执行一次）。内嵌文件随二进制发布且由
// 测试锚定，解析失败说明发布物损坏：记日志返回空目录。
func (l *glmCatalogLoader) baseEntries() []ModelInfo {
	if l.base != nil {
		return l.base
	}
	doc, err := parseGLMCatalog(glmCatalogJSON)
	if err != nil {
		slog.Error("内嵌 GLM 目录解析失败", "error", err)
		l.base = []ModelInfo{}
		return l.base
	}
	l.base = buildGLMModels(doc.Entries)
	l.baseVersion, l.baseUpdated = 2, doc.Updated
	if doc.HasVersion {
		l.baseVersion = doc.Version
	}
	return l.base
}

// buildGLMModels 条目 → ModelInfo（kind 省略时按引擎/名称分类兜底）。
func buildGLMModels(entries []glmCatalogEntry) []ModelInfo {
	models := make([]ModelInfo, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		id := strings.TrimSpace(e.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		m := ModelInfo{ID: id, OwnedBy: "glm", Kind: glmEntryKind(e, id)}
		e.applyTo(&m)
		models = append(models, m)
	}
	return models
}

// applyTo 把条目的元数据字段写进 ModelInfo：显式给出（非零值/非空，Free
// 为指针区分显式 false）才写，否则保留原值——覆盖合并与初建共用同一规则。
func (e glmCatalogEntry) applyTo(m *ModelInfo) {
	if e.Kind != "" {
		m.Kind = e.Kind
	}
	if e.ContextLength > 0 {
		m.ContextLength = e.ContextLength
	}
	if e.MaxOutput > 0 {
		m.MaxOutput = e.MaxOutput
	}
	if e.PriceIn != 0 {
		m.PriceIn = e.PriceIn
	}
	if e.PriceOut != 0 {
		m.PriceOut = e.PriceOut
	}
	if e.Currency != "" {
		m.Currency = e.Currency
	}
	if e.Unit != "" {
		m.Unit = e.Unit
	}
	if e.Free != nil {
		m.Free = *e.Free
	}
	if len(e.Caps) > 0 {
		m.Caps = append([]string(nil), e.Caps...)
	}
	if e.PriceNote != "" {
		m.PriceNote = e.PriceNote
	}
	if e.PointsIn != 0 {
		m.PointsIn = e.PointsIn
	}
	if e.PointsCached != 0 {
		m.PointsCached = e.PointsCached
	}
	if e.PointsOut != 0 {
		m.PointsOut = e.PointsOut
	}
	if e.PointsPeak != 0 {
		m.PointsPeak = e.PointsPeak
	}
}

// mergeGLMEntries 覆盖合并：同 ID 替换（字段显式给出才覆盖，否则保留下层
// 值），新 ID 追加尾部。
func mergeGLMEntries(base []ModelInfo, entries []glmCatalogEntry) []ModelInfo {
	merged := append([]ModelInfo(nil), base...)
	byID := make(map[string]int, len(merged))
	for i := range merged {
		byID[merged[i].ID] = i
	}
	for _, e := range entries {
		id := strings.TrimSpace(e.ID)
		if id == "" {
			continue
		}
		if idx, ok := byID[id]; ok {
			e.applyTo(&merged[idx])
			continue
		}
		byID[id] = len(merged)
		m := ModelInfo{ID: id, OwnedBy: "glm", Kind: glmEntryKind(e, id)}
		e.applyTo(&m)
		merged = append(merged, m)
	}
	return merged
}

// glmEntryKind 条目 kind：空时交给 ClassifyModelKind 统一判型。
func glmEntryKind(e glmCatalogEntry, id string) string {
	if e.Kind != "" {
		return e.Kind
	}
	return ClassifyModelKind(EngineGLM, id)
}

// glmCatalogPrice 在当前生效 GLM 目录中查 model 的目录价（stats.estimatePrice
// 的目录优先层）。normalized 为 normalizeModelID 归一后的模型 ID：精确匹配
// 优先；否则取目录 ID 前缀匹配中的最长前缀（与 modelPricing 长前缀在前
// 同口径）。条目须带价（free 或 price 非零）才返回 true；无价条目返回
// false，调用方回退内置定价表（现状不变）。
func glmCatalogPrice(normalized string) (modelPrice, bool) {
	models := glmStaticModels()
	hit := -1
	for i := range models {
		if models[i].ID == normalized {
			hit = i
			break
		}
	}
	if hit < 0 {
		best := ""
		for i := range models {
			if id := models[i].ID; len(id) > len(best) && strings.HasPrefix(normalized, id) {
				hit, best = i, id
			}
		}
	}
	if hit < 0 {
		return modelPrice{}, false
	}
	m := models[hit]
	if m.Free {
		return modelPrice{Currency: "CNY"}, true
	}
	if m.PriceIn == 0 && m.PriceOut == 0 {
		return modelPrice{}, false // 目录条目无价：估算回退内置表
	}
	return modelPrice{InputPerM: m.PriceIn, OutputPerM: m.PriceOut, Currency: m.Currency, Unit: m.Unit}, true
}

// glmCatalogInfo 返回当前生效目录的 schema 版本与来源描述
// （source 如 "builtin v2 (2026-09-02)" / "override" / "remote 2"），
// 供 GetModelCallStats 透传 ModelStatsSummary.CatalogVersion/CatalogSource
// （零新绑定）。
func glmCatalogInfo() (version, source string) {
	glmCatalog.mu.Lock()
	defer glmCatalog.mu.Unlock()
	glmCatalog.modelsLocked()
	return glmCatalog.effVersion, glmCatalog.source
}

// SetGLMCatalogPath 设置 GLM 目录覆盖文件路径（app 启动注入，照
// SetUsdCnyRate 先例；非绑定方法）。空=只用内嵌目录。
func (m *Manager) SetGLMCatalogPath(path string) {
	setGLMCatalogPath(path)
}

// glmCatalogModels 当前端点家族视角的 GLM 目录：在静态目录上按 coding
// 家族补 alias_of 注记（服务端旧名自动切换，见 glm_alias.go）；std 家族
// 不注记。TestConnection/fetchModels 的 GLM 分支统一走这里。
func (m *Manager) glmCatalogModels() []ModelInfo {
	models := glmStaticModels()
	if fam := m.GlmEndpointFamily(); fam == "coding" {
		for i := range models {
			if a := GlmAliasOf(fam, models[i].ID); a != "" {
				models[i].AliasOf = a
			}
		}
	}
	return models
}
