package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
)

// newFingerprintTestApp 构造带临时小说项目的测试 App（不依赖 LLM）。
func newFingerprintTestApp(t *testing.T) *App {
	t.Helper()
	a := newCharacterLibTestApp(t)
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	return a
}

// mustWriteChapter 写一章多段中文样本（重复段落至 ~1200 字，句式有起伏）。
func mustWriteChapter(t *testing.T, a *App, num int, seed string) {
	t.Helper()
	para := seed + "他攥紧了拳头，指节发白。夜风从窗缝里钻进来，吹得烛火一阵乱晃。" +
		"她笑了笑，没说话，只把茶碗往他面前推了推。「喝吧，凉了就不好喝了。」" +
		"远处传来更夫的梆子声，一下，又一下，敲得人心里发慌。\n\n"
	var sb strings.Builder
	for i := 0; i < 12; i++ {
		sb.WriteString(para)
	}
	if err := a.getPM().WriteChapter(num, sb.String()); err != nil {
		t.Fatalf("写章节 %d: %v", num, err)
	}
}

func TestNovelFingerprintStatus_NoRef(t *testing.T) {
	a := newFingerprintTestApp(t)
	st, err := a.NovelFingerprintStatus()
	if err != nil {
		t.Fatalf("未构建参考档应返回正常态: %v", err)
	}
	if st.Exists {
		t.Fatalf("未构建应 exists=false: %+v", st)
	}
}

func TestNovelFingerprintBuild_SampleTooSmall(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteChapter(t, a, 1, "第一章 只有一章。")
	if _, err := a.NovelFingerprintBuild(); err == nil || !strings.Contains(err.Error(), "样本不足") {
		t.Fatalf("样本不足应诚实拒绝: %v", err)
	}
	// 拒绝后不落盘。
	if _, err := a.getPM().ReadStyleFingerprint(); err == nil {
		t.Fatal("拒绝构建时不应写参考档文件")
	}
}

func TestNovelFingerprintBuild_AndStatusRoundtrip(t *testing.T) {
	a := newFingerprintTestApp(t)
	for i := 1; i <= 3; i++ {
		mustWriteChapter(t, a, i, "第三章样文。")
	}
	st, err := a.NovelFingerprintBuild()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if !st.Exists || st.Chapters != 3 || st.Chars < fingerprintMinChars || st.BuiltAt == "" {
		t.Fatalf("构建负载不完整: %+v", st)
	}
	if st.Summary == nil || st.Summary.SentenceMean <= 0 {
		t.Fatalf("摘要应为正: %+v", st.Summary)
	}
	// Status 读回与构建负载一致。
	st2, err := a.NovelFingerprintStatus()
	if err != nil {
		t.Fatalf("Status 失败: %v", err)
	}
	if !st2.Exists || st2.Chapters != st.Chapters || st2.Chars != st.Chars || st2.BuiltAt != st.BuiltAt {
		t.Fatalf("Status 回环不一致: %+v vs %+v", st2, st)
	}
	if st2.Summary == nil || st2.Summary.SentenceMean != st.Summary.SentenceMean {
		t.Fatalf("摘要回环不一致: %+v vs %+v", st2.Summary, st.Summary)
	}
}

func TestNovelFingerprintScore_WithAndWithoutRef(t *testing.T) {
	a := newFingerprintTestApp(t)
	for i := 1; i <= 3; i++ {
		mustWriteChapter(t, a, i, "第三章样文。")
	}
	if _, err := a.NovelFingerprintBuild(); err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 无参考档路径：单独验证（未构建工程）。
	a2 := newFingerprintTestApp(t)
	mustWriteChapter(t, a2, 1, "他的眼帘微微上扬，内心充满了震撼。他又缓缓抬头，眸光流转。")
	sc2, err := a2.NovelFingerprintScore(1)
	if err != nil {
		t.Fatalf("无参考档体检失败: %v", err)
	}
	if sc2.RefExists || sc2.Delta != nil {
		t.Fatalf("无参考档应 refExists=false/delta 缺省: %+v", sc2)
	}
	if sc2.Score <= 0 || len(sc2.Issues) == 0 {
		t.Fatalf("黑名单样本应有命中: %+v", sc2)
	}
	if sc2.Issues == nil {
		t.Fatal("issues 应为空数组而非 null（Go nil slice→JSON null 前端防不胜防）")
	}

	// 有参考档路径：delta 在位，同一作者文本 delta 应较小。
	sc, err := a.NovelFingerprintScore(1)
	if err != nil {
		t.Fatalf("有参考档体检失败: %v", err)
	}
	if !sc.RefExists || sc.Delta == nil {
		t.Fatalf("有参考档应 refExists=true/delta 在位: %+v", sc)
	}
	if *sc.Delta < 0 || *sc.Delta > 2 {
		t.Fatalf("delta 超出合理量程: %v", *sc.Delta)
	}
	for _, iss := range sc.Issues {
		if iss.Excerpt == "" {
			t.Fatalf("摘录应为空串兜底非缺省: %+v", iss)
		}
	}
}

func TestNovelFingerprintScore_EmptyChapterRejected(t *testing.T) {
	a := newFingerprintTestApp(t)
	if err := a.getPM().WriteChapter(1, "  \n\n  "); err != nil {
		t.Fatalf("写空章: %v", err)
	}
	if _, err := a.NovelFingerprintScore(1); err == nil || !strings.Contains(err.Error(), "为空") {
		t.Fatalf("空白章应拒收: %v", err)
	}
	if _, err := a.NovelFingerprintScore(0); err == nil || !strings.Contains(err.Error(), "章节号非法") {
		t.Fatalf("非法章节号应拒收: %v", err)
	}
}

// TestFingerprintPayloadWireShape 形状锁：负载 marshal 键为 camelCase
// （v4.276 形状普查先例——Wails 线上键形以 Go json 标签为唯一真源）。
func TestFingerprintPayloadWireShape(t *testing.T) {
	d := 0.31
	scoreJSON, err := json.Marshal(FingerprintScorePayload{ChapterNum: 1, Score: 42, Delta: &d})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var scoreKeys map[string]json.RawMessage
	if err := json.Unmarshal(scoreJSON, &scoreKeys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"chapterNum", "refExists", "score", "issues"} {
		if _, ok := scoreKeys[key]; !ok {
			t.Fatalf("缺键 %q: %s", key, scoreJSON)
		}
	}
	for _, bad := range []string{"ChapterNum", "RefExists", "Score", "Delta", "Issues"} {
		if _, ok := scoreKeys[bad]; ok {
			t.Fatalf("出现 PascalCase 键 %q: %s", bad, scoreJSON)
		}
	}
	// delta=0 也要在位（omitempty 对指针只判 nil）。
	zero := 0.0
	scoreJSON, _ = json.Marshal(FingerprintScorePayload{Delta: &zero})
	if !strings.Contains(string(scoreJSON), `"delta":0`) {
		t.Fatalf("delta=0 不应被吞: %s", scoreJSON)
	}

	statusJSON, err := json.Marshal(FingerprintStatusPayload{Exists: true, BuiltAt: "2026-09-13T00:00:00Z", Chapters: 3, Chars: 3200,
		Summary: &FingerprintSummary{TopBigrams: []string{"目光"}, TopTrigrams: []string{"目光流转"}, AuthorSignWords: []string{"攥紧"}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var statusKeys map[string]json.RawMessage
	if err := json.Unmarshal(statusJSON, &statusKeys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"exists", "builtAt", "chapters", "chars", "summary"} {
		if _, ok := statusKeys[key]; !ok {
			t.Fatalf("缺键 %q: %s", key, statusJSON)
		}
	}
	// 参考档文件键形锁定（types.StyleFingerprintFile 落盘形状）。
	fileJSON, err := json.Marshal(typesStyleFingerprintFixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"builtAt", "chapters", "chars", "fingerprint"} {
		if !strings.Contains(string(fileJSON), `"`+key+`"`) {
			t.Fatalf("参考档文件缺键 %q: %s", key, fileJSON)
		}
	}
}

func typesStyleFingerprintFixture() map[string]interface{} {
	return map[string]interface{}{
		"builtAt":     "2026-09-13T00:00:00Z",
		"chapters":    3,
		"chars":       3200,
		"fingerprint": json.RawMessage(`{}`),
	}
}

// TestCollectChapterSamples_StopsOnMissing 遍历口径：连续章节收集，缺章即停。
func TestCollectChapterSamples_StopsOnMissing(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteChapter(t, a, 1, "第一章。")
	mustWriteChapter(t, a, 2, "第二章。")
	mustWriteChapter(t, a, 4, "第四章缺第三章不应被读到。")
	samples, chapters, chars := collectChapterSamples(a.getPM().ReadChapterAsStitch)
	if chapters != 2 || len(samples) != 2 {
		t.Fatalf("缺章应停: chapters=%d len=%d", chapters, len(samples))
	}
	if chars <= 0 || strings.Contains(strings.Join(samples, ""), "第四章") {
		t.Fatalf("样本越界: chars=%d samples=%v", chars, samples)
	}
}

// ── oh-story T2 内核消费：书级白名单豁免（v4.286）──

// 写含 AI 词与否定翻转句的一章（去味/体检两用的最小命中样本）。
func mustWriteAiTasteChapter(t *testing.T, a *App, num int) {
	t.Helper()
	content := "她的眸光流转，带着几分笑意。\n\n这不是失败，而是他计划的第一步。夜风从窗缝里钻进来。"
	if err := a.getPM().WriteChapter(num, content); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	// v4 项目走场景路由：去味按场景读写，直接建一个场景承载同一正文。
	sm := a.getPM().SceneManager(num)
	sc, err := sm.Create("opening", "开场")
	if err != nil {
		t.Fatalf("建场景: %v", err)
	}
	sc.Content = content
	if err := sm.Write(sc); err != nil {
		t.Fatalf("写场景: %v", err)
	}
}

func TestDeSlopChapterAiTaste_BookWhitelistExempts(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteAiTasteChapter(t, a, 1)

	// 无白名单：眸光流转被替换（changes>0）
	res, err := a.DeSlopChapterAiTaste(1)
	if err != nil {
		t.Fatalf("去味: %v", err)
	}
	if res["changes"].(int) == 0 {
		t.Fatalf("无白名单应产生替换: %+v", res)
	}

	// 建白名单授权该片段：重建项目重跑，替换应被豁免
	a2 := newFingerprintTestApp(t)
	mustWriteAiTasteChapter(t, a2, 1)
	wlPath := filepath.Join(a2.getPM().Dir, ".deslop-whitelist")
	if err := os.WriteFile(wlPath, []byte("她的眸光流转，带着几分笑意。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res2, err := a2.DeSlopChapterAiTaste(1)
	if err != nil {
		t.Fatalf("白名单去味: %v", err)
	}
	if res2["changes"].(int) != 0 {
		t.Fatalf("授权片段应豁免替换: %+v", res2)
	}
	content, _ := a2.getPM().ReadChapter(1)
	if !strings.Contains(content, "眸光流转") {
		t.Fatalf("授权词应原样保留: %q", content)
	}
}

func TestNovelFingerprintScore_WhitelistExempts(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteAiTasteChapter(t, a, 1)

	// 无白名单：否定翻转命中并计分
	p1, err := a.NovelFingerprintScore(1)
	if err != nil {
		t.Fatalf("体检: %v", err)
	}
	hit := false
	for _, iss := range p1.Issues {
		if strings.Contains(iss.Reason, "否定铺垫后肯定翻转") {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("无白名单应命中否定翻转: %+v", p1.Issues)
	}

	// 白名单整句授权：命中摘除、分数下降（或有其它 issue 时权重减少）
	wlPath := filepath.Join(a.getPM().Dir, ".deslop-whitelist")
	if err := os.WriteFile(wlPath, []byte("这不是失败，而是他计划的第一步。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p2, err := a.NovelFingerprintScore(1)
	if err != nil {
		t.Fatalf("白名单体检: %v", err)
	}
	if p2.Score > p1.Score {
		t.Fatalf("豁免后分数不应升高: %d > %d", p2.Score, p1.Score)
	}
	for _, iss := range p2.Issues {
		if strings.Contains(iss.Reason, "否定铺垫后肯定翻转") {
			t.Fatalf("授权命中应摘除: %+v", iss)
		}
	}
}
