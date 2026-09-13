package novelstyle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 模式级判定（oh-story T2 内核消费）──

func TestPatternsAssetSelfCheck(t *testing.T) {
	c := currentPatterns()
	if c == nil || len(c.neg) == 0 || len(c.expl) == 0 {
		t.Fatal("内置模式表应已装载且两族非空")
	}
}

func TestRuleNegationFlip_SameSentenceHighConfidence(t *testing.T) {
	text := "这不是失败，而是他计划的第一步。窗外的雨还在下。"
	score, err := ScoreTextNoRef(text)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, iss := range score.Issues {
		if strings.Contains(iss.Reason, "否定铺垫后肯定翻转") {
			if iss.Severity != "high" {
				t.Fatalf("门禁 B 同句应为 high: %+v", iss)
			}
			hit := string([]rune(text)[iss.Start:iss.End])
			if !strings.Contains(hit, "而是") {
				t.Fatalf("span 应覆盖翻转模式: %q", hit)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("同句否定翻转应命中: %+v", score.Issues)
	}

	// 无翻转的否定句不误报
	clean, _ := ScoreTextNoRef("他不是来了吗。窗外的雨还在下。")
	for _, iss := range clean.Issues {
		if strings.Contains(iss.Reason, "否定铺垫后肯定翻转") {
			t.Fatalf("普通否定句不应命中: %+v", iss)
		}
	}
}

func TestRuleExplanatoryMarkers_AdvisoryCapped(t *testing.T) {
	markers := []string{"之所以", "这意味着", "她不知道的是", "殊不知", "多年以后", "之所以", "这意味着"}
	var b strings.Builder
	for _, m := range markers {
		b.WriteString("他把这一切扛了下来。" + m + "，他不能退。\n")
	}
	score, err := ScoreTextNoRef(b.String())
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, iss := range score.Issues {
		if strings.Contains(iss.Reason, "解释腔标记") {
			n++
			if iss.Severity != "low" {
				t.Fatalf("解释腔为 advisory/low: %+v", iss)
			}
		}
	}
	if n == 0 || n > explanatoryMarkerCap {
		t.Fatalf("解释腔标记应命中且封顶 %d: got %d", explanatoryMarkerCap, n)
	}
}

// ── 书级白名单 ──

func TestApplyWhitelist_RescoresAndExempts(t *testing.T) {
	text := "这不是失败，而是他计划的第一步。窗外的雨还在下，他知道这一步意味着什么。"
	score, _ := ScoreTextNoRef(text)
	before := score.Score

	// 白名单整句授权 → 否定翻转 issue 被摘除，分数重算下降
	wl := []string{"这不是失败，而是他计划的第一步"}
	after := ApplyWhitelist(score, text, wl)
	if after >= before {
		t.Fatalf("豁免后分数应下降: before=%d after=%d", before, after)
	}
	for _, iss := range score.Issues {
		if strings.Contains(iss.Reason, "否定铺垫后肯定翻转") {
			t.Fatalf("授权片段的命中应被摘除: %+v", iss)
		}
	}

	// 无白名单=原样（分数不变）
	score2, _ := ScoreTextNoRef(text)
	if got := ApplyWhitelist(score2, text, nil); got != score2.Score {
		t.Fatal("空白名单不应改动分数")
	}
}

func TestLoadWhitelistFile_MissingNilAndCommentsSkipped(t *testing.T) {
	wl, err := LoadWhitelistFile(filepath.Join(t.TempDir(), "不存在"))
	if err != nil || wl != nil {
		t.Fatalf("无文件=(nil,nil): %v %v", wl, err)
	}
	p := filepath.Join(t.TempDir(), ".deslop-whitelist")
	os.WriteFile(p, []byte("# 注释\n\n  这不是失败，而是他计划的第一步  \n"), 0o644)
	wl, err = LoadWhitelistFile(p)
	if err != nil || len(wl) != 1 {
		t.Fatalf("注释与空行应剔除: %v %v", wl, err)
	}

	// 上游 style-resolution 注记格式兼容：条目上一行 `# 来源：…；用途：…` 整行忽略
	p2 := filepath.Join(t.TempDir(), ".deslop-whitelist")
	os.WriteFile(p2, []byte("# 来源：本书文风明确允许；用途：口头禅式停顿\n——且慢——\n"), 0o644)
	wl2, err := LoadWhitelistFile(p2)
	if err != nil || len(wl2) != 1 || wl2[0] != "——且慢——" {
		t.Fatalf("注记格式应兼容: %v %v", wl2, err)
	}
}

// ── 去味 × 白名单 ──

func TestDeSlopRewriteEx_WhitelistSkipsReplacement(t *testing.T) {
	text := "她的眸光流转，带着几分笑意。"
	// 无白名单：AI 词被替换
	out, rep, err := DeSlopRewriteEx(text, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	replaced := strings.Contains(out, "目光流转")
	if !replaced || len(rep.Changes) == 0 {
		t.Fatalf("无白名单应替换 AI 词: %q %+v", out, rep.Changes)
	}

	// 白名单含该片段：原样保留，AfterScore 同口径豁免
	wl := []string{"眸光流转"}
	out2, rep2, err := DeSlopRewriteEx(text, nil, wl)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "眸光流转") || len(rep2.Changes) != 0 {
		t.Fatalf("授权片段应原样保留: %q %+v", out2, rep2.Changes)
	}
}

// ── 模式表覆盖（惯例目录整体替换，fail-closed）──

func TestLoadPatternsFile_FailClosedAndReplace(t *testing.T) {
	dir := t.TempDir()

	// 坏正则整表拒绝，内置表不被换出
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`{"negationFlipHigh":["不是([，?）"],"explanatoryMarkers":[]}`), 0o644)
	if err := LoadPatternsFile(bad); err == nil {
		t.Fatal("坏正则应报错")
	}
	if len(currentPatterns().neg) == 0 {
		t.Fatal("校验失败不得换出（内置表应保持）")
	}

	// 合法覆盖：空两族=整体替换语义，规则不再命中
	empty := filepath.Join(dir, "empty.json")
	os.WriteFile(empty, []byte(`{"negationFlipHigh":["ZZZ"],"explanatoryMarkers":["QQQ"]}`), 0o644)
	if err := LoadPatternsFile(empty); err != nil {
		t.Fatalf("合法覆盖: %v", err)
	}
	score, _ := ScoreTextNoRef("这不是失败，而是开始。之所以如此。")
	for _, iss := range score.Issues {
		if strings.Contains(iss.Reason, "否定铺垫") || strings.Contains(iss.Reason, "解释腔") {
			t.Fatalf("覆盖后内置模式不应再命中: %+v", iss)
		}
	}

	// 文件缺失=保持当前表（不回退内置——替换语义可预测）
	if err := LoadPatternsFile(filepath.Join(dir, "缺失.json")); err != nil {
		t.Fatal(err)
	}
	if len(currentPatterns().neg) != 1 {
		t.Fatalf("缺失文件应保持覆盖表: %d", len(currentPatterns().neg))
	}
}
