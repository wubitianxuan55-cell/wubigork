package app

import (
	"os"
	"path/filepath"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// 设置中枢「文学提示词收编」绑对语义钉（2026-11-19）：
// 1. ga.cfg 未初始化（测试/早期构建）→ Get 恒报板块默认，绝不回落办公人格
//    （与 wenxueSystemPrompt() 同语义）；2. file 键优先——file 指向的档位
//    内容即 effective；3. 双空 = 代码默认。Set 体一行委托 gaeaApplyCfg
// （落盘+双引擎重建，既有测试面），此处不重复端到端。
func TestWenxuePromptGet_Semantics(t *testing.T) {
	a := shelfTestApp(t)
	orig := ga.cfg
	defer func() { ga.cfg = orig }()

	// nil cfg：仅板块默认
	ga.cfg = nil
	v, err := a.WenxuePromptGet()
	if err != nil {
		t.Fatalf("nil cfg get: %v", err)
	}
	if v.Effective != gaeaConfig.DefaultWenxueSystemPrompt {
		t.Fatalf("nil cfg effective = %q, want board default", v.Effective)
	}
	if v.Prompt != "" || v.File != "" {
		t.Fatalf("nil cfg should expose empty keys, got %+v", v)
	}

	// file 键优先：档位文件内容即 effective
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "persona.md")
	const persona = "你是测试文学人格"
	if err := os.WriteFile(promptFile, []byte(persona), 0o644); err != nil {
		t.Fatal(err)
	}
	ga.cfg = &gaeaConfig.Config{Agent: gaeaConfig.AgentConfig{
		WenxueSystemPrompt:     "inline-prompt",
		WenxueSystemPromptFile: promptFile,
	}}
	v, err = a.WenxuePromptGet()
	if err != nil {
		t.Fatalf("file-first get: %v", err)
	}
	if v.Prompt != "inline-prompt" || v.File != promptFile {
		t.Fatalf("raw keys not round-tripped: %+v", v)
	}
	if v.Effective != persona {
		t.Fatalf("effective = %q, want file content %q", v.Effective, persona)
	}

	// 双空：回落板块默认
	ga.cfg = &gaeaConfig.Config{}
	v, err = a.WenxuePromptGet()
	if err != nil {
		t.Fatalf("both-empty get: %v", err)
	}
	if v.Effective != gaeaConfig.DefaultWenxueSystemPrompt {
		t.Fatalf("both-empty effective = %q, want default", v.Effective)
	}

	// file 键指向缺档：用户显式配置的档位缺失必须硬报错（不静默吞）
	ga.cfg = &gaeaConfig.Config{Agent: gaeaConfig.AgentConfig{
		WenxueSystemPromptFile: filepath.Join(dir, "missing.md"),
	}}
	if _, err := a.WenxuePromptGet(); err == nil {
		t.Fatal("missing prompt file must be a hard error")
	}
}
