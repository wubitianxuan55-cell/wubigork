package modelengine

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// ── GLM 静态目录（内嵌 JSON + 可选覆盖文件热更新）─────────────
//
// 智谱官方 API 无模型列表端点，目录锚定 docs.bigmodel.cn「模型概览」。
// 内嵌 JSON 与覆盖文件同 schema（条目数组，id 必填、kind 可选——kind 省略时
// 由 ClassifyModelKind 统一分类兜底，glm-image 系为生图、glm-tts→tts、
// glm-asr→stt、embedding-3→embedding，其余为 llm）。
// 覆盖语义：同 ID 替换 + 新 ID 追加；解析失败静默回退内嵌（只打日志）。

//go:embed glm_catalog.json
var glmCatalogJSON []byte

// glmCatalogEntry 目录条目（内嵌与覆盖文件同 schema）。
type glmCatalogEntry struct {
	ID   string `json:"id"`
	Kind string `json:"kind,omitempty"`
}

// glmCatalogLoader 内嵌目录解析 + 覆盖文件 mtime 热更新状态。
// mtime 未变化时直接复用上次合并结果（零磁盘 IO）。
type glmCatalogLoader struct {
	mu      sync.Mutex
	base    []ModelInfo // 内嵌目录解析结果（只解析一次）
	merged  []ModelInfo // 上次合并结果缓存（nil=未算）
	modTime time.Time   // 覆盖文件上次加载时的修改时间（零值=未用覆盖）
	path    string      // 覆盖文件路径（空=只用内嵌）
}

var glmCatalog = &glmCatalogLoader{}

// setGLMCatalogPath 设置覆盖文件路径（app 启动按 ~/.gaea_config.json 的
// glm_catalog_path 注入一次；空=只用内嵌目录）。
func setGLMCatalogPath(path string) {
	p := strings.TrimSpace(path)
	glmCatalog.mu.Lock()
	defer glmCatalog.mu.Unlock()
	if p == glmCatalog.path {
		return
	}
	glmCatalog.path = p
	glmCatalog.merged = nil // 路径变化强制重算
	glmCatalog.modTime = time.Time{}
}

// glmStaticModels 返回 GLM 静态模型目录（内嵌 + 覆盖文件合并后的快照）。
// 每次调用检查覆盖文件 mtime，变化才重读合并；保持无参签名，
// TestConnection/fetchModels 的 GLM 分支调用点不变。
func glmStaticModels() []ModelInfo {
	return glmCatalog.models()
}

// models 返回目录快照（需持锁调用路径内部处理）。
func (l *glmCatalogLoader) models() []ModelInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	base := l.baseEntries()
	if l.path == "" {
		if l.merged == nil {
			l.merged = base
		}
		return append([]ModelInfo(nil), l.merged...)
	}
	st, err := os.Stat(l.path)
	if err != nil {
		// 覆盖文件不存在/不可读：回退内嵌（mtime 归零，文件重新出现时自动重载）
		if l.merged == nil || !l.modTime.IsZero() {
			l.modTime = time.Time{}
			l.merged = base
		}
		return append([]ModelInfo(nil), l.merged...)
	}
	if l.merged != nil && st.ModTime().Equal(l.modTime) {
		return append([]ModelInfo(nil), l.merged...)
	}
	// mtime 变化：重读合并；解析失败静默回退内嵌并记下 mtime，
	// 避免每次调用都重放同一条错误日志。
	l.merged = l.merge(base, l.path)
	l.modTime = st.ModTime()
	return append([]ModelInfo(nil), l.merged...)
}

// baseEntries 懒解析内嵌 JSON（只执行一次）。内嵌文件随二进制发布且由
// TestGLMStaticModels 锚定，解析失败说明发布物损坏：记日志返回空目录。
func (l *glmCatalogLoader) baseEntries() []ModelInfo {
	if l.base != nil {
		return l.base
	}
	var entries []glmCatalogEntry
	if err := json.Unmarshal(glmCatalogJSON, &entries); err != nil {
		slog.Error("内嵌 GLM 目录解析失败", "error", err)
		l.base = []ModelInfo{}
		return l.base
	}
	l.base = buildGLMModels(entries)
	return l.base
}

// merge 覆盖合并：同 ID 替换（kind 显式给出才覆盖，否则维持分类结果），
// 新 ID 追加尾部；读取/解析失败返回内嵌目录。
func (l *glmCatalogLoader) merge(base []ModelInfo, path string) []ModelInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("GLM 目录覆盖文件读取失败，回退内嵌目录", "path", path, "error", err)
		return base
	}
	var entries []glmCatalogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		slog.Warn("GLM 目录覆盖文件解析失败，回退内嵌目录", "path", path, "error", err)
		return base
	}
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
			if e.Kind != "" {
				merged[idx].Kind = e.Kind
			}
			continue
		}
		byID[id] = len(merged)
		merged = append(merged, ModelInfo{ID: id, OwnedBy: "glm", Kind: glmEntryKind(e, id)})
	}
	return merged
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
		models = append(models, ModelInfo{ID: id, OwnedBy: "glm", Kind: glmEntryKind(e, id)})
	}
	return models
}

// glmEntryKind 条目 kind：空时交给 ClassifyModelKind 统一判型。
func glmEntryKind(e glmCatalogEntry, id string) string {
	if e.Kind != "" {
		return e.Kind
	}
	return ClassifyModelKind(EngineGLM, id)
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
