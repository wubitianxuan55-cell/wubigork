package novelreview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dimOf 取指定维度结果（不存在则失败）。
func dimOf(t *testing.T, rep *Report, id string) Dimension {
	t.Helper()
	for _, d := range rep.Dimensions {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("维度 %s 不在报告中（%v）", id, rep.Dimensions)
	return Dimension{}
}

// mustReview 跑一次评审（出错即失败）。
func mustReview(t *testing.T, text, platform string, opts Options) *Report {
	t.Helper()
	rep, err := Review(text, platform, opts)
	if err != nil {
		t.Fatalf("Review(%s) 失败: %v", platform, err)
	}
	return rep
}

// TestRubricAssetSanity 数据资产自检：四档齐备、严重度合法、词表非空。
func TestRubricAssetSanity(t *testing.T) {
	ids := PlatformIDs()
	for _, want := range []string{"general", "fanqie", "qidian", "zhihu"} {
		found := false
		for _, id := range ids {
			if id == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("内置 rubric 缺档位 %s（现有 %v）", want, ids)
		}
	}
	r := currentRubric()
	if len(r.Advisories) < 3 {
		t.Fatalf("黄金三问缺失: %v", r.Advisories)
	}
	for _, p := range r.Platforms {
		for dim, sev := range p.Severity {
			if !severityValues[sev] {
				t.Fatalf("档位 %s 维度 %s 严重度非法: %s", p.ID, dim, sev)
			}
		}
	}
	if labels := PlatformLabels(); labels["fanqie"] != "番茄小说" {
		t.Fatalf("档位中文名不符: %v", labels)
	}
}

// TestUnknownPlatformFallsBackGeneral 未知档位回落 general；空文本报错。
func TestUnknownPlatformFallsBackGeneral(t *testing.T) {
	rep := mustReview(t, "他抬头看了一眼。\n「走吧。」他说。\n门外有人在等。", "不存在的档位", Options{})
	if rep.Platform != "general" {
		t.Fatalf("未知档位应回落 general，实际 %s", rep.Platform)
	}
	if _, err := Review("   \n\n  ", "general", Options{}); err == nil {
		t.Fatal("空文本应报错")
	}
}

// TestOverrideFileValidates 覆盖文件：非法内容拒绝且不换出，合法内容整体替换。
func TestOverrideFileValidates(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"platforms":[{"id":"x","label":"X","form":"chapter","chapterWordsMin":10,"chapterWordsMax":5,"longParagraphChars":1,"openingHookParagraphs":1,"severity":{"nope":"S1"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadRubricFile(bad); err == nil {
		t.Fatal("非法覆盖文件应报错")
	}
	if ids := PlatformIDs(); len(ids) < 4 {
		t.Fatalf("非法覆盖不应换出资产: %v", ids)
	}

	good := filepath.Join(dir, "good.json")
	content := `{
	  "meta": {"source": "test", "spec": "test", "version": "test"},
	  "advisories": ["a", "b", "c"],
	  "markers": {"conflict": ["打"], "emotion": ["笑"], "suspense": ["?"]},
	  "platforms": [{
	    "id": "general", "label": "测试档", "form": "chapter",
	    "chapterWordsMin": 100, "chapterWordsMax": 200,
	    "longParagraphChars": 50, "openingHookParagraphs": 1,
	    "dialogRatioMin": 0.0, "dialogRatioMax": 1.0,
	    "emotionPerKilo": 0.0, "emotionGapWarn": 99999, "emotionGapFail": 99999,
	    "ellipsisPerKiloWarn": 99.0, "paraUniformRatioWarn": 0.99,
	    "severity": {"length_band": "S1"}
	  }]
	}`
	if err := os.WriteFile(good, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadRubricFile(good); err != nil {
		t.Fatalf("合法覆盖文件应加载成功: %v", err)
	}
	if labels := PlatformLabels(); labels["general"] != "测试档" {
		t.Fatalf("覆盖未生效: %v", labels)
	}
	// 复位内置资产，避免影响同包其它用例。
	rubric.Store(mustLoadEmbeddedRubric())
	// 不存在的文件=无覆盖，返回 nil。
	if err := LoadRubricFile(filepath.Join(dir, "absent.json")); err != nil {
		t.Fatalf("缺省覆盖文件应静默: %v", err)
	}
}

// TestLengthDimension 字数区间：超区间 warn、严重偏离 fail（严重度随档位）。
func TestLengthDimension(t *testing.T) {
	short := strings.Repeat("他走进房间然后坐下。", 20) // ~200 字
	rep := mustReview(t, short, "fanqie", Options{})
	d := dimOf(t, rep, "length_band")
	if d.Verdict != verdictFail {
		t.Fatalf("番茄 200 字应判 fail，实际 %s（%s）", d.Verdict, d.Detail)
	}
	if d.Severity != "S2" {
		t.Fatalf("番茄档位 length 严重度应为 S2，实际 %s", d.Severity)
	}
	mid := strings.Repeat("他走进房间然后坐下。", 120) // ~1200 字
	rep = mustReview(t, mid, "fanqie", Options{})
	if got := dimOf(t, rep, "length_band").Verdict; got != verdictWarn {
		t.Fatalf("番茄 1200 字应判 warn，实际 %s", got)
	}
}

// TestOpeningHookDimension 开篇钩子：纯描写 fail / 台词 pass / 悬念 warn。
func TestOpeningHookDimension(t *testing.T) {
	plain := "夜色很深。\n风从窗缝里钻进来。\n桌上的茶已经凉透了。\n" + strings.Repeat("他坐着。", 30)
	if got := dimOf(t, mustReview(t, plain, "general", Options{}), "opening_freshness").Verdict; got != verdictFail {
		t.Fatalf("纯描写开场应 fail，实际 %s", got)
	}
	dialog := "「你终于来了。」\n他抬起头。\n" + strings.Repeat("屋里的钟走了一格。", 30)
	if got := dimOf(t, mustReview(t, dialog, "general", Options{}), "opening_freshness").Verdict; got != verdictPass {
		t.Fatalf("台词开场应 pass，实际 %s", got)
	}
	suspense := "抽屉里的钥匙是新的。\n这很反常，他出差前明明锁好了。\n他站在门口没动。\n" + strings.Repeat("楼道里的声控灯灭了。", 20)
	if got := dimOf(t, mustReview(t, suspense, "general", Options{}), "opening_freshness").Verdict; got != verdictWarn {
		t.Fatalf("只有悬念铺垫应 warn，实际 %s", got)
	}
}

// TestEndingHookAndTrailer 章尾钩子与预告式收尾（含 S1 → REJECT）。
func TestEndingHookAndTrailer(t *testing.T) {
	body := strings.Repeat("他把单据按顺序理好，放进文件袋。\n", 60)
	plained := body + "他把文件袋放进抽屉，关上了门。"
	rep := mustReview(t, plained, "general", Options{})
	if got := dimOf(t, rep, "hook_expectation").Verdict; got != verdictWarn {
		t.Fatalf("平铺结尾应 warn，实际 %s", got)
	}
	questioned := body + "门外是谁在等他？"
	if got := dimOf(t, mustReview(t, questioned, "general", Options{}), "hook_expectation").Verdict; got != verdictPass {
		t.Fatalf("疑问结尾应 pass，实际 %s", got)
	}
	trailer := body + "属于他的反击，才刚刚开始。"
	rep = mustReview(t, trailer, "general", Options{})
	d := dimOf(t, rep, "trailer_ending")
	if d.Verdict != verdictFail || d.Severity != "S1" {
		t.Fatalf("预告式收尾应 fail/S1，实际 %s/%s", d.Verdict, d.Severity)
	}
	if len(d.Evidence) == 0 {
		t.Fatal("预告式收尾应给原文证据")
	}
	if rep.Verdict != VerdictReject {
		t.Fatalf("含 S1 应判 REJECT，实际 %s", rep.Verdict)
	}
}

// TestEmotionDensityDimension 情绪节点密度：长平直段 fail / 密度达标 pass。
func TestEmotionDensityDimension(t *testing.T) {
	flat := strings.Repeat("他走进房间看了看四周然后坐下。", 200) // 3000 字零命中
	rep := mustReview(t, flat, "general", Options{})
	d := dimOf(t, rep, "emotion_curve")
	if d.Verdict != verdictFail {
		t.Fatalf("3000 字无情绪节点应 fail，实际 %s（%s）", d.Verdict, d.Detail)
	}
	if len(d.Evidence) == 0 {
		t.Fatal("长平直段应给证据位置")
	}
	rich := strings.Repeat("他攥紧拳头，喉咙发紧，笑了一下。\n", 120)
	if got := dimOf(t, mustReview(t, rich, "general", Options{}), "emotion_curve").Verdict; got != verdictPass {
		t.Fatalf("高密度文本应 pass，实际 %s", got)
	}
}

// TestDashAndPunctuation 破折号 blocking + 省略号密度 + 数字区间豁免。
func TestDashAndPunctuation(t *testing.T) {
	text := "他停下——没有回头。\n" + strings.Repeat("雨还在下。\n", 40)
	rep := mustReview(t, text, "general", Options{})
	d := dimOf(t, rep, "dash_usage")
	if d.Verdict != verdictFail || len(d.Evidence) == 0 {
		t.Fatalf("破折号应 fail 且带证据，实际 %s/%d", d.Verdict, len(d.Evidence))
	}
	rangeText := "1990—2000年他都在南方。\n" + strings.Repeat("那年冬天很冷。\n", 40)
	if got := dimOf(t, mustReview(t, rangeText, "general", Options{}), "dash_usage").Verdict; got != verdictPass {
		t.Fatalf("数字区间不应算破折号，实际 %s", got)
	}
	ell := strings.Repeat("他张了张嘴……又忍住。\n", 30)
	if got := dimOf(t, mustReview(t, ell, "general", Options{}), "punctuation_rhythm").Verdict; got == verdictPass {
		t.Fatalf("高密度省略号不应 pass，实际 %s", got)
	}
}

// TestPerspectiveDimension 盐言档位第一人称：第三人称主导 → fail（S1）。
func TestPerspectiveDimension(t *testing.T) {
	third := strings.Repeat("他看着她，她也看着他。他转身，她低下头。", 20)
	rep := mustReview(t, third, "zhihu", Options{})
	d := dimOf(t, rep, "perspective_consistency")
	if d.Verdict != verdictFail || d.Severity != "S1" {
		t.Fatalf("第三人称主导应 fail/S1，实际 %s/%s", d.Verdict, d.Severity)
	}
	first := strings.Repeat("我看着她，她也看着我。我转身，她低下头。", 20)
	if got := dimOf(t, mustReview(t, first, "zhihu", Options{}), "perspective_consistency").Verdict; got != verdictPass {
		t.Fatalf("第一人称应 pass，实际 %s", got)
	}
}

// TestProtagonistAndLever 主角存在感（含 skip 口径）与金手指提及。
func TestProtagonistAndLever(t *testing.T) {
	text := strings.Repeat("老陈把账本合上，叹了口气。\n", 40)
	rep := mustReview(t, text, "qidian", Options{ProtagonistNames: []string{"林深"}})
	d := dimOf(t, rep, "protagonist_presence")
	if d.Verdict != verdictFail {
		t.Fatalf("主角缺席应 fail，实际 %s（%s）", d.Verdict, d.Detail)
	}
	rep = mustReview(t, text, "qidian", Options{})
	d = dimOf(t, rep, "protagonist_presence")
	if d.Verdict != verdictSkip {
		t.Fatalf("无主角名应 skip，实际 %s", d.Verdict)
	}
	if !strings.Contains(d.Detail, "角色库") {
		t.Fatalf("skip 应说明原因，实际 %q", d.Detail)
	}
	lever := dimOf(t, mustReview(t, text, "qidian", Options{}), "lever_mention")
	if lever.Verdict != verdictWarn {
		t.Fatalf("起点档无金手指提及应 warn，实际 %s", lever.Verdict)
	}
}

// TestWordcountExprDimension 「这五个字」类表述的字数与实际核对。
func TestWordcountExprDimension(t *testing.T) {
	bad := "「今天不回家」这三个字，他念了一遍。\n" + strings.Repeat("窗外的雨没有停。\n", 40)
	d := dimOf(t, mustReview(t, bad, "general", Options{}), "stated_length_accuracy")
	if d.Verdict != verdictWarn || !strings.Contains(d.Detail, "实际 5 字") {
		t.Fatalf("字数表述不符应 warn 并报实际字数，实际 %s（%s）", d.Verdict, d.Detail)
	}
	good := "「别回头」这三个字，他念了一遍。\n" + strings.Repeat("窗外的雨没有停。\n", 40)
	if got := dimOf(t, mustReview(t, good, "general", Options{}), "stated_length_accuracy").Verdict; got != verdictPass {
		t.Fatalf("字数相符应 pass，实际 %s", got)
	}
}

// TestVerdictAggregation 结论门槛：无 S1/S2 且 S3<3 → APPROVE；S2 → CONCERNS。
func TestVerdictAggregation(t *testing.T) {
	rep := mustReview(t, goodChapter(), "general", Options{})
	if rep.Verdict != VerdictApprove {
		var bad []string
		for _, d := range rep.Dimensions {
			if d.Verdict == verdictWarn || d.Verdict == verdictFail {
				bad = append(bad, d.ID+"="+d.Verdict+"/"+d.Severity+"("+d.Detail+")")
			}
		}
		t.Fatalf("合格章节应 APPROVE，实际 %s：%v", rep.Verdict, bad)
	}
	if rep.Counts["S1"] != 0 || rep.Counts["S2"] != 0 {
		t.Fatalf("合格章节不该有 S1/S2：%v", rep.Counts)
	}
	if len(rep.Advisories) != 3 {
		t.Fatalf("报告应带黄金三问，实际 %v", rep.Advisories)
	}

	// 一处 S2（开篇纯描写）→ CONCERNS。
	plainOpen := "夜色很深。\n风从窗缝里钻进来。\n桌上的茶已经凉透了。\n" + strings.Repeat("他把单据理好放进文件袋，又看了一眼窗外。\n", 60) + "门外是谁在等他？"
	rep = mustReview(t, plainOpen, "general", Options{})
	if rep.Verdict != VerdictConcerns {
		t.Fatalf("含 S2 应判 CONCERNS，实际 %s（%v）", rep.Verdict, rep.Counts)
	}
}

// goodChapter 构造一批「合格章节」夹具：开篇有冲突与台词、章尾有疑问、
// 情绪节点分布均匀、段落长短交错、无破折号/无预告腔。
func goodChapter() string {
	lines := []string{
		"林深推开门，屋里的说话声停了。",
		"「你迟到了。」老陈把账本合上，“最后一次机会。”",
		"林深走过去，把三张单据摊在桌上。",
		"「这批货，谁签的字？」",
		"「我签的。」林深说。",
		"屋里没人接话。有人把椅子往后挪了半寸。",
		"老陈盯着他看了几秒，脸色沉下来。",
		"“签字的人要担责。”老陈的手指敲了两下桌面，“你担得起？”",
		"林深没回答，把最上面那张单据转了个方向。",
		"上面有一处涂改，红笔圈着三个数字。",
		"“这不是我写的。”他说。",
		"“那就查。”",
		"老陈笑了一声，笑意没到眼睛里。",
		"他起身去里屋，翻箱子的声音传出来。",
		"林深站在原地，攥着那张单据，指节发白。",
		"窗外传来卸货的叉车声，一下一下，压得很实。",
		"“找到了。”",
		"老陈把一沓复印件摔在桌上。",
		"纸页散开，露出下面的签名栏。",
		"“同一批货，两套单子。”老陈说，“你早就知道。”",
		"林深没吭声。",
		"他知道那份单子是谁做的，可他现在不能说。",
		"“给我三天。”他说。",
		"“一天。”",
		"“两天。”",
		"老陈看了他很久，最后把复印件推过来。",
		"“两天后这个点，我在这儿等你。”",
		"林深把复印件收进文件袋，转身出门。",
		"雨还在下，台阶上积了一层水。",
		"他站在雨里，拿出手机，翻到一个很久没拨过的号码。",
		"屏幕上那个名字亮着，他按了下去。",
		"响了三声，对面接了，没有人说话。",
		"“是我。”林深说。",
		"对面沉默了几秒，呼吸声很清楚。",
		"“你在哪儿？”他终于问。",
	}
	return strings.Join(lines, "\n")
}
