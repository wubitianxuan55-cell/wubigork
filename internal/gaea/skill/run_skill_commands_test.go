package skill

// run_skill_commands_test.go — 项目斜杠命令并入 run_skill 的合同（2026-11-25 办公
// 工具梳理二批 B2，slash_command 工具下线）。钉：
//   - 命令按名展开、参数透传、回执框成「现在照做的指令」；
//   - 首部斜杠与 [..] 标记容错；name="list"（不分大小写）列技能与命令；
//   - 同名时技能优先（并入前 run_skill 的既有行为不变）；
//   - unknown 报错同时点名两边；命令展开与取清单不计入技能用量。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runSkillArgs(name, args string) json.RawMessage {
	raw, _ := json.Marshal(map[string]string{"name": name, "arguments": args})
	return raw
}

func sampleCommands() []CommandEntry {
	return []CommandEntry{
		{Name: "review", Description: "review the diff", ArgHint: "[path]",
			Render: func(a []string) string { return "REVIEW " + strings.Join(a, ",") }},
		{Name: "/git:commit", Description: "commit",
			Render: func(a []string) string { return "COMMIT" }},
	}
}

func TestRunSkillCommandExpandsWithArgs(t *testing.T) {
	tl := WithRunSkillCommands(
		NewRunSkillTool(New(Options{HomeDir: t.TempDir(), DisableBuiltins: true}), nil), sampleCommands())
	out, err := tl.Execute(context.Background(), runSkillArgs("review", "a b"))
	if err != nil {
		t.Fatalf("命令应可达: %v", err)
	}
	for _, want := range []string{"Expanded /review", "follow these instructions now", "REVIEW a,b"} {
		if !strings.Contains(out, want) {
			t.Fatalf("回执缺 %q:\n%s", want, out)
		}
	}
	// 条目名带首部斜杠、调用也带：两端都归一。
	out, err = tl.Execute(context.Background(), runSkillArgs("/git:commit", ""))
	if err != nil || !strings.Contains(out, "COMMIT") {
		t.Fatalf("首部斜杠应归一: out=%q err=%v", out, err)
	}
	// [..] 标记容错（模型常把索引里的标记一起抄进来）。
	out, err = tl.Execute(context.Background(), runSkillArgs("review [prompt]", "x"))
	if err != nil || !strings.Contains(out, "REVIEW x") {
		t.Fatalf("标记应被剥掉: out=%q err=%v", out, err)
	}
}

func TestRunSkillListNamesSkillsAndCommands(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, home, ".gaea/skills/note.md", "---\ndescription: take a note\n---\nDo the thing.")
	tl := WithRunSkillCommands(NewRunSkillTool(New(Options{HomeDir: home, DisableBuiltins: true}), nil), sampleCommands())
	for _, name := range []string{"list", "LIST", "/list"} {
		out, err := tl.Execute(context.Background(), runSkillArgs(name, ""))
		if err != nil {
			t.Fatalf("list(%q): %v", name, err)
		}
		for _, want := range []string{"note", "/review", "[path]", "review the diff", "/git:commit"} {
			if !strings.Contains(out, want) {
				t.Fatalf("list(%q) 缺 %q:\n%s", name, want, out)
			}
		}
	}
	// 没有命令的工作区：如实说没有，不报错。
	bare := NewRunSkillTool(New(Options{HomeDir: home, DisableBuiltins: true}), nil)
	out, err := bare.Execute(context.Background(), runSkillArgs("list", ""))
	if err != nil || !strings.Contains(out, "(none in this workspace)") {
		t.Fatalf("无命令应如实说明: out=%q err=%v", out, err)
	}
}

func TestRunSkillSkillWinsOverSameNameCommand(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, home, ".gaea/skills/dup.md", "---\ndescription: dup skill\n---\nSKILL-BODY")
	tl := WithRunSkillCommands(NewRunSkillTool(New(Options{HomeDir: home, DisableBuiltins: true}), nil),
		[]CommandEntry{{Name: "dup", Render: func([]string) string { return "FROM-COMMAND" }}})
	out, err := tl.Execute(context.Background(), runSkillArgs("dup", ""))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "SKILL-BODY") || strings.Contains(out, "FROM-COMMAND") {
		t.Fatalf("同名应按技能跑（并入前 run_skill 行为不变）: %s", out)
	}
}

func TestRunSkillUnknownNamesBothSides(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, home, ".gaea/skills/note.md", "---\ndescription: take a note\n---\nDo the thing.")
	tl := WithRunSkillCommands(NewRunSkillTool(New(Options{HomeDir: home, DisableBuiltins: true}), nil), sampleCommands())
	_, err := tl.Execute(context.Background(), runSkillArgs("nope", ""))
	if err == nil {
		t.Fatal("unknown 应报错")
	}
	for _, want := range []string{`unknown skill "nope"`, "note", "review"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("unknown 报错应同时点名技能与命令，缺 %q: %v", want, err)
		}
	}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{}`)); err == nil ||
		!strings.Contains(err.Error(), `name="list"`) {
		t.Fatalf("缺 name 应报错并提示 name=list: %v", err)
	}
}

func TestRunSkillCountsAsSkillUse(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, home, ".gaea/skills/note.md", "---\ndescription: take a note\n---\nDo the thing.")
	tl := WithRunSkillCommands(NewRunSkillTool(New(Options{HomeDir: home, DisableBuiltins: true}), nil), sampleCommands())
	rs, ok := tl.(*runSkillTool)
	if !ok {
		t.Fatalf("WithRunSkillCommands 应返回同一 run_skill 实例，得到 %T", tl)
	}
	yes := []string{"note", "note [tag]", "/note", "nosuch-skill"}
	no := []string{"review", "/review", "list", "LIST", "", "   "}
	for _, n := range yes {
		if !rs.CountsAsSkillUse(n) {
			t.Fatalf("%q 应计入技能用量", n)
		}
	}
	for _, n := range no {
		if rs.CountsAsSkillUse(n) {
			t.Fatalf("%q 不应计入技能用量", n)
		}
	}
	// 传进来的不是 run_skill：原样返回（装配方不必关心构造分支）。
	other := NewInstallSkillTool(nil, nil)
	if got := WithRunSkillCommands(other, sampleCommands()); got != other {
		t.Fatalf("非 run_skill 应原样返回，得到 %T", got)
	}
}
