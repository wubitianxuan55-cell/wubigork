package app

// AP1-08 现状钉子测试：NovelApplyRewriteVersion / NovelRestoreRewriteVersion
// 四条路径（Apply 场景拼接/整章 × Restore 场景拼接/整章）与错误分支的
// **重构前行为快照**。版本全部用 SaveRewriteVersion 直接钉造（不依赖 LLM 桩），
// slog 文案与级别经默认 logger 捕获逐字断言。
//
// 这些测试在写回重构（AP1-08 收敛）前后必须一字不改仍绿——前后同绿 =
// 行为逐字段不变的机器证明。刻意钉死的语义差异（并错必红）：
//   - Apply 场景路带完整性校验 + SceneNew 校验 + 应用前快照（Capture 先于写回，
//     快照的是旧文）；Restore 场景路**无**完整性校验、**无**快照（无条件还原）；
//   - Apply/Restore 整章路写回后 rebuildScenesFromBlob（v4 重置单场景）；
//   - Apply 幂等分支在任何写盘之前返回（applied 已存在时不重写正文）。

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// captureSlogAll 抓取默认 slog 输出（钉 slog 文案与级别用；同 captureSlogWarn
// 手法，包内测试无 t.Parallel，串行安全，t.Cleanup 还原全局 logger）。
func captureSlogAll(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return func() string { return buf.String() }
}

// mustPinV4Chapter 造 v4 场景章（复用既有夹具：v4 标记 + blob + 两场景），
// 返回按序两个场景 ID，并钉住初始 blob 投影。
func mustPinV4Chapter(t *testing.T, a *App, chapterNum int, s1, s2 string) (id1, id2 string) {
	t.Helper()
	mustMakeV4SceneChapter(t, a, chapterNum, s1, s2)
	metas, err := a.getPM().SceneManager(chapterNum).List()
	if err != nil {
		t.Fatalf("列场景: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("夹具应有两个场景，得到 %d", len(metas))
	}
	if blob, _ := a.getPM().ReadChapter(chapterNum); blob != s1+"\n\n"+s2 {
		t.Fatalf("夹具初始 blob 应为拼接: %q", blob)
	}
	return metas[0].ID, metas[1].ID
}

// mustPinVersion 直接钉造并落盘一个重写版本（绕过 LLM 链路）。
func mustPinVersion(t *testing.T, a *App, v *types.RewriteVersion) {
	t.Helper()
	if err := a.getPM().SaveRewriteVersion(v); err != nil {
		t.Fatalf("落版本: %v", err)
	}
}

// mustPinSceneVersion 钉造一个场景拼接语义的版本（Status 由调用方给定）。
func mustPinSceneVersion(t *testing.T, a *App, chapterNum int, status types.RewriteStatus, sceneID, sceneOriginal, sceneNew string) *types.RewriteVersion {
	t.Helper()
	v := &types.RewriteVersion{
		ChapterNum:      chapterNum,
		Mode:            types.RewriteModePartial,
		Status:          status,
		Source:          types.RewriteSourceCustom,
		SceneID:         sceneID,
		SceneOriginal:   sceneOriginal,
		SceneNew:        sceneNew,
		OriginalContent: sceneOriginal + "\n\n场景二原文。", // 全文口径（契约）
		NewContent:      sceneNew + "\n\n场景二原文。",
	}
	mustPinVersion(t, a, v)
	return v
}

// ── 路径 1：Apply 场景拼接路（happy：校验过 → 快照 → 写回 → blob 同步 → 落账）──

func TestApplyRewritePin_ScenePathHappy(t *testing.T) {
	a := newFingerprintTestApp(t)
	const s1 = "场景一原文。"
	const s2 = "场景二原文。"
	const sceneNew = "场景一重写完成后的全新正文，比原文长了不少。"
	id1, id2 := mustPinV4Chapter(t, a, 2, s1, s2)
	logs := captureSlogAll(t)
	v := mustPinSceneVersion(t, a, 2, types.RewriteCompleted, id1, s1, sceneNew)

	res, err := a.NovelApplyRewriteVersion(2, v.ID)
	if err != nil {
		t.Fatalf("应用: %v", err)
	}
	if res["applied"] != true || len(res) != 1 {
		t.Fatalf("返回应恰为 {applied:true}: %v", res)
	}
	// slog：场景路专属文案 + INFO 级别
	if want := "level=INFO msg=局部重写版本已应用（场景拼接）"; !strings.Contains(logs(), want) {
		t.Fatalf("缺 slog 行 %q，日志:\n%s", want, logs())
	}
	if !strings.Contains(logs(), "scene="+id1) {
		t.Fatalf("slog 应带 scene=%s，日志:\n%s", id1, logs())
	}

	sm := a.getPM().SceneManager(2)
	// 结构保持：两场景不变，他场字节不动
	metas, _ := sm.List()
	if len(metas) != 2 {
		t.Fatalf("应用后场景结构必须保持，得到 %d", len(metas))
	}
	sc2, _ := sm.Read(id2)
	if sc2.Content != s2 {
		t.Fatalf("他场必须字节不动: %q", sc2.Content)
	}
	// 归属场景 = SceneNew
	sc1, _ := sm.Read(id1)
	if sc1.Content != sceneNew {
		t.Fatalf("归属场景应写回 SceneNew: %q", sc1.Content)
	}
	// blob 投影同步
	if blob, _ := a.getPM().ReadChapter(2); blob != sceneNew+"\n\n"+s2 {
		t.Fatalf("blob 应同步: %q", blob)
	}
	// 版本落账：applied + AppliedAt
	got, err := a.NovelGetRewriteVersion(2, v.ID)
	if err != nil || got.Status != types.RewriteApplied || got.AppliedAt == nil {
		t.Fatalf("版本应落账 applied+AppliedAt: %v %+v", err, got)
	}
	// 应用前快照：恰好 +1，标签/触发器/字数钉死（Capture 的必须是写回**前**的旧文）
	snaps, err := a.getPM().SnapshotStore(2).List(id1)
	if err != nil {
		t.Fatalf("列快照: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("应用应恰好落 1 个快照，得到 %d", len(snaps))
	}
	if snaps[0].Label != "局部重写应用前" || snaps[0].Trigger != "partial-apply" {
		t.Fatalf("快照标签/触发器不对: %+v", snaps[0])
	}
	if snaps[0].WordCount != len([]rune(s1)) {
		t.Fatalf("快照字数应为旧文字数 %d（证明快照先于写回），得到 %d", len([]rune(s1)), snaps[0].WordCount)
	}
}

// ── Apply 场景路错误分支（文案逐字 + 校验顺序 + 拒绝时零副作用）──────────

func TestApplyRewritePin_ScenePathRejects(t *testing.T) {
	const s1 = "场景一原文。"
	const s2 = "场景二原文。"

	t.Run("完整性校验拒绝_文案逐字_零副作用", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		id1, _ := mustPinV4Chapter(t, a, 2, s1, s2)
		logs := captureSlogAll(t)
		v := mustPinSceneVersion(t, a, 2, types.RewriteCompleted, id1, s1, "新场景。")
		// 提案后旁路手改归属场景
		sm := a.getPM().SceneManager(2)
		sc, _ := sm.Read(id1)
		sc.Content = "场景一被手改了。"
		if err := sm.Write(sc); err != nil {
			t.Fatal(err)
		}

		_, err := a.NovelApplyRewriteVersion(2, v.ID)
		if err == nil || err.Error() != "归属场景「场景 1」在提案后被修改过，本版本不可应用（请重新选段重写）" {
			t.Fatalf("完整性拒绝文案应逐字一致: %v", err)
		}
		// 零副作用：场景保持手改值、版本仍 completed、无快照、无 applied 日志
		sc2r, _ := sm.Read(id1)
		if sc2r.Content != "场景一被手改了。" {
			t.Fatalf("拒绝后场景不得被写: %q", sc2r.Content)
		}
		got, _ := a.NovelGetRewriteVersion(2, v.ID)
		if got.Status != types.RewriteCompleted {
			t.Fatalf("拒绝后版本应仍 completed: %+v", got)
		}
		if snaps, _ := a.getPM().SnapshotStore(2).List(id1); len(snaps) != 0 {
			t.Fatalf("拒绝时不应落快照，得到 %d", len(snaps))
		}
		if strings.Contains(logs(), "已应用") {
			t.Fatalf("拒绝时不应有已应用日志:\n%s", logs())
		}
	})

	t.Run("校验顺序_完整性先于SceneNew校验", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		id1, _ := mustPinV4Chapter(t, a, 2, s1, s2)
		v := mustPinSceneVersion(t, a, 2, types.RewriteCompleted, id1, s1, "   ") // SceneNew 空白
		sm := a.getPM().SceneManager(2)
		sc, _ := sm.Read(id1)
		sc.Content = "场景一被手改了。" // 完整性也坏 → 完整性错误必须先报
		if err := sm.Write(sc); err != nil {
			t.Fatal(err)
		}
		_, err := a.NovelApplyRewriteVersion(2, v.ID)
		if err == nil || !strings.Contains(err.Error(), "修改过") {
			t.Fatalf("完整性校验应先于 SceneNew 校验: %v", err)
		}
	})

	t.Run("SceneNew空白_文案逐字", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		id1, _ := mustPinV4Chapter(t, a, 2, s1, s2)
		v := mustPinSceneVersion(t, a, 2, types.RewriteCompleted, id1, s1, " \n\t ") // 纯空白钉 TrimSpace
		_, err := a.NovelApplyRewriteVersion(2, v.ID)
		if err == nil || err.Error() != "版本缺场景新内容，不可应用" {
			t.Fatalf("SceneNew 空白文案应逐字一致: %v", err)
		}
	})

	t.Run("归属场景已删_读取失败前缀", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		id1, _ := mustPinV4Chapter(t, a, 2, s1, s2)
		v := mustPinSceneVersion(t, a, 2, types.RewriteCompleted, id1, s1, "新场景。")
		if err := a.getPM().SceneManager(2).Delete(id1); err != nil {
			t.Fatal(err)
		}
		_, err := a.NovelApplyRewriteVersion(2, v.ID)
		if err == nil || !strings.HasPrefix(err.Error(), "读取归属场景失败: ") {
			t.Fatalf("场景已删应报读取归属场景失败: %v", err)
		}
	})
}

// ── 路径 2：Apply 整章路（v3 直写正文 + v4 rebuild 单场景）───────────────

func TestApplyRewritePin_WholePath(t *testing.T) {
	t.Run("v3整章_写回+落账+slog", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		mustWriteChapter(t, a, 2, "旧正文种子")
		if err := a.getPM().WriteChapter(2, "这是原始的旧正文。"); err != nil {
			t.Fatal(err)
		}
		logs := captureSlogAll(t)
		v := &types.RewriteVersion{
			ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteCompleted,
			Source:          types.RewriteSourceCustom,
			OriginalContent: "这是原始的旧正文。", NewContent: "这是重写后的全新正文。",
		}
		mustPinVersion(t, a, v)

		res, err := a.NovelApplyRewriteVersion(2, v.ID)
		if err != nil {
			t.Fatalf("应用: %v", err)
		}
		if res["applied"] != true || len(res) != 1 {
			t.Fatalf("返回应恰为 {applied:true}: %v", res)
		}
		if c, _ := a.getPM().ReadChapter(2); c != "这是重写后的全新正文。" {
			t.Fatalf("正文应写回 NewContent: %q", c)
		}
		if want := "level=INFO msg=重写版本已应用 "; !strings.Contains(logs(), want) {
			t.Fatalf("缺 slog 行 %q，日志:\n%s", want, logs())
		}
		got, _ := a.NovelGetRewriteVersion(2, v.ID)
		if got.Status != types.RewriteApplied || got.AppliedAt == nil {
			t.Fatalf("版本应落账 applied+AppliedAt: %+v", got)
		}
	})

	t.Run("v4整章rebuild_应用与恢复对称重置单场景", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		const s1, s2 = "场景一原文。", "场景二原文。"
		mustPinV4Chapter(t, a, 2, s1, s2)
		v := &types.RewriteVersion{
			ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteCompleted,
			Source:          types.RewriteSourceCustom,
			OriginalContent: s1 + "\n\n" + s2, NewContent: "重置后的整章新文。",
		}
		mustPinVersion(t, a, v)

		if _, err := a.NovelApplyRewriteVersion(2, v.ID); err != nil {
			t.Fatalf("应用: %v", err)
		}
		sm := a.getPM().SceneManager(2)
		metas, _ := sm.List()
		if len(metas) != 1 {
			t.Fatalf("应用后应重置为单场景，得到 %d", len(metas))
		}
		if blob, _ := a.getPM().ReadChapter(2); blob != "重置后的整章新文。" {
			t.Fatalf("应用后 blob: %q", blob)
		}

		if _, err := a.NovelRestoreRewriteVersion(2, v.ID); err != nil {
			t.Fatalf("恢复: %v", err)
		}
		if blob, _ := a.getPM().ReadChapter(2); blob != s1+"\n\n"+s2 {
			t.Fatalf("恢复后 blob 应为原文: %q", blob)
		}
		metas2, _ := a.getPM().SceneManager(2).List()
		if len(metas2) != 1 {
			t.Fatalf("恢复后也应重置为单场景，得到 %d", len(metas2))
		}
	})
}

// ── Apply 幂等分支：applied 再应用在任何写盘之前返回 ─────────────────────

func TestApplyRewritePin_Idempotent(t *testing.T) {
	a := newFingerprintTestApp(t)
	if err := a.getPM().WriteChapter(2, "旧正文。"); err != nil {
		t.Fatal(err)
	}
	v := &types.RewriteVersion{
		ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteCompleted,
		Source:          types.RewriteSourceCustom,
		OriginalContent: "旧正文。", NewContent: "新正文。",
	}
	mustPinVersion(t, a, v)

	if _, err := a.NovelApplyRewriteVersion(2, v.ID); err != nil {
		t.Fatalf("首次应用: %v", err)
	}
	first, _ := a.NovelGetRewriteVersion(2, v.ID)
	if first.Status != types.RewriteApplied || first.AppliedAt == nil {
		t.Fatalf("首次应用后应 applied: %+v", first)
	}
	// 篡改磁盘正文：若幂等分支误重写，这里立刻红
	if err := a.getPM().WriteChapter(2, "别的写者后来改的。"); err != nil {
		t.Fatal(err)
	}

	res, err := a.NovelApplyRewriteVersion(2, v.ID)
	if err != nil {
		t.Fatalf("幂等应用: %v", err)
	}
	if res["applied"] != true || res["idempotent"] != true || len(res) != 2 {
		t.Fatalf("幂等返回应恰为 {applied:true,idempotent:true}: %v", res)
	}
	if c, _ := a.getPM().ReadChapter(2); c != "别的写者后来改的。" {
		t.Fatalf("幂等应用不得重写正文: %q", c)
	}
	second, _ := a.NovelGetRewriteVersion(2, v.ID)
	if !second.AppliedAt.Equal(*first.AppliedAt) {
		t.Fatalf("幂等应用不得更新 AppliedAt: %v -> %v", first.AppliedAt, second.AppliedAt)
	}
}

// ── Apply 通用守卫（状态机 + 空新内容，文案逐字）────────────────────────

func TestApplyRewritePin_Guards(t *testing.T) {
	a := newFingerprintTestApp(t)
	if err := a.getPM().WriteChapter(2, "旧正文。"); err != nil {
		t.Fatal(err)
	}

	failed := &types.RewriteVersion{
		ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteFailed,
		Source: types.RewriteSourceCustom, NewContent: "x",
	}
	mustPinVersion(t, a, failed)
	_, err := a.NovelApplyRewriteVersion(2, failed.ID)
	if err == nil || err.Error() != `版本状态 "failed" 不可应用` {
		t.Fatalf("failed 状态文案应逐字一致: %v", err)
	}

	empty := &types.RewriteVersion{
		ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteCompleted,
		Source: types.RewriteSourceCustom, NewContent: "  ",
	}
	mustPinVersion(t, a, empty)
	_, err = a.NovelApplyRewriteVersion(2, empty.ID)
	if err == nil || err.Error() != "版本无新内容，不可应用" {
		t.Fatalf("空新内容文案应逐字一致: %v", err)
	}
}

// ── 路径 3：Restore 场景拼接路（关键语义差异：无完整性校验、无快照）────────

func TestRestoreRewritePin_ScenePathNoIntegrityNoSnapshot(t *testing.T) {
	a := newFingerprintTestApp(t)
	const s1 = "场景一原文。"
	const s2 = "场景二原文。"
	const sceneNew = "场景一重写后的新正文。"
	id1, id2 := mustPinV4Chapter(t, a, 2, s1, s2)
	v := mustPinSceneVersion(t, a, 2, types.RewriteApplied, id1, s1, sceneNew)

	// 应用后的世界状态：归属场景=SceneNew、blob 同步。
	sm := a.getPM().SceneManager(2)
	sc, _ := sm.Read(id1)
	sc.Content = sceneNew
	if err := sm.Write(sc); err != nil {
		t.Fatal(err)
	}
	// 恢复前归属场景再被手改（若 Restore 带完整性校验，这里必红——该差异是刻意保留）。
	sc, _ = sm.Read(id1)
	sc.Content = "应用后又被手改过。"
	if err := sm.Write(sc); err != nil {
		t.Fatal(err)
	}
	logs := captureSlogAll(t)

	res, err := a.NovelRestoreRewriteVersion(2, v.ID)
	if err != nil {
		t.Fatalf("恢复（场景被改过也不得拦）: %v", err)
	}
	if res["restored"] != true || len(res) != 1 {
		t.Fatalf("返回应恰为 {restored:true}: %v", res)
	}
	if want := "level=INFO msg=局部重写版本已恢复原文（场景拼接）"; !strings.Contains(logs(), want) {
		t.Fatalf("缺 slog 行 %q，日志:\n%s", want, logs())
	}
	sc, _ = sm.Read(id1)
	if sc.Content != s1 {
		t.Fatalf("恢复应无条件写回 SceneOriginal: %q", sc.Content)
	}
	sc2, _ := sm.Read(id2)
	if sc2.Content != s2 {
		t.Fatalf("他场必须字节不动: %q", sc2.Content)
	}
	metas, _ := sm.List()
	if len(metas) != 2 {
		t.Fatalf("恢复后结构必须保持，得到 %d", len(metas))
	}
	if blob, _ := a.getPM().ReadChapter(2); blob != s1+"\n\n"+s2 {
		t.Fatalf("恢复后 blob 应同步: %q", blob)
	}
	got, _ := a.NovelGetRewriteVersion(2, v.ID)
	if got.RestoredAt == nil || got.RestoredFrom != v.ID {
		t.Fatalf("恢复审计缺失: %+v", got)
	}
	// 恢复不落快照（与 Apply 的差异之二）
	if snaps, _ := a.getPM().SnapshotStore(2).List(id1); len(snaps) != 0 {
		t.Fatalf("恢复不应落快照，得到 %d", len(snaps))
	}
}

// ── 路径 4：Restore 整章路（v3）+ 通用守卫 ──────────────────────────────

func TestRestoreRewritePin_WholePathAndGuards(t *testing.T) {
	t.Run("v3整章_恢复+审计+slog", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		if err := a.getPM().WriteChapter(2, "这是原始的旧正文。"); err != nil {
			t.Fatal(err)
		}
		v := &types.RewriteVersion{
			ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteApplied,
			Source:          types.RewriteSourceCustom,
			OriginalContent: "这是原始的旧正文。", NewContent: "这是重写后的全新正文。",
		}
		mustPinVersion(t, a, v)
		if err := a.getPM().WriteChapter(2, "这是重写后的全新正文。"); err != nil {
			t.Fatal(err)
		}
		logs := captureSlogAll(t)

		res, err := a.NovelRestoreRewriteVersion(2, v.ID)
		if err != nil {
			t.Fatalf("恢复: %v", err)
		}
		if res["restored"] != true || len(res) != 1 {
			t.Fatalf("返回应恰为 {restored:true}: %v", res)
		}
		if c, _ := a.getPM().ReadChapter(2); c != "这是原始的旧正文。" {
			t.Fatalf("正文应恢复为原文: %q", c)
		}
		if want := "level=INFO msg=重写版本已恢复原文 "; !strings.Contains(logs(), want) {
			t.Fatalf("缺 slog 行 %q，日志:\n%s", want, logs())
		}
		got, _ := a.NovelGetRewriteVersion(2, v.ID)
		if got.RestoredAt == nil || got.RestoredFrom != v.ID {
			t.Fatalf("恢复审计缺失: %+v", got)
		}
	})

	t.Run("状态非applied_文案逐字", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		v := &types.RewriteVersion{
			ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteCompleted,
			Source: types.RewriteSourceCustom, OriginalContent: "旧", NewContent: "新",
		}
		mustPinVersion(t, a, v)
		_, err := a.NovelRestoreRewriteVersion(2, v.ID)
		if err == nil || err.Error() != `仅已应用的版本可恢复原文（当前 "completed"）` {
			t.Fatalf("非 applied 状态文案应逐字一致: %v", err)
		}
	})

	t.Run("缺全文原文快照_文案逐字", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		v := &types.RewriteVersion{
			ChapterNum: 2, Mode: types.RewriteModeWhole, Status: types.RewriteApplied,
			Source: types.RewriteSourceCustom, OriginalContent: "", NewContent: "新",
		}
		mustPinVersion(t, a, v)
		_, err := a.NovelRestoreRewriteVersion(2, v.ID)
		if err == nil || err.Error() != "版本缺原文快照，无法恢复" {
			t.Fatalf("缺原文快照文案应逐字一致: %v", err)
		}
	})

	t.Run("场景路缺SceneOriginal_先全文闸后场景闸_文案逐字", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		id1, _ := mustPinV4Chapter(t, a, 2, "场景一原文。", "场景二原文。")
		// OriginalContent 非空（过全文闸）但 SceneOriginal 空（场景闸）——
		// 顺序钉死：全文闸在前，场景闸在后。
		v := &types.RewriteVersion{
			ChapterNum: 2, Mode: types.RewriteModePartial, Status: types.RewriteApplied,
			Source:  types.RewriteSourceCustom,
			SceneID: id1, SceneOriginal: "", SceneNew: "新场景。",
			OriginalContent: "全文原文非空。", NewContent: "全文新。",
		}
		mustPinVersion(t, a, v)
		_, err := a.NovelRestoreRewriteVersion(2, v.ID)
		if err == nil || err.Error() != "版本缺场景原文快照，无法恢复" {
			t.Fatalf("缺场景原文快照文案应逐字一致: %v", err)
		}
	})

	t.Run("场景路归属场景已删_读取失败前缀", func(t *testing.T) {
		a := newFingerprintTestApp(t)
		id1, _ := mustPinV4Chapter(t, a, 2, "场景一原文。", "场景二原文。")
		v := mustPinSceneVersion(t, a, 2, types.RewriteApplied, id1, "场景一原文。", "新场景。")
		if err := a.getPM().SceneManager(2).Delete(id1); err != nil {
			t.Fatal(err)
		}
		_, err := a.NovelRestoreRewriteVersion(2, v.ID)
		if err == nil || !strings.HasPrefix(err.Error(), "读取归属场景失败: ") {
			t.Fatalf("场景已删应报读取归属场景失败: %v", err)
		}
	})
}
