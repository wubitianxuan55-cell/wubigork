package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 对标拆解（novel-dissect 蒸馏刀）测试：复用剧情分支的 httptest SSE 桩环境 ──

// dissectReply 构造一段合规拆解 JSON 回复。
func dissectReply(payoff, hookType string) string {
	b := `{"payoff":"` + payoff + `","hook_type":"` + hookType + `","hook_detail":"债主堵门讨薪","emotion_arc":"压抑→爆发→余悸","info_push":["主角拿到账本","与盟友决裂","火力全开"],"craft_formulas":["条件：谈判破裂→手法：用一件旧物收尾→效果：情绪落地","条件：连续压抑→手法：短句连排爆发→效果：爽点兑现"]}`
	return b
}

// seedDissectChapter 写一章可拆解的正文。
func seedDissectChapter(t *testing.T, env *plotBranchTestEnv, num int) {
	t.Helper()
	body := strings.Repeat("他攥着账本站在门口，雨声敲着铁皮棚，一句话都说不出来。", 4)
	if err := env.pm.WriteChapter(num, body); err != nil {
		t.Fatalf("写章节: %v", err)
	}
}

func TestNovelDissectChapter_ParsePersistResume(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{dissectReply("兑现第2章的耳光承诺", "危机")})
	seedDissectChapter(t, env, 1)

	res, err := env.a.NovelDissectChapter(1, false)
	if err != nil {
		t.Fatalf("NovelDissectChapter: %v", err)
	}
	if res["hook_type"] != "危机" || res["payoff"] != "兑现第2章的耳光承诺" {
		t.Fatalf("拆解字段不对: %+v", res)
	}
	infoPush, _ := res["info_push"].([]string)
	if len(infoPush) != 3 {
		t.Fatalf("info_push 应 3 条, got %d", len(infoPush))
	}
	if _, err := os.Stat(filepath.Join(env.pm.Dir, "dissections", "001.json")); err != nil {
		t.Fatalf("拆解档未落盘: %v", err)
	}

	// 断点续跑：force=false 命中存量，零 LLM 调用（桩计数不变）
	callsBefore := *env.calls
	res2, err := env.a.NovelDissectChapter(1, false)
	if err != nil {
		t.Fatalf("续跑: %v", err)
	}
	if res2["payoff"] != res["payoff"] || *env.calls != callsBefore {
		t.Fatalf("force=false 应直接返回存量且不再调 LLM: calls %d→%d", callsBefore, *env.calls)
	}
}

func TestNovelDissectChapter_InvalidHookTypeFallsBack(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{`{"payoff":"无","hook_type":"超级大钩子","hook_detail":"","emotion_arc":"","info_push":[],"craft_formulas":[]}`})
	seedDissectChapter(t, env, 1)

	res, err := env.a.NovelDissectChapter(1, false)
	if err != nil {
		t.Fatalf("NovelDissectChapter: %v", err)
	}
	if res["hook_type"] != "无" {
		t.Fatalf("非法 hook_type 应落保守档「无」, got %v", res["hook_type"])
	}
}

func TestNovelDissectChapter_ShortChapterRejected(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{dissectReply("x", "危机")})
	if err := env.pm.WriteChapter(1, "太短"); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	if _, err := env.a.NovelDissectChapter(1, false); err == nil || !strings.Contains(err.Error(), "太短") {
		t.Fatalf("过短章节应被拒: %v", err)
	}
}

func TestNovelDissectReport_AggregatesDeterministically(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{dissectReply("兑现承诺", "危机")})
	seedDissectChapter(t, env, 1)
	if _, err := env.a.NovelDissectChapter(1, false); err != nil {
		t.Fatalf("NovelDissectChapter: %v", err)
	}

	// 直接落一档第 2 章（反转钩子，与第 1 章共享一条公式→验证去重）
	if err := os.MkdirAll(filepath.Join(env.pm.Dir, "dissections"), 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	entry2 := `{"chapter":2,"payoff":"","hook_type":"反转","hook_detail":"","emotion_arc":"平静→惊","info_push":[],"craft_formulas":["条件：谈判破裂→手法：用一件旧物收尾→效果：情绪落地"],"dissected_at":"2026-10-24T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(env.pm.Dir, "dissections", "002.json"), []byte(entry2), 0o644); err != nil {
		t.Fatalf("写第二档: %v", err)
	}

	rep, err := env.a.NovelDissectReport()
	if err != nil {
		t.Fatalf("NovelDissectReport: %v", err)
	}
	if rep["chapters_dissected"] != 2 {
		t.Fatalf("应聚合 2 章, got %v", rep["chapters_dissected"])
	}
	hookDist, _ := rep["hook_dist"].(map[string]int)
	if hookDist["危机"] != 1 || hookDist["反转"] != 1 {
		t.Fatalf("钩子分布不对: %+v", hookDist)
	}
	formulas, _ := rep["formulas"].([]string)
	if len(formulas) != 2 {
		t.Fatalf("公式去重后应 2 条, got %d: %v", len(formulas), formulas)
	}
	// 剧情线按章号升序（timeline 是 handler 内部切片，经 JSON 圆桌断言）
	tlJSON, err := json.Marshal(rep["timeline"])
	if err != nil {
		t.Fatalf("timeline 序列化: %v", err)
	}
	var tl []struct {
		Chapter int `json:"chapter"`
	}
	if err := json.Unmarshal(tlJSON, &tl); err != nil {
		t.Fatalf("timeline 反序列化: %v", err)
	}
	if len(tl) != 2 || tl[0].Chapter != 1 || tl[1].Chapter != 2 {
		t.Fatalf("剧情线应按章号升序: %+v", tl)
	}
}

func TestNovelDissectList_EmptyAndLoaded(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{dissectReply("x", "留白")})
	seedDissectChapter(t, env, 3)

	lst, err := env.a.NovelDissectList()
	if err != nil {
		t.Fatalf("NovelDissectList: %v", err)
	}
	if entries := lst["entries"].([]map[string]interface{}); len(entries) != 0 {
		t.Fatalf("空项目应返回空清单, got %d", len(entries))
	}

	if _, err := env.a.NovelDissectChapter(3, false); err != nil {
		t.Fatalf("NovelDissectChapter: %v", err)
	}
	lst, err = env.a.NovelDissectList()
	if err != nil {
		t.Fatalf("NovelDissectList: %v", err)
	}
	entries := lst["entries"].([]map[string]interface{})
	if len(entries) != 1 || entries[0]["chapter"] != 3 {
		t.Fatalf("清单应有第 3 章: %+v", entries)
	}
}
