package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func replayTestMsgs() []HistoryMessage {
	return []HistoryMessage{
		{Role: "system", Content: "系统提示应被跳过"},
		{Role: "user", Content: "把这周的会议纪要整理成周报"},
		{Role: "assistant", Content: ""},
		{Role: "tool", ToolName: "read_file", ToolArgs: `{"path":"minutes/..."}`, ToolID: "t1"},
		{Role: "tool_result", ToolOutput: "本周三次会议纪要内容…", ToolID: "t1"},
		{Role: "assistant", Content: "已整理完成，周报如下…"},
		{Role: "tool_result", ToolOutput: ""},
	}
}

func TestBuildSkillReplay(t *testing.T) {
	got := buildSkillReplay(replayTestMsgs())
	for _, want := range []string{
		"【用户】把这周的会议纪要整理成周报",
		"【工具】read_file",
		"【工具结果】本周三次会议纪要内容",
		"【助手】已整理完成",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("replay 缺少 %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "系统提示") {
		t.Error("system 角色不应进入回放")
	}
	if strings.Contains(got, "【工具结果】\n") {
		t.Error("空工具结果不应输出条目")
	}
}

func TestBuildSkillReplay_WindowAndTruncate(t *testing.T) {
	var msgs []HistoryMessage
	for i := 0; i < skillReplayMaxTurns+20; i++ {
		msgs = append(msgs, HistoryMessage{Role: "user", Content: strings.Repeat("早", skillReplayTurnMax+50)})
	}
	got := buildSkillReplay(msgs)
	lines := strings.Split(got, "\n")
	if len(lines) != skillReplayMaxTurns {
		t.Errorf("回放条数 = %d, want %d（应取末段）", len(lines), skillReplayMaxTurns)
	}
	if !strings.HasSuffix(lines[0], "…") {
		t.Errorf("超长消息应带省略号截断：%q…", lines[0][:20])
	}
	if r := len([]rune(strings.TrimSuffix(strings.TrimPrefix(lines[0], "【用户】"), "…"))); r != skillReplayTurnMax {
		t.Errorf("单条截断 = %d rune, want %d", r, skillReplayTurnMax)
	}
	if buildSkillReplay(nil) != "" {
		t.Error("空会话应返回空串")
	}
}

func TestParseSkillDraft(t *testing.T) {
	d := SkillDraft{
		Name:        "Weekly Report Gen",
		Description: "生成周报",
		Scenario:    "每周五汇总本周产出时使用",
		Steps:       []string{"  ", "读取本周会议纪要", "汇总完成事项", "输出三段式周报"},
		Cautions:    []string{"", "控制在三百字内"},
	}
	if err := parseSkillDraft(&d); err != nil {
		t.Fatalf("parseSkillDraft: %v", err)
	}
	if d.Name != "weekly-report-gen" {
		t.Errorf("name 归一 = %q, want weekly-report-gen", d.Name)
	}
	if len(d.Steps) != 3 || d.Steps[0] != "读取本周会议纪要" {
		t.Errorf("空步骤未剔除: %v", d.Steps)
	}
	if len(d.Cautions) != 1 {
		t.Errorf("空注意事项未剔除: %v", d.Cautions)
	}

	// 全中文名清洗后为空 → 报错
	bad := SkillDraft{Name: "周报生成", Steps: []string{"步骤"}}
	if err := parseSkillDraft(&bad); err == nil {
		t.Error("非法技能名应报错")
	}
	// 无步骤 → 报错
	noSteps := SkillDraft{Name: "ok-name", Steps: nil}
	if err := parseSkillDraft(&noSteps); err == nil {
		t.Error("缺步骤应报错")
	}
	// 步骤条数与单条长度上限
	var many []string
	for i := 0; i < skillDraftStepsMax+5; i++ {
		many = append(many, strings.Repeat("步", skillDraftStepMax+10))
	}
	capped := SkillDraft{Name: "ok-name", Steps: many}
	if err := parseSkillDraft(&capped); err != nil {
		t.Fatalf("parseSkillDraft(capped): %v", err)
	}
	if len(capped.Steps) != skillDraftStepsMax {
		t.Errorf("步骤上限 = %d, want %d", len(capped.Steps), skillDraftStepsMax)
	}
}

func TestRenderSkillDraftBody(t *testing.T) {
	body := renderSkillDraftBody(SkillDraft{
		Name: "weekly-report", Scenario: "周五汇总",
		Steps: []string{"读纪要", "写周报"}, Cautions: []string{"三百字内"},
	})
	for _, want := range []string{
		"# weekly-report 技能", "## 适用场景\n周五汇总", "## 操作步骤\n1. 读纪要\n2. 写周报",
		"## 注意事项\n- 三百字内", "## 调用方式",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("正文缺少 %q:\n%s", want, body)
		}
	}
	noCaution := renderSkillDraftBody(SkillDraft{Name: "x", Scenario: "s", Steps: []string{"a"}})
	if strings.Contains(noCaution, "注意事项") {
		t.Error("无注意事项时不应渲染该段")
	}
}

func TestGaeaSkillDraftSave(t *testing.T) {
	tmp := t.TempDir()
	old := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: tmp}
	defer func() { ga.cfg = old }()
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("HOME", tmp)

	a := &App{}
	res, err := a.GaeaSkillDraftSave(SkillDraft{
		Name: "weekly-report", Description: "周报生成",
		Scenario: "周五汇总本周产出",
		Steps:    []string{"读取本周会议纪要", "汇总完成事项", "输出三段式周报"},
		Cautions: []string{"控制在三百字内"},
	})
	if err != nil {
		t.Fatalf("GaeaSkillDraftSave: %v", err)
	}
	want := filepath.Join(tmp, ".gaea", "skills", "weekly-report", "SKILL.md")
	if res.Path != want {
		t.Fatalf("path = %q, want %q", res.Path, want)
	}
	b, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	content := string(b)
	for _, wantSub := range []string{
		"---\nname: weekly-report",
		"## 适用场景\n周五汇总本周产出",
		"## 操作步骤\n1. 读取本周会议纪要",
		"## 注意事项\n- 控制在三百字内",
	} {
		if !strings.Contains(content, wantSub) {
			t.Errorf("skill 缺少 %q:\n%s", wantSub, content)
		}
	}

	// 编辑后非法草稿（无场景）→ 报错不落盘
	if _, err := a.GaeaSkillDraftSave(SkillDraft{Name: "no-scen", Steps: []string{"a"}}); err == nil {
		t.Error("缺适用场景应报错")
	}
}

func TestGaeaSkillDraftFromSession_Guards(t *testing.T) {
	a := &App{}
	if _, err := a.GaeaSkillDraftFromSession(); err == nil {
		t.Fatal("办公引擎未初始化应报错")
	} else if !strings.Contains(err.Error(), "办公引擎未初始化") {
		t.Fatalf("错误信息不符: %v", err)
	}
}
