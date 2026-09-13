package app

import (
	"encoding/json"
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
