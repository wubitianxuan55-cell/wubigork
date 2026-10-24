package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 设定事实词典（novel-facts 蒸馏刀）测试 ──────────────────────────────

func newFactsEnv(t *testing.T) *plotBranchTestEnv {
	t.Helper()
	return newPlotBranchTestEnv(t, []string{`{}`})
}

func TestNovelFactsSaveList_RoundtripAndValidation(t *testing.T) {
	env := newFactsEnv(t)

	in := `[{"name":"倒计时","category":"金手指","first_chapter":1,"reader_known":"主角能看到别人头顶的倒计时","full_truth":"倒计时是死亡倒计时","reveal_chapter":80,"revealed":false},
	{"name":"北境失守","reader_known":"北境战报一直大捷","full_truth":"北境早已失守，战报是假的","reveal_chapter":0,"revealed":false}]`
	res, err := env.a.NovelFactsSave(in)
	if err != nil {
		t.Fatalf("NovelFactsSave: %v", err)
	}
	if res["saved"] != 2 {
		t.Fatalf("应落库 2 条, got %v", res["saved"])
	}

	lst, err := env.a.NovelFactsList()
	if err != nil {
		t.Fatalf("NovelFactsList: %v", err)
	}
	facts, _ := lst["facts"].([]novelFact)
	if len(facts) != 2 {
		t.Fatalf("清单应 2 条, got %d", len(facts))
	}
	if facts[0].ID == "" || facts[1].ID == "" {
		t.Fatalf("缺 ID 应自动补: %+v", facts)
	}
	if facts[0].ID == facts[1].ID {
		t.Fatalf("两条 ID 不应相同（同批自动补须错开）")
	}

	// 校验拒绝：空名称 / 空读者已知 / 负揭示章
	if _, err := env.a.NovelFactsSave(`[{"name":"","reader_known":"x"}]`); err == nil || !strings.Contains(err.Error(), "名称") {
		t.Fatalf("空名称应拒: %v", err)
	}
	if _, err := env.a.NovelFactsSave(`[{"name":"a","reader_known":""}]`); err == nil || !strings.Contains(err.Error(), "读者已知") {
		t.Fatalf("空读者已知应拒: %v", err)
	}
	if _, err := env.a.NovelFactsSave(`[{"name":"a","reader_known":"x","reveal_chapter":-1}]`); err == nil {
		t.Fatalf("负揭示章应拒")
	}
	if _, err := env.a.NovelFactsSave(`not-json`); err == nil || !strings.Contains(err.Error(), "解析") {
		t.Fatalf("坏 JSON 应拒: %v", err)
	}
}

func TestBuildFactsSection_HiddenDueRevealed(t *testing.T) {
	env := newFactsEnv(t)
	in := `[
	{"id":"f1","name":"倒计时","category":"金手指","first_chapter":1,"reader_known":"能看到倒计时","full_truth":"死亡倒计时","reveal_chapter":80,"revealed":false},
	{"id":"f2","name":"旧约","category":"秘密","reader_known":"两家世代交好","full_truth":"两家有血仇","reveal_chapter":10,"revealed":false},
	{"id":"f3","name":"身份","category":"身世","reader_known":"普通学徒","full_truth":"皇子","reveal_chapter":0,"revealed":true},
	{"id":"f4","name":"无隐藏","reader_known":"剑冢封印百年","full_truth":"","reveal_chapter":0,"revealed":false}]`
	if _, err := env.a.NovelFactsSave(in); err != nil {
		t.Fatalf("NovelFactsSave: %v", err)
	}

	// 第 12 章：f1 未到（80 章）→ 禁示；f2 已到（10 章）→ 可揭示；f3 已揭示；f4 无隐藏层
	got := buildFactsSection(env.pm, 12)
	for _, want := range []string{
		"## 设定事实（口径约束）",
		"第80章揭示，本章不得提前揭示",
		"已可揭示",
		"普通学徒",
		"剑冢封印百年",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("区段应含 %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "不得提前揭示") != 1 {
		t.Fatalf("仅未揭示且未到期的 f1 带禁示纪律（f2 第10章已到期转可揭示）:\n%s", got)
	}
	if !strings.Contains(got, "（已可揭示）：两家有血仇") {
		t.Fatalf("已到揭示章的 f2 应按可揭示口径:\n%s", got)
	}
	if strings.Contains(got, "皇子") == false {
		t.Fatalf("作者已标记揭示的 f3 完整真相应可见:\n%s", got)
	}

	// 空词典：无区段
	other := newPlotBranchTestEnv(t, []string{`{}`})
	if s := buildFactsSection(other.pm, 1); s != "" {
		t.Fatalf("空词典应返回空串, got:\n%s", s)
	}
}

func TestBuildChapterContextSections_FactsInjected(t *testing.T) {
	env := newFactsEnv(t)
	if _, err := env.a.NovelFactsSave(`[{"id":"f1","name":"倒计时","reader_known":"能看到倒计时","full_truth":"死亡倒计时","reveal_chapter":80}]`); err != nil {
		t.Fatalf("NovelFactsSave: %v", err)
	}
	got := buildChapterContextSections(env.pm, 5)
	if !strings.Contains(got, "设定事实（口径约束）") || !strings.Contains(got, "第80章揭示，本章不得提前揭示") {
		t.Fatalf("章节上下文应注入事实区段:\n%s", got)
	}
}

func TestNovelFactsFile_ShapeOnDisk(t *testing.T) {
	env := newFactsEnv(t)
	if _, err := env.a.NovelFactsSave(`[{"name":"a","reader_known":"x"}]`); err != nil {
		t.Fatalf("NovelFactsSave: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(env.pm.Dir, "facts.json"))
	if err != nil {
		t.Fatalf("读盘: %v", err)
	}
	var f novelFactsFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("盘上结构应合法: %v", err)
	}
	if f.Version != 1 || len(f.Facts) != 1 {
		t.Fatalf("盘上结构不对: %+v", f)
	}
}
