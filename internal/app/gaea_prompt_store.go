package app

// t6 提示词工坊·线 B（规格书 进度计划/gaea-prompt-workshop-t6-20260916.md §4）：
// 模板三级解析（全局覆盖 → 磁盘 JSON → embed JSON）的 App 接线层。
// 纯校验/归一/合并（NormalizeKey/Validate/ActiveOverride/Upsert/Remove）在
// internal/promptstore（线 A，零 IO 表驱动，taskinbox 同款分工）；本文件只做：
//   1. 状态文件持久化 <DataRoot>/prompt_overrides.json（taskinbox 同款容错读
//      + temp+rename 原子写——收件箱先例 gaea_task_inbox.go，覆盖丢得起，
//      不该为一个坏文件挡住整个工坊面板）；
//   2. 覆盖缓存与互斥（writingState 侧持有，save/reset 后失效、override 闭包
//      惰性重读，规格 §4.3）；
//   3. applyPromptOverrides 引擎接线 + 五个 App 绑定方法（规格 §4.2，签名与
//      JSON 标签逐字一致——前端线 C 与绑定再生都以它为准）。
// 纪律：模板正文语义内容零触碰；覆盖是「整模板替换」（MuMu 同款），基线
// （磁盘/embed）永远可对照可恢复。

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/promptstore"
)

// ── 状态文件持久化（<DataRoot>/prompt_overrides.json，规格 §4.1）──────────
// 结构 {"version":1,"templates":[Override]}，Override json 标签 camelCase；
// load 容错（缺失/损坏/nil 回空表恒非 nil）、save 原子（temp+rename，
// task_inbox 同款手法——收件箱先例）。

type promptOverrideFile struct {
	Version   int                    `json:"version"`
	Templates []promptstore.Override `json:"templates"`
}

func promptOverridesPath(dataRoot string) string {
	return filepath.Join(dataRoot, "prompt_overrides.json")
}

// loadPromptOverrides 读模板覆盖表；文件缺失/损坏/空表一律回空表
// （恒非 nil，调用方零判空）。
func loadPromptOverrides(dataRoot string) []promptstore.Override {
	b, err := os.ReadFile(promptOverridesPath(dataRoot))
	if err != nil {
		return []promptstore.Override{}
	}
	var f promptOverrideFile
	if err := json.Unmarshal(b, &f); err != nil || f.Templates == nil {
		return []promptstore.Override{}
	}
	return f.Templates
}

// savePromptOverrides 整写覆盖表（temp+rename 原子落盘，task_inbox 同款）。
func savePromptOverrides(dataRoot string, entries []promptstore.Override) error {
	if entries == nil {
		entries = []promptstore.Override{}
	}
	b, err := json.MarshalIndent(promptOverrideFile{Version: 1, Templates: entries}, "", "  ")
	if err != nil {
		return err
	}
	p := promptOverridesPath(dataRoot)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "prompt-overrides-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := fileutil.RenameWithRetry(tmpName, p); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// ── 覆盖缓存与互斥（规格 §4.3：App 侧持缓存，选侵入最小的 writingState）──

// promptOverridesSnapshot 返回当前覆盖表（只读共享，恒非 nil）；未装载或已
// 失效时惰性重读状态文件（override 闭包每次引擎解析都走这里——save/reset
// 只作废缓存，不动引擎）。
func (ws *writingState) promptOverridesSnapshot() []promptstore.Override {
	ws.promptOverridesMu.Lock()
	defer ws.promptOverridesMu.Unlock()
	if !ws.promptOverridesLoaded {
		ws.promptOverridesCache = loadPromptOverrides(config.DataRoot())
		ws.promptOverridesLoaded = true
	}
	return ws.promptOverridesCache
}

// invalidatePromptOverrides 作废覆盖缓存（save/reset 落盘后调用；下一次
// 引擎覆盖解析或工坊读取时重读磁盘）。
func (ws *writingState) invalidatePromptOverrides() {
	ws.promptOverridesMu.Lock()
	defer ws.promptOverridesMu.Unlock()
	ws.promptOverridesCache = nil
	ws.promptOverridesLoaded = false
}

// notePromptEmbeddedFS 记录内置模板 FS 并作废基线引擎缓存（SetPromptFS
// 重建引擎时调用，基线引擎下次懒重建时拾取新 FS——磁盘优先、embed 兜底，
// 与主引擎同构但永不挂覆盖层）。
func (ws *writingState) notePromptEmbeddedFS(fsys fs.FS) {
	ws.promptOverridesMu.Lock()
	defer ws.promptOverridesMu.Unlock()
	ws.promptEmbeddedFS = fsys
	ws.promptBaseEng = nil
}

// promptBaselineEngine 返回磁盘/embed 基线引擎（无覆盖层，懒构造一次）。
// 工坊 Base 对比视图与 Meta 回落字段必须用它：主引擎 eng.Get 已是三级解析
// 后的「生效视图」，覆盖激活时取不到基线（规格 §4.2 Base 语义）。
func (ws *writingState) promptBaselineEngine() *prompt.Engine {
	if ws == nil || ws.core == nil || ws.core.cfg == nil {
		return nil
	}
	ws.promptOverridesMu.Lock()
	defer ws.promptOverridesMu.Unlock()
	if ws.promptBaseEng == nil {
		dir := filepath.Join(ws.core.cfg.ResourceDir, "prompts")
		if ws.promptEmbeddedFS != nil {
			ws.promptBaseEng = prompt.NewEngineWithEmbedded(dir, ws.promptEmbeddedFS)
		} else {
			ws.promptBaseEng = prompt.NewEngine(dir)
		}
	}
	return ws.promptBaseEng
}

// promptBaselineTemplate 取基线模板（磁盘/embed 两级，无覆盖层）；键缺于
// 基线时回 nil（调用方回落零值）。
func (ws *writingState) promptBaselineTemplate(name string) *prompt.Template {
	base := ws.promptBaselineEngine()
	if base == nil {
		return nil
	}
	return base.Get(name)
}

// applyPromptOverrides 给引擎挂 t6 全局覆盖层（规格 §4.3）：SetOverride 闭包
// 经 promptOverridesSnapshot 惰性读覆盖表 → ActiveOverride 命中（IsActive 且
// Key 匹配）返回其 Content 副本；未命中回 nil（引擎回落磁盘/embed 内置）。
// 生产装配点（app.go New 的 writingState 初始化与 SetPromptFS 引擎重建后）
// 都必须调用——重建引擎不重挂，覆盖层即丢失。
func (a *App) applyPromptOverrides(eng *prompt.Engine) {
	if eng == nil || a.writingState == nil {
		return
	}
	ws := a.writingState
	eng.SetOverride(func(name string) *prompt.Template {
		ov := promptstore.ActiveOverride(ws.promptOverridesSnapshot(), name)
		if ov == nil {
			return nil // 无覆盖行或已停用 → 引擎回落磁盘/embed
		}
		t := ov.Content // 返回副本，避免引擎持缓存 backing array 内指针
		return &t
	})
}

// findPromptOverrideRow 找键的覆盖行（无论启停；Upsert 保证同键至多一行）。
func findPromptOverrideRow(entries []promptstore.Override, key string) *promptstore.Override {
	for i := range entries {
		if entries[i].Key == key {
			return &entries[i]
		}
	}
	return nil
}

// ── 五个 App 绑定方法（规格 §4.2，签名/JSON 标签逐字一致）────────────────

// PromptTemplateMeta 模板清单行视图（列表不回传正文——MuMu 1.74MB 下发
// 教训，规格 §1 裁剪）。
type PromptTemplateMeta struct {
	Key            string `json:"key"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Source         string `json:"source"`         // "override" | "builtin"
	HasOverride    bool   `json:"hasOverride"`    // 覆盖行存在（无论启停）
	OverrideActive bool   `json:"overrideActive"` // 覆盖行存在且 IsActive
	Version        int    `json:"version"`        // 生效版本（覆盖=保存次数，内置=0）
	UpdatedAt      int64  `json:"updatedAt"`      // 覆盖最近保存毫秒；内置 0
}

// buildPromptTemplateMeta 组装单个键的清单行：Category/Description 优先取
// 覆盖行、缺省（空）回落内置模板字段（规格 §4.2 List 语义）；生效来源由
// 覆盖行是否激活决定。
func (a *App) buildPromptTemplateMeta(name string, entries []promptstore.Override) PromptTemplateMeta {
	meta := PromptTemplateMeta{Key: name, Source: "builtin"}
	if base := a.writingState.promptBaselineTemplate(name); base != nil {
		meta.Category = base.Category
		meta.Description = base.Description
	}
	if row := findPromptOverrideRow(entries, name); row != nil {
		meta.HasOverride = true
		if row.Category != "" {
			meta.Category = row.Category
		}
		if row.Description != "" {
			meta.Description = row.Description
		}
		meta.UpdatedAt = row.UpdatedAt // 覆盖行最近保存毫秒；无覆盖行为 0
		if row.IsActive {
			meta.Source = "override"
			meta.OverrideActive = true
			meta.Version = row.Version // 生效版本=保存次数；停用回落内置为 0
		}
	}
	return meta
}

// promptEngineHas 引擎是否有该模板键（Names 判断，规格 §4.2 Save 流程）。
func (a *App) promptEngineHas(key string) bool {
	if a.writingState == nil || a.writingState.eng == nil {
		return false
	}
	for _, n := range a.writingState.eng.Names() {
		if n == key {
			return true
		}
	}
	return false
}

// PromptTemplateList 模板清单：引擎 Names() 全量 × 覆盖表状态（零轮询，
// 前端打开面板才拉一次）。引擎未初始化回空表恒非 nil。
func (a *App) PromptTemplateList() []PromptTemplateMeta {
	out := []PromptTemplateMeta{}
	if a.writingState == nil || a.writingState.eng == nil {
		return out
	}
	entries := a.writingState.promptOverridesSnapshot()
	for _, name := range a.writingState.eng.Names() {
		out = append(out, a.buildPromptTemplateMeta(name, entries))
	}
	return out
}

// PromptTemplateDetail 模板详情视图：生效模板（覆盖激活=覆盖内容）与
// 磁盘/embed 基线（对比/恢复参照）并列。
type PromptTemplateDetail struct {
	Meta     PromptTemplateMeta `json:"meta"`
	Template prompt.Template    `json:"template"` // 生效模板（覆盖激活=覆盖内容）
	Base     prompt.Template    `json:"base"`     // 磁盘/embed 基线（对比/恢复参照）
}

// PromptTemplateGet 取单个模板详情。生效模板直接读主引擎 Get（线 A 三级
// 解析语义：覆盖命中优先）；基线读专用基线引擎（无覆盖层）。
func (a *App) PromptTemplateGet(key string) (PromptTemplateDetail, error) {
	normKey, err := promptstore.NormalizeKey(key)
	if err != nil {
		return PromptTemplateDetail{}, &appError{err.Error()}
	}
	if a.writingState == nil || a.writingState.eng == nil {
		return PromptTemplateDetail{}, &appError{"模板引擎未初始化"}
	}
	eff := a.writingState.eng.Get(normKey)
	if eff == nil {
		return PromptTemplateDetail{}, &appError{"未知模板键: " + normKey}
	}
	detail := PromptTemplateDetail{
		Meta:     a.buildPromptTemplateMeta(normKey, a.writingState.promptOverridesSnapshot()),
		Template: *eff,
	}
	if base := a.writingState.promptBaselineTemplate(normKey); base != nil {
		detail.Base = *base
	}
	return detail, nil
}

// PromptSaveResult 保存结果：error 阻断时 Saved=false 且 Issues 带回全部
// 校验问题；warn 不阻断（已保存也带回提示）。
type PromptSaveResult struct {
	Saved   bool                `json:"saved"`
	Issues  []promptstore.Issue `json:"issues"`  // 含 warn（已保存也带回提示）
	Version int                 `json:"version"` // 保存后版本；未保存 0
}

// promptTemplateSaveInput Save 请求形状（reqJSON 反序列化目标，规格 §4.2）。
// isActive 用 *bool：nil=缺省 true（前端不传即启用）。
type promptTemplateSaveInput struct {
	IsActive    *bool           `json:"isActive"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Content     prompt.Template `json:"content"`
}

// PromptTemplateSave 保存模板覆盖（整模板替换，MuMu 同款）。流程（规格 §4.2）：
// NormalizeKey → 引擎存在该键（Names 判断，否则 error「未知模板键」）→
// Validate：有 error 不落盘、原样带回 Issues；warn 落盘且带回 → Upsert
// （Version 自增、CreatedAt 保留）→ 原子落盘 → 失效覆盖缓存（引擎下次
// 解析惰性重读，即时生效于生成链路）。
func (a *App) PromptTemplateSave(key string, reqJSON string) (PromptSaveResult, error) {
	var in promptTemplateSaveInput
	if err := json.Unmarshal([]byte(reqJSON), &in); err != nil {
		return PromptSaveResult{Issues: []promptstore.Issue{}}, &appError{"模板保存请求格式不正确: " + err.Error()}
	}
	active := true // isActive 缺省 true（规格 §4.2）
	if in.IsActive != nil {
		active = *in.IsActive
	}
	normKey, err := promptstore.NormalizeKey(key)
	if err != nil {
		return PromptSaveResult{Issues: []promptstore.Issue{}}, &appError{err.Error()}
	}
	if !a.promptEngineHas(normKey) {
		return PromptSaveResult{Issues: []promptstore.Issue{}}, &appError{"未知模板键: " + normKey}
	}
	in.Content.Name = normKey // 覆盖行模板名与键对齐（前端草稿不带 Name）

	issues := promptstore.Validate(in.Content)
	if issues == nil {
		issues = []promptstore.Issue{}
	}
	for _, is := range issues {
		if is.Severity == "error" {
			// error 阻断：不落盘，原样带回 Issues（warn 不阻断，规格 §3.2）
			return PromptSaveResult{Saved: false, Issues: issues}, nil
		}
	}

	now := time.Now().UnixMilli()
	dataRoot := config.DataRoot()
	merged := promptstore.Upsert(loadPromptOverrides(dataRoot), promptstore.Override{
		Key:         normKey,
		Category:    in.Category,
		Description: in.Description,
		Content:     in.Content,
		IsActive:    active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, now)
	if err := savePromptOverrides(dataRoot, merged); err != nil {
		return PromptSaveResult{Issues: issues}, fmt.Errorf("写入模板覆盖状态文件失败: %w", err)
	}
	a.writingState.invalidatePromptOverrides() // 缓存失效 → 引擎覆盖解析即时重读

	version := 0
	if row := findPromptOverrideRow(merged, normKey); row != nil {
		version = row.Version // Upsert 后的最终版本（新键 1 / 同键自增）
	}
	slog.Info("提示词工坊：保存模板覆盖", "key", normKey, "version", version, "active", active)
	return PromptSaveResult{Saved: true, Issues: issues, Version: version}, nil
}

// PromptTemplateReset 删覆盖行 → 回落磁盘/embed 内置；覆盖行不存在也幂等
// 成功（不落盘）。
func (a *App) PromptTemplateReset(key string) error {
	normKey, err := promptstore.NormalizeKey(key)
	if err != nil {
		return &appError{err.Error()}
	}
	if a.writingState == nil {
		return &appError{"模板引擎未初始化"}
	}
	dataRoot := config.DataRoot()
	merged, removed := promptstore.Remove(loadPromptOverrides(dataRoot), normKey)
	if !removed {
		return nil // 幂等：无覆盖行即已是内置
	}
	if err := savePromptOverrides(dataRoot, merged); err != nil {
		return fmt.Errorf("写入模板覆盖状态文件失败: %w", err)
	}
	a.writingState.invalidatePromptOverrides()
	slog.Info("提示词工坊：恢复内置模板", "key", normKey)
	return nil
}

// PromptPreviewResult 预览结果：渲染后的 system 段 + 未解析变量名警告。
type PromptPreviewResult struct {
	SystemPrompt string   `json:"systemPrompt"`
	Warnings     []string `json:"warnings"` // RenderPlaceholders 未解析变量名
}

// promptTemplatePreviewInput Preview 请求形状（reqJSON 反序列化目标，
// 规格 §4.2：{"content":{Template}}）。
type promptTemplatePreviewInput struct {
	Content prompt.Template `json:"content"`
}

// PromptTemplatePreview 预览未保存草稿：content 来自 reqJSON（不经引擎、
// 不落盘），varsJSON 可空串；渲染 BuildSystemPrompt("") 的 {{}} 占位
// （V1 只预览 system 段）。缺失变量保留原文并进 Warnings（RenderPlaceholders
// 语义，规格 §3.1）。
func (a *App) PromptTemplatePreview(reqJSON string, varsJSON string) (PromptPreviewResult, error) {
	var in promptTemplatePreviewInput
	if err := json.Unmarshal([]byte(reqJSON), &in); err != nil {
		return PromptPreviewResult{}, &appError{"模板预览请求格式不正确: " + err.Error()}
	}
	vars := map[string]string{}
	if strings.TrimSpace(varsJSON) != "" {
		if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
			return PromptPreviewResult{}, &appError{"预览变量格式不正确（需 map[string]string）: " + err.Error()}
		}
	}
	rendered, unresolved := prompt.RenderPlaceholders(in.Content.BuildSystemPrompt(""), vars)
	warnings := unresolved
	if warnings == nil {
		warnings = []string{}
	}
	return PromptPreviewResult{SystemPrompt: rendered, Warnings: warnings}, nil
}

// ── t6-C2 模板包导入导出（规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md
// §4）：线 A promptstore/bundle.go 纯函数的 App 装配层。导出只回 JSON 字符串
// （落盘走前端 saveExportBlob 双门，Q6）；导入回三态结果（选文件走前端
// pickFileAsFile 双门，Q7）。───────────────────────────────────────────

// PromptBundleExport 导出模板包：引擎 Names() 全量 × 覆盖快照 × 基线闭包 →
// BuildBundle（纯函数）→ MarshalIndent 明文 JSON 字符串。引擎未初始化报错。
func (a *App) PromptBundleExport() (string, error) {
	if a.writingState == nil || a.writingState.eng == nil {
		return "", &appError{"模板引擎未初始化"}
	}
	names := a.writingState.eng.Names()
	bundle := promptstore.BuildBundle(names, a.writingState.promptOverridesSnapshot(),
		a.writingState.promptBaselineTemplate, time.Now().UnixMilli())
	b, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", fmt.Errorf("模板包序列化失败: %w", err)
	}
	slog.Info("提示词工坊：导出模板包", "total", bundle.Statistics.Total, "customized", bundle.Statistics.Customized)
	return string(b), nil
}

// PromptBundleImport 导入模板包：解析（坏 JSON / version!=1 报错）→
// ImportBundle 三态决策（engineHas=引擎已知键、baseline=基线闭包）→
// Applied 才原子落盘 + 失效覆盖缓存。空包/全跳过 Applied=false 不落盘不算错。
func (a *App) PromptBundleImport(bundleJSON string) (promptstore.BundleImportResult, error) {
	if strings.TrimSpace(bundleJSON) == "" {
		return promptstore.BundleImportResult{}, &appError{"模板包内容为空"}
	}
	var bundle promptstore.ExportBundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		return promptstore.BundleImportResult{}, &appError{"模板包格式不正确: " + err.Error()}
	}
	if bundle.Version != 1 {
		return promptstore.BundleImportResult{}, &appError{fmt.Sprintf("不支持的模板包版本: %d", bundle.Version)}
	}
	if a.writingState == nil || a.writingState.eng == nil {
		return promptstore.BundleImportResult{}, &appError{"模板引擎未初始化"}
	}
	keySet := make(map[string]bool)
	for _, n := range a.writingState.eng.Names() {
		keySet[n] = true
	}
	merged, res := promptstore.ImportBundle(bundle, a.writingState.promptOverridesSnapshot(),
		func(k string) bool { return keySet[k] },
		a.writingState.promptBaselineTemplate, time.Now().UnixMilli())
	if res.Applied {
		dataRoot := config.DataRoot()
		if err := savePromptOverrides(dataRoot, merged); err != nil {
			return promptstore.BundleImportResult{}, fmt.Errorf("写入模板覆盖状态文件失败: %w", err)
		}
		a.writingState.invalidatePromptOverrides() // 缓存失效 → 引擎覆盖解析即时重读
	}
	slog.Info("提示词工坊：导入模板包", "total", res.Statistics.Total,
		"createdOrUpdate", res.Statistics.CreatedOrUpdate, "converted", res.Statistics.ConvertedToCustom,
		"kept", res.Statistics.KeptSystemDefault, "skipped",
		res.Statistics.SkippedInvalid+res.Statistics.SkippedUnknown+res.Statistics.SkippedDuplicate,
		"applied", res.Applied)
	return res, nil
}
