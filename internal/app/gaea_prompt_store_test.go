package app

// t6 提示词工坊·线 B 测试（规格书 进度计划/gaea-prompt-workshop-t6-20260916.md
// §4.4 五组）：状态文件 roundtrip、Save 校验闸、引擎覆盖生效 e2e、
// substituteWordCount 双语法矩阵、Preview 渲染。引擎用 ../../prompts 真模板
// （create_chapter_cancel_test.go 先例）；DataRoot 用 GAEA_DATA_ROOT 指向
// t.TempDir()（config.DataRoot 同款来源，与收件箱先例一致）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/promptstore"
)

// newPromptWorkshopApp 构造工坊测试 App：真模板引擎 + 独立 DataRoot，
// 覆盖层按生产装配点（app.go New/SetPromptFS）同款挂接。
func newPromptWorkshopApp(t *testing.T) (*App, string) {
	t.Helper()
	dataRoot := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)
	// ResourceDir 指仓库根：基线引擎读 ../../prompts 与主引擎同源。
	cfg := &config.Config{ResourceDir: "../.."}
	a := &App{core: &core{cfg: cfg}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.applyPromptOverrides(a.writingState.eng)
	return a, dataRoot
}

// mustWorkshopTemplate 取真模板副本（键缺失直接失败）。
func mustWorkshopTemplate(t *testing.T, key string) prompt.Template {
	t.Helper()
	tmpl := prompt.NewEngine("../../prompts").Get(key)
	if tmpl == nil {
		t.Fatalf("缺少 %s 模板", key)
	}
	return *tmpl
}

// promptWorkshopSaveReq 构造 Save 请求 JSON（规格 §4.2 reqJSON 形状）；
// isActive 传 nil 序列化时省略，走「缺省 true」路径。
func promptWorkshopSaveReq(t *testing.T, tmpl prompt.Template, isActive *bool) string {
	t.Helper()
	content, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatalf("序列化模板: %v", err)
	}
	req := struct {
		IsActive    *bool           `json:"isActive,omitempty"`
		Category    string          `json:"category"`
		Description string          `json:"description"`
		Content     json.RawMessage `json:"content"`
	}{isActive, "测试分组", "测试覆盖说明", content}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("序列化保存请求: %v", err)
	}
	return string(b)
}

// findMetaRow 从清单中取指定键的行。
func findMetaRow(list []PromptTemplateMeta, key string) (PromptTemplateMeta, bool) {
	for _, m := range list {
		if m.Key == key {
			return m, true
		}
	}
	return PromptTemplateMeta{}, false
}

// ── 1. 状态文件 roundtrip（保存→List/Get 生效→Reset 回落→坏文件容错）────

func TestPromptStore_StateFileRoundtrip(t *testing.T) {
	a, dataRoot := newPromptWorkshopApp(t)

	// 初始态：全部内置、无覆盖行
	if m, ok := findMetaRow(a.PromptTemplateList(), "create-chapter"); !ok {
		t.Fatalf("List 缺少 create-chapter")
	} else if m.Source != "builtin" || m.HasOverride || m.Version != 0 || m.UpdatedAt != 0 {
		t.Fatalf("初始态应为内置零版本: %+v", m)
	}

	// 保存覆盖（isActive 缺省 → true）
	tmpl := mustWorkshopTemplate(t, "create-chapter")
	tmpl.System = "【覆盖标记】" + tmpl.System
	res, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, tmpl, nil))
	if err != nil || !res.Saved {
		t.Fatalf("保存覆盖失败: res=%+v err=%v", res, err)
	}
	if res.Version != 1 {
		t.Fatalf("首次保存版本应为 1: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "prompt_overrides.json")); err != nil {
		t.Fatalf("状态文件应已落盘: %v", err)
	}

	// List 生效：来源 override、覆盖行分组/说明优先、版本与保存时间可见
	m, ok := findMetaRow(a.PromptTemplateList(), "create-chapter")
	if !ok {
		t.Fatalf("保存后 List 缺少 create-chapter")
	}
	if m.Source != "override" || !m.HasOverride || !m.OverrideActive {
		t.Fatalf("覆盖激活态 Meta 不符: %+v", m)
	}
	if m.Category != "测试分组" || m.Description != "测试覆盖说明" {
		t.Fatalf("覆盖行 Category/Description 应优先: %+v", m)
	}
	if m.Version != 1 || m.UpdatedAt <= 0 {
		t.Fatalf("覆盖行版本/保存时间不符: %+v", m)
	}

	// 二次保存：版本自增（Upsert 语义）
	res2, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, tmpl, nil))
	if err != nil || res2.Version != 2 {
		t.Fatalf("二次保存版本应自增: res=%+v err=%v", res2, err)
	}

	// Get 生效：Template=覆盖内容、Base=磁盘/embed 基线
	detail, err := a.PromptTemplateGet("create-chapter")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(detail.Template.System, "【覆盖标记】") {
		t.Errorf("生效模板应为覆盖内容: %s", detail.Template.System)
	}
	if strings.Contains(detail.Base.System, "【覆盖标记】") {
		t.Errorf("Base 应为磁盘/embed 基线（不含覆盖）: %s", detail.Base.System)
	}
	if detail.Meta.Source != "override" || detail.Meta.Version != 2 {
		t.Errorf("Detail.Meta 应反映覆盖态: %+v", detail.Meta)
	}

	// Reset 回落 + 幂等
	if err := a.PromptTemplateReset("create-chapter"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := a.PromptTemplateReset("create-chapter"); err != nil {
		t.Fatalf("Reset 应幂等成功: %v", err)
	}
	detail2, err := a.PromptTemplateGet("create-chapter")
	if err != nil {
		t.Fatalf("Reset 后 Get: %v", err)
	}
	if strings.Contains(detail2.Template.System, "【覆盖标记】") {
		t.Errorf("Reset 后应回落内置: %s", detail2.Template.System)
	}
	if detail2.Meta.Source != "builtin" || detail2.Meta.HasOverride {
		t.Errorf("Reset 后 Meta 应回内置: %+v", detail2.Meta)
	}

	// 坏文件容错：损坏状态文件 → 空表（面板不炸，全部回落内置）
	if err := os.WriteFile(filepath.Join(dataRoot, "prompt_overrides.json"), []byte("{不是合法JSON"), 0o644); err != nil {
		t.Fatalf("写坏文件: %v", err)
	}
	a.writingState.invalidatePromptOverrides() // 作废缓存，逼下一次读取走坏文件路径
	if m, ok := findMetaRow(a.PromptTemplateList(), "create-chapter"); !ok {
		t.Fatalf("坏文件下 List 仍应完整")
	} else if m.Source != "builtin" || m.HasOverride {
		t.Fatalf("坏文件应容错回空表（全部内置）: %+v", m)
	}
	if got := a.writingState.eng.Get("create-chapter"); got == nil || strings.Contains(got.System, "【覆盖标记】") {
		t.Fatalf("坏文件下引擎应回落内置模板")
	}
}

// ── 2. Save 校验闸 + 未知键拒绝 + isActive=false 回落 ─────────────────────

func TestPromptStore_SaveValidationGate(t *testing.T) {
	a, dataRoot := newPromptWorkshopApp(t)

	// error 闸：System trim 后为空 → 不落盘、带回 Issues（不返回 Go error）
	bad := mustWorkshopTemplate(t, "create-chapter")
	bad.System = "   "
	bad.Task = "正文任务。"
	res, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, bad, nil))
	if err != nil {
		t.Fatalf("校验 error 应带回 Issues 而非 Go error: %v", err)
	}
	if res.Saved {
		t.Fatalf("存在 error 时不得落盘: %+v", res)
	}
	hasEmptySystem := false
	for _, is := range res.Issues {
		if is.Severity == "error" && is.Code == "empty-system" {
			hasEmptySystem = true
		}
	}
	if !hasEmptySystem {
		t.Fatalf("Issues 应含 empty-system error: %+v", res.Issues)
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "prompt_overrides.json")); !os.IsNotExist(err) {
		t.Errorf("校验失败不应写状态文件 (err=%v)", err)
	}

	// warn 闸：未声明变量 + 旧语法占位 → 落盘且带回 warn
	warned := mustWorkshopTemplate(t, "create-chapter")
	warned.System = "正文不少于{{word_count}}字，参考{{mystery}}与旧写法{word_count}。"
	warned.Parameters = []string{"word_count"}
	res2, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, warned, nil))
	if err != nil || !res2.Saved {
		t.Fatalf("warn 不阻断应落盘: res=%+v err=%v", res2, err)
	}
	codes := map[string]bool{}
	for _, is := range res2.Issues {
		if is.Severity != "warn" {
			t.Errorf("本例只应产生 warn: %+v", is)
		}
		codes[is.Code] = true
	}
	if !codes["undeclared-var"] || !codes["legacy-brace"] {
		t.Errorf("Issues 应含 undeclared-var 与 legacy-brace: %+v", res2.Issues)
	}

	// 未知模板键拒绝
	if _, err := a.PromptTemplateSave("no-such-template", promptWorkshopSaveReq(t, warned, nil)); err == nil || !strings.Contains(err.Error(), "未知模板键") {
		t.Fatalf("未知模板键应被拒绝: %v", err)
	}

	// isActive=false 保存：覆盖行存在但不参与三级解析（引擎 Get 回落内置）
	base := mustWorkshopTemplate(t, "rewrite-chapter")
	inactive := false
	res3, err := a.PromptTemplateSave("rewrite-chapter", promptWorkshopSaveReq(t, base, &inactive))
	if err != nil || !res3.Saved {
		t.Fatalf("停用保存应成功: res=%+v err=%v", res3, err)
	}
	if got := a.writingState.eng.Get("rewrite-chapter"); got == nil || got.System != base.System {
		t.Errorf("停用覆盖应回落内置 (got=%v)", got)
	}
	if m, ok := findMetaRow(a.PromptTemplateList(), "rewrite-chapter"); !ok {
		t.Fatalf("List 缺少 rewrite-chapter")
	} else if !m.HasOverride || m.OverrideActive || m.Source != "builtin" {
		t.Fatalf("停用覆盖 Meta 应为 builtin + HasOverride: %+v", m)
	}
}

// ── 3. 引擎覆盖生效 e2e（writingState.eng.Get 层，不起真模型）────────────

func TestPromptStore_OverrideReachesEngine(t *testing.T) {
	a, _ := newPromptWorkshopApp(t)

	tmpl := mustWorkshopTemplate(t, "create-chapter")
	tmpl.System = "【e2e覆盖】仅系统段被改写。"
	if _, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, tmpl, nil)); err != nil {
		t.Fatalf("保存覆盖: %v", err)
	}
	// 保存即失效缓存：已挂覆盖闭包的生产引擎取到覆盖标记（三级解析第一级）
	sys := a.writingState.eng.Get("create-chapter").BuildSystemPrompt("")
	if !strings.Contains(sys, "【e2e覆盖】") {
		t.Fatalf("引擎 Get 应命中覆盖层: %s", sys)
	}
	// 覆盖按键隔离：未覆盖键不受影响
	if got := a.writingState.eng.Get("rewrite-chapter"); got == nil || strings.Contains(got.System, "【e2e覆盖】") {
		t.Fatalf("未覆盖键不应受影响")
	}
	// Reset 后引擎立即回落（闭包惰性重读空表）
	if err := a.PromptTemplateReset("create-chapter"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if sys := a.writingState.eng.Get("create-chapter").BuildSystemPrompt(""); strings.Contains(sys, "【e2e覆盖】") {
		t.Fatalf("Reset 后引擎应回落内置: %s", sys)
	}
}

// ── 4. substituteWordCount 双语法矩阵（规格 §4.4）─────────────────────────

func TestSubstituteWordCount_DualSyntax(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"新语法", "不少于{{word_count}}字", "不少于3000字"},
		{"旧语法", "不少于{word_count}字", "不少于3000字"},
		{"混合", "{{word_count}}字，旧写法{word_count}字", "3000字，旧写法3000字"},
		{"无占位符", "保持原样的 5000 字标准", "保持原样的 5000 字标准"},
		{"新语法不残留单层花括号", "{{word_count}}", "3000"},
	}
	for _, c := range cases {
		if got := substituteWordCount(c.in, 3000); got != c.want {
			t.Errorf("%s: substituteWordCount(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// ── 5. Preview 渲染 + 未解析变量警告（规格 §4.4）─────────────────────────

func TestPromptStore_Preview(t *testing.T) {
	a, _ := newPromptWorkshopApp(t)

	// 草稿模板：清掉模板自带的占位/约束噪声，只留 System 两个 {{}} 变量
	// （对线 A 迁移 {word_count}→{{word_count}} 免疫）
	tmpl := mustWorkshopTemplate(t, "create-chapter")
	tmpl.System = "目标 {{alpha}} 与 {{beta}} 两变量。"
	tmpl.Task = "任务正文。"
	tmpl.Output = prompt.OutputDef{Format: "markdown", Description: "正文输出"}
	tmpl.Constraints = prompt.ConstraintDef{}
	content, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatalf("序列化草稿: %v", err)
	}
	req := `{"content":` + string(content) + `}`

	// 部分变量：命中替换、缺失保留原文并进 Warnings
	res, err := a.PromptTemplatePreview(req, `{"alpha":"甲"}`)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !strings.Contains(res.SystemPrompt, "目标 甲 与 {{beta}} 两变量。") {
		t.Errorf("命中变量应替换、缺失变量应保留原文: %q", res.SystemPrompt)
	}
	if len(res.Warnings) != 1 || res.Warnings[0] != "beta" {
		t.Errorf("Warnings 应只含未解析的 beta: %+v", res.Warnings)
	}

	// varsJSON 空串：全部变量未解析
	res2, err := a.PromptTemplatePreview(req, "")
	if err != nil {
		t.Fatalf("Preview 空变量: %v", err)
	}
	if len(res2.Warnings) != 2 {
		t.Errorf("空变量应产生两个警告: %+v", res2.Warnings)
	}

	// 坏请求 / 坏变量格式报错
	if _, err := a.PromptTemplatePreview("not-json", ""); err == nil {
		t.Errorf("坏 reqJSON 应报错")
	}
	if _, err := a.PromptTemplatePreview(req, "not-json"); err == nil {
		t.Errorf("坏 varsJSON 应报错")
	}
}

// ── 附：覆盖闭包与缓存失效的并发安全冒烟（-race 下有意义）────────────────

func TestPromptStore_OverrideSnapshotInvalidation(t *testing.T) {
	a, _ := newPromptWorkshopApp(t)
	tmpl := mustWorkshopTemplate(t, "create-chapter")
	tmpl.System = "【失效冒烟】"
	if _, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, tmpl, nil)); err != nil {
		t.Fatalf("保存: %v", err)
	}
	// 缓存失效后 snapshot 惰性重读（同一数据源可见同一行）
	entries := a.writingState.promptOverridesSnapshot()
	if row := promptstore.ActiveOverride(entries, "create-chapter"); row == nil {
		t.Fatalf("快照应含激活覆盖行: %+v", entries)
	}
	if err := a.PromptTemplateReset("create-chapter"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if row := promptstore.ActiveOverride(a.writingState.promptOverridesSnapshot(), "create-chapter"); row != nil {
		t.Fatalf("Reset 后快照不应再有激活覆盖行: %+v", row)
	}
}

// ── t6-C2 模板包导入导出（规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md
// §6）：导出→清状态→导入→覆盖回魂 round-trip、三态分支接线、防御路径。────

func TestPromptBundle_RoundTripRestore(t *testing.T) {
	a, dataRoot := newPromptWorkshopApp(t)
	tmpl := mustWorkshopTemplate(t, "create-chapter")
	tmpl.System = "【搬家】覆盖正文——round-trip 应回魂"
	if _, err := a.PromptTemplateSave("create-chapter", promptWorkshopSaveReq(t, tmpl, nil)); err != nil {
		t.Fatalf("保存: %v", err)
	}

	bundleJSON, err := a.PromptBundleExport()
	if err != nil {
		t.Fatalf("导出: %v", err)
	}
	var bundle promptstore.ExportBundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		t.Fatalf("包应为合法 JSON: %v", err)
	}
	if bundle.Version != 1 || bundle.Statistics.Total == 0 {
		t.Fatalf("包头/统计不对: %+v", bundle.Statistics)
	}
	if bundle.Statistics.Customized != 1 {
		t.Fatalf("应恰有一行自定义: %+v", bundle.Statistics)
	}

	// 模拟换机：清掉状态文件再导入，覆盖层应逐字节回魂。
	if err := os.Remove(filepath.Join(dataRoot, "prompt_overrides.json")); err != nil {
		t.Fatalf("清状态: %v", err)
	}
	a.writingState.invalidatePromptOverrides()
	if row := promptstore.ActiveOverride(a.writingState.promptOverridesSnapshot(), "create-chapter"); row != nil {
		t.Fatal("清状态后不应有覆盖行")
	}
	res, err := a.PromptBundleImport(bundleJSON)
	if err != nil {
		t.Fatalf("导入: %v", err)
	}
	if !res.Applied || res.Statistics.CreatedOrUpdate != 1 {
		t.Fatalf("导入统计不对: %+v", res.Statistics)
	}
	got := a.writingState.eng.Get("create-chapter")
	if got == nil || !strings.Contains(got.System, "【搬家】覆盖正文") {
		t.Fatalf("引擎应解析到回魂覆盖: %v", got)
	}
	if row := promptstore.ActiveOverride(a.writingState.promptOverridesSnapshot(), "create-chapter"); row == nil {
		t.Fatal("覆盖行应回魂")
	}
}

func TestPromptBundle_ImportSameAsBaselineRemovesOverride(t *testing.T) {
	a, _ := newPromptWorkshopApp(t)
	// 本地有覆盖行，导入一份「内容=基线」的内置快照 → 三态判 kept：覆盖行被删。
	base := mustWorkshopTemplate(t, "chapter-summary")
	tmpl := base
	tmpl.System = "本地临时覆盖"
	if _, err := a.PromptTemplateSave("chapter-summary", promptWorkshopSaveReq(t, tmpl, nil)); err != nil {
		t.Fatalf("保存: %v", err)
	}
	bundle := promptstore.ExportBundle{Version: 1, Templates: []promptstore.BundleTemplate{
		{Key: "chapter-summary", Content: base, IsActive: true},
	}}
	raw, _ := json.Marshal(bundle)
	res, err := a.PromptBundleImport(string(raw))
	if err != nil {
		t.Fatalf("导入: %v", err)
	}
	if res.Statistics.KeptSystemDefault != 1 || res.Statistics.CreatedOrUpdate != 0 {
		t.Fatalf("三态应判 kept: %+v", res.Statistics)
	}
	if row := promptstore.ActiveOverride(a.writingState.promptOverridesSnapshot(), "chapter-summary"); row != nil {
		t.Fatal("kept 分支应删本地覆盖行")
	}
}

func TestPromptBundle_ImportDefenses(t *testing.T) {
	a, _ := newPromptWorkshopApp(t)
	if _, err := a.PromptBundleImport(""); err == nil {
		t.Error("空串应报错")
	}
	if _, err := a.PromptBundleImport("not-json"); err == nil {
		t.Error("坏 JSON 应报错")
	}
	badVer, _ := json.Marshal(promptstore.ExportBundle{Version: 9})
	if _, err := a.PromptBundleImport(string(badVer)); err == nil {
		t.Error("version!=1 应报错")
	}
	// 空包：Applied=false 不落盘不算错。
	empty, _ := json.Marshal(promptstore.ExportBundle{Version: 1})
	res, err := a.PromptBundleImport(string(empty))
	if err != nil || res.Applied {
		t.Errorf("空包应静默: res=%+v err=%v", res, err)
	}
	// 坏模板行（error 级校验）跳行不阻断，未知键不建行。
	bad := prompt.Template{Name: "create-chapter", System: " ", Task: "x"}
	bundle := promptstore.ExportBundle{Version: 1, Templates: []promptstore.BundleTemplate{
		{Key: "create-chapter", Content: bad, IsCustomized: true},
		{Key: "no-such-key", Content: prompt.Template{System: "s", Task: "t"}, IsCustomized: true},
	}}
	raw, _ := json.Marshal(bundle)
	res, err = a.PromptBundleImport(string(raw))
	if err != nil {
		t.Fatalf("导入: %v", err)
	}
	if res.Statistics.SkippedInvalid != 1 || res.Statistics.SkippedUnknown != 1 || res.Statistics.CreatedOrUpdate != 0 {
		t.Fatalf("两道闸统计不对: %+v", res.Statistics)
	}
	if promptstore.ActiveOverride(a.writingState.promptOverridesSnapshot(), "create-chapter") != nil {
		t.Fatal("坏模板行不应落盘")
	}
}
