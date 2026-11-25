package app

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestGaeaDynamicToolSchemasValid 检查 boot.Build 动态注册的工具（skill/memory/
// ask/task）的 Schema 是否为合法 JSON。非法 schema 会在 bridge 发送消息时导致
// json.Marshal 失败（"invalid character '}' after top-level value"）。

func TestGaeaDynamicToolSchemasValid(t *testing.T) {
	bridge.SetClient(ai.NewClient(config.Load()))
	skStore := skill.New(skill.Options{DisableBuiltins: true})
	memSet := memory.Load(memory.Options{})
	_ = memSet // memory.Set 内部持有 Store

	reg := tool.NewRegistry()
	reg.Add(agent.NewAskTool())
	// 项目斜杠命令 2026-11-25 并入 run_skill（slash_command 下线）：带命令的
	// run_skill 也过一遍 schema 合法性。
	reg.Add(skill.WithRunSkillCommands(skill.NewRunSkillTool(skStore, nil), []skill.CommandEntry{
		{Name: "test", Description: "d", Render: func([]string) string { return "" }},
	}))
	reg.Add(skill.NewInstallSkillTool(skStore, nil))
	reg.Add(memory.NewRememberTool(memory.Store{}, nil))

	bad := 0
	for _, sch := range reg.Schemas() {
		if len(sch.Parameters) == 0 {
			continue
		}
		if !json.Valid(sch.Parameters) {
			t.Errorf("动态工具 %s 的 Schema 非法: %s", sch.Name, string(sch.Parameters))
			bad++
		}
	}
	if bad > 0 {
		t.Fatalf("%d 个动态工具 Schema 非法", bad)
	}
}
