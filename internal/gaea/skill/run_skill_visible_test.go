package skill

// run_skill 可见集合收口测试（2026-11-15 众神殿首刀验收合同，pan_1794754879290
// 裁决五例）：① extras 内嵌件经 run_skill 返回正文；② 被 filter 拒名 fail-closed
// 且 unknown 提示名单不泄露隐藏名；③ 办公引擎（nil-filter）store 活读行为不变；
// ④ store/extras 同名不重复且序维持 store-first；⑤ subagent 型技能经 run_skill
// 仍进 runner 非 inline。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func runSkillOnce(t *testing.T, tl tool.Tool, name, args string) (string, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"name": name, "arguments": args})
	if err != nil {
		t.Fatal(err)
	}
	return tl.Execute(context.Background(), raw)
}

// ① extras 内嵌件经 run_skill 返回正文（wenxue-workbench 场景：索引可见、
// store 无此名——从集合构造的 run_skill 必须可达）。
func TestRunSkillFromSetResolvesExtrasBody(t *testing.T) {
	extras := []Skill{
		{Name: "wenxue-workbench", Description: "工作台纪律", Body: "# 图文工作台纪律\n\n写完一章必须回写：角色卡状态演进。"},
	}
	tl := NewRunSkillToolFromSet(extras, nil)
	out, err := runSkillOnce(t, tl, "wenxue-workbench", "")
	if err != nil {
		t.Fatalf("extras 技能应可达: %v", err)
	}
	if !strings.Contains(out, "写完一章必须回写") {
		t.Fatalf("run_skill 应返回内嵌正文，收到: %s", out)
	}
}

// ② 被 filter 拒名 fail-closed：unknown 且提示名单只列可见（不泄露隐藏名）。
func TestRunSkillFromSetFailsClosedOnHiddenName(t *testing.T) {
	visible := []Skill{{Name: "demo-review", Description: "评审", Body: "review body"}}
	tl := NewRunSkillToolFromSet(visible, nil)
	out, err := runSkillOnce(t, tl, "docx", "")
	if err == nil {
		t.Fatalf("隐藏技能应 unknown，收到: %s", out)
	}
	if !strings.Contains(err.Error(), `unknown skill "docx"`) {
		t.Fatalf("错误语义应变： %v", err)
	}
	if strings.Contains(err.Error(), "gongwen") {
		t.Fatalf("unknown 提示名单不应泄露隐藏名: %v", err)
	}
	if !strings.Contains(err.Error(), "demo-review") {
		t.Fatalf("unknown 提示名单应只列可见: %v", err)
	}
}

// ③ 办公引擎（nil-filter）现状：NewRunSkillTool(store) 活读 store——装配后
// 新写盘的技能立即可解析（活读语义=现状逐字节，boot 双 nil 分流保此路径）。
func TestRunSkillStoreLiveReadUnchanged(t *testing.T) {
	home := t.TempDir()
	st := New(Options{HomeDir: home, DisableBuiltins: true})
	tl := NewRunSkillTool(st, nil)
	// 装配（构造）时 store 还没有 gongwen——活读语义下写盘后即可用。
	writeSkill(t, home, ".gaea/skills/gongwen.md", "---\ndescription: 公文\n---\n公文正文")
	out, err := runSkillOnce(t, tl, "gongwen", "")
	if err != nil {
		t.Fatalf("store 活读应解析装配后新技能: %v", err)
	}
	if !strings.Contains(out, "公文正文") {
		t.Fatalf("应返回技能正文，收到: %s", out)
	}
	// unknown 提示名单来自 store 全量（办公无隐藏名概念，现状不变）。
	_, err = runSkillOnce(t, tl, "nosuch", "")
	if err == nil || !strings.Contains(err.Error(), "gongwen") {
		t.Fatalf("store 面 unknown 名单应含 store 技能: %v", err)
	}
}

// ④ store/extras 同名去重 store-first：VisibleSkills 单装配点，同名只留 store
// 版、序维持 store 先 extras 后；nil-filter（办公）产出=store.List() 原序原样。
func TestVisibleSkillsDedupesStoreFirst(t *testing.T) {
	storeList := []Skill{
		{Name: "beta", Body: "store-beta"},
		{Name: "alpha", Body: "store-alpha"},
	}
	extras := []Skill{
		{Name: "alpha", Body: "extra-alpha"}, // 与 store 同名 → store-first
		{Name: "gamma", Body: "extra-gamma"},
	}
	merged := VisibleSkills(storeList, extras, nil)
	if len(merged) != 3 {
		t.Fatalf("同名应去重（期望 3 条，得 %d）: %+v", len(merged), merged)
	}
	if merged[0].Name != "beta" || merged[1].Name != "alpha" || merged[2].Name != "gamma" {
		t.Fatalf("序应维持 store 先 extras 后: %+v", merged)
	}
	for _, sk := range merged {
		if sk.Name == "alpha" && sk.Body != "store-alpha" {
			t.Fatalf("同名应保留 store 版: %+v", sk)
		}
	}
	// filter 面：同名且被拒 → 整名消失（store 版定夺，extras 不复活）。
	denied := VisibleSkills(storeList, extras, func(name string) bool { return name != "alpha" })
	if len(denied) != 2 {
		t.Fatalf("被拒名应整条摘除（期望 2 条，得 %d）: %+v", len(denied), denied)
	}
	// 办公现状：extras 空 + nil filter = store.List() 原序原样。
	office := VisibleSkills(storeList, nil, nil)
	if len(office) != 2 || office[0].Name != "beta" || office[1].Name != "alpha" {
		t.Fatalf("办公（双 nil）应原序原样: %+v", office)
	}
}

// ⑤ subagent 型技能经 run_skill 仍进 runner 非 inline（RunAs 信息在完整 Skill
// 上保留——可见集合共享的是 Skill 不是字符串）。
func TestRunSkillFromSetSubagentGoesToRunner(t *testing.T) {
	called := false
	runner := func(ctx context.Context, sk Skill, task string) (string, error) {
		called = true
		if sk.Name != "demo-writing" || task != "写第三章" {
			t.Fatalf("runner 应收到完整 Skill 与任务: %s %q", sk.Name, task)
		}
		return "子代理完成", nil
	}
	visible := []Skill{{Name: "demo-writing", RunAs: RunSubagent, Body: "子代理卡"}}
	tl := NewRunSkillToolFromSet(visible, runner)
	out, err := runSkillOnce(t, tl, "demo-writing", "写第三章")
	if err != nil {
		t.Fatalf("subagent 技能应进 runner: %v", err)
	}
	if !called {
		t.Fatalf("应走 runner 非 inline，收到: %s", out)
	}
	if !strings.Contains(out, "子代理完成") {
		t.Fatalf("应返回 runner 结果，收到: %s", out)
	}
}
