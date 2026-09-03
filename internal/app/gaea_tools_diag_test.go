package app

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/command"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestGaeaDynamicToolSchemasValid 检查 boot.Build 动态注册的工具（skill/memory/
// command/ask/task）的 Schema 是否为合法 JSON。非法 schema 会在 bridge 发送
// 消息时导致 json.Marshal 失败（"invalid character '}' after top-level value"）。

func TestGaeaDynamicToolSchemasValid(t *testing.T) {
	bridge.SetClient(ai.NewClient(config.Load()))
	skStore := skill.New(skill.Options{DisableBuiltins: true})
	memSet := memory.Load(memory.Options{})
	_ = memSet // memory.Set 内部持有 Store

	reg := tool.NewRegistry()
	reg.Add(agent.NewAskTool())
	reg.Add(command.NewSlashCommandTool([]command.SlashEntry{{Name: "test", Description: "d", Render: func([]string) string { return "" }}}))
	reg.Add(skill.NewRunSkillTool(skStore, nil))
	reg.Add(skill.NewInstallSkillTool(skStore, nil))
	reg.Add(memory.NewPromoteSessionFactsTool())

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
