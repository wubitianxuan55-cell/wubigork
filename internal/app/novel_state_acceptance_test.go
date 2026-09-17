package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/narrative"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 刀 8 验收 · 线 C：叙事审批链 + 快照回退 + v3→v4 迁移（app 绑定级）──────
//
// 三个演练全部走 App 绑定方法（与前端同一路径），零生产代码改动。

// stateNewApp 建一个最小 App 并挂上真实临时项目（v3 初始态）。
func stateNewApp(t *testing.T) (*App, *project.Manager) {
	t.Helper()
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)
	return a, pm
}

// stateSceneAt 经绑定读取某章首个场景，返回 (场景ID, 正文)。
func stateSceneAt(t *testing.T, a *App, chapterNum int) (string, string) {
	t.Helper()
	scenes, err := a.GetChapterScenes(chapterNum)
	if err != nil {
		t.Fatalf("GetChapterScenes(%d) 不应报错: %v", chapterNum, err)
	}
	if len(scenes) == 0 {
		t.Fatalf("第 %d 章应有场景，得到 0 个", chapterNum)
	}
	id, _ := scenes[0]["id"].(string)
	content, _ := scenes[0]["content"].(string)
	return id, content
}

// TestKnife8_ApprovalChain_NoAutoSettleAndReplay 审批制叙事状态全链：
// AI 只能建议（StatePatch），驳回不入账本，批准才写 append-only 账本，
// 回放重建的快照与 GetNovelState 永远一致；账本只增；坏补丁明确报错。
func TestKnife8_ApprovalChain_NoAutoSettleAndReplay(t *testing.T) {
	a, pm := stateNewApp(t)
	// 第 1 章正文 + 含两个出场角色的摘要（CharactersAppeared 是 patch 的数据源）
	if err := pm.WriteChapter(1, "第一章正文：叶辰与苏青在青云山门相遇。"); err != nil {
		t.Fatalf("写章节失败: %v", err)
	}
	if err := pm.WriteChapterSummary(1, &types.ChapterSummary{
		Title:              "第一章 初遇",
		Summary:            "叶辰入宗，与苏青结识。",
		CharactersAppeared: []string{"叶辰", "苏青"},
	}); err != nil {
		t.Fatalf("写章节摘要失败: %v", err)
	}

	// 1) 构造建议补丁：确定性 ch001-ai，两角色 alive，ProposedBy=ai
	built, err := a.BuildNovelStatePatch(1)
	if err != nil {
		t.Fatalf("BuildNovelStatePatch 不应报错: %v", err)
	}
	patch, ok := built["patch"].(narrative.StatePatch)
	if !ok {
		t.Fatalf("返回值 \"patch\" 应为 narrative.StatePatch，得到 %T", built["patch"])
	}
	if patch.ID != "ch001-ai" {
		t.Errorf("patch.ID 应为 ch001-ai，得到 %q", patch.ID)
	}
	if patch.ProposedBy != "ai" {
		t.Errorf("patch.ProposedBy 应为 ai，得到 %q", patch.ProposedBy)
	}
	if amount, _ := built["amount"].(int); amount != 2 {
		t.Errorf("amount 应为 2，得到 %v", built["amount"])
	}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("序列化 patch 失败: %v", err)
	}

	// 2) 作者驳回（approved=false）：无一自动入库
	rej, err := a.SettleNovelState(string(patchJSON), false)
	if err != nil {
		t.Fatalf("驳回结算不应报错: %v", err)
	}
	if v, _ := rej["approved"].(bool); v {
		t.Errorf("驳回结果 approved 应为 false")
	}
	if v, _ := rej["version"].(int); v != 0 {
		t.Errorf("驳回后 version 应为 0，得到 %v", rej["version"])
	}
	if ents, _ := rej["entities"].(map[string]narrative.EntityState); len(ents) != 0 {
		t.Errorf("驳回后 entities 应为空，得到 %d 条", len(ents))
	}
	st, err := a.GetNovelState()
	if err != nil {
		t.Fatalf("GetNovelState 不应报错: %v", err)
	}
	if v, _ := st["version"].(int); v != 0 {
		t.Errorf("驳回后 GetNovelState version 应为 0，得到 %v", st["version"])
	}
	if ents, _ := st["entities"].(map[string]narrative.EntityState); len(ents) != 0 {
		t.Errorf("驳回后 GetNovelState entities 应为空，得到 %d 条", len(ents))
	}
	j, err := narrative.Open(pm.Dir)
	if err != nil {
		t.Fatalf("narrative.Open 失败: %v", err)
	}
	replay, err := j.Replay()
	if err != nil {
		t.Fatalf("Replay 失败: %v", err)
	}
	if len(replay) != 0 {
		t.Errorf("驳回后账本不应有 settled 记录，得到 %d 条", len(replay))
	}

	// 3) 作者批准（approved=true）：入库，两角色 alive
	app1, err := a.SettleNovelState(string(patchJSON), true)
	if err != nil {
		t.Fatalf("批准结算不应报错: %v", err)
	}
	if v, _ := app1["version"].(int); v != 1 {
		t.Errorf("批准后 version 应为 1，得到 %v", app1["version"])
	}
	ents1, _ := app1["entities"].(map[string]narrative.EntityState)
	if len(ents1) != 2 {
		t.Fatalf("批准后 entities 应有 2 条，得到 %d", len(ents1))
	}
	for _, name := range []string{"叶辰", "苏青"} {
		e, ok := ents1["char:"+name]
		if !ok {
			t.Errorf("entities 应含 char:%s", name)
			continue
		}
		if e.Status != "alive" {
			t.Errorf("char:%s 状态应为 alive，得到 %q", name, e.Status)
		}
	}

	// 4) append-only 回放：独立 Open 后 Replay/Snapshot 与绑定读数一致
	j2, err := narrative.Open(pm.Dir)
	if err != nil {
		t.Fatalf("narrative.Open 失败: %v", err)
	}
	rep2, err := j2.Replay()
	if err != nil {
		t.Fatalf("Replay 失败: %v", err)
	}
	if len(rep2) != 1 || rep2[0].ID != "ch001-ai" {
		t.Errorf("账本应恰有 1 条 ch001-ai，得到 %d 条", len(rep2))
	}
	snap2, err := j2.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot 失败: %v", err)
	}
	if snap2.Version != 1 {
		t.Errorf("回放快照 version 应为 1，得到 %d", snap2.Version)
	}
	st2, err := a.GetNovelState()
	if err != nil {
		t.Fatalf("GetNovelState 不应报错: %v", err)
	}
	if v, _ := st2["version"].(int); v != snap2.Version {
		t.Errorf("绑定 version(%v) 应与回放快照(%d) 一致", st2["version"], snap2.Version)
	}
	stEnts, _ := st2["entities"].(map[string]narrative.EntityState)
	if len(stEnts) != len(snap2.Entities) {
		t.Errorf("绑定 entities 数(%d) 应与回放快照(%d) 一致", len(stEnts), len(snap2.Entities))
	}
	for id, want := range snap2.Entities {
		got, ok := stEnts[id]
		if !ok {
			t.Errorf("绑定 entities 缺 %q", id)
			continue
		}
		if got.Status != want.Status || got.Name != want.Name || got.Type != want.Type {
			t.Errorf("实体 %q 绑定读数与回放不一致: %+v vs %+v", id, got, want)
		}
	}

	// 5) 再批准一次：账本只增（version 1→2）
	app2, err := a.SettleNovelState(string(patchJSON), true)
	if err != nil {
		t.Fatalf("二次批准结算不应报错: %v", err)
	}
	if v, _ := app2["version"].(int); v != 2 {
		t.Errorf("二次批准后 version 应为 2，得到 %v", app2["version"])
	}

	// 6) 坏 JSON 明确报错，且账本不受污染
	if _, err := a.SettleNovelState("{这不是合法JSON", true); err == nil {
		t.Errorf("坏 JSON 应明确报错，得到 nil")
	} else if !strings.Contains(err.Error(), "解析") {
		t.Errorf("坏 JSON 错误应提示解析失败，得到 %q", err.Error())
	}
	st3, err := a.GetNovelState()
	if err != nil {
		t.Fatalf("GetNovelState 不应报错: %v", err)
	}
	if v, _ := st3["version"].(int); v != 2 {
		t.Errorf("坏 JSON 后 version 应保持 2，得到 %v", st3["version"])
	}
}

// TestKnife8_SnapshotRestore_Drill 快照回退演练（app 绑定级往返）：
// v4 场景改写 → 建快照 → 再改写 → RestoreSnapshot 应回到快照时刻的内容。
func TestKnife8_SnapshotRestore_Drill(t *testing.T) {
	initial := "夜雨初落，山门无声。\n叶辰握紧断剑，听远处的钟。"
	a, _ := newMaterializeApp(t, initial) // v4 工程 + 第 3 章 blob
	sceneID, content := stateSceneAt(t, a, 3)
	if content != initial {
		t.Fatalf("物化场景正文应等于初始 blob，得到 %q", content)
	}

	// 第一次改写 → 此刻建快照
	changed := "夜雨初落，山门无声。\n叶辰握紧断剑，钟声更近了。\n雨里有血腥气。"
	if err := a.SaveScene(3, sceneID, changed); err != nil {
		t.Fatalf("SaveScene 不应报错: %v", err)
	}
	snap, err := a.CreateSnapshot(sceneID, 3, "刀8回退演练")
	if err != nil {
		t.Fatalf("CreateSnapshot 不应报错: %v", err)
	}
	snapID, _ := snap["id"].(string)
	if snapID == "" {
		t.Fatalf("快照应返回非空 id，得到 %v", snap["id"])
	}

	// 第二次改写（偏离快照态）
	further := "结局已改写，山门崩塌。\n新的线索就此埋下。"
	if err := a.SaveScene(3, sceneID, further); err != nil {
		t.Fatalf("SaveScene 不应报错: %v", err)
	}
	if _, cur := stateSceneAt(t, a, 3); cur != further {
		t.Fatalf("改写后正文应为偏离态，得到 %q", cur)
	}

	// 快照可列出；回退后正文必须精确回到快照时刻
	snaps, err := a.ListSnapshots(sceneID, 3)
	if err != nil {
		t.Fatalf("ListSnapshots 不应报错: %v", err)
	}
	if len(snaps) < 1 {
		t.Fatalf("应至少列出 1 个快照，得到 %d 个", len(snaps))
	}
	if err := a.RestoreSnapshot(snapID, sceneID, 3); err != nil {
		t.Fatalf("RestoreSnapshot 不应报错: %v", err)
	}
	if _, after := stateSceneAt(t, a, 3); after != changed {
		t.Errorf("回退后正文应等于快照态 %q，得到 %q", changed, after)
	}
}

// TestKnife8_MigrateV3ToV4_NonDestructive 真实 v3→v4 迁移演练：
// 有 blob 章的 v3 工程迁移后场景 ≥1、拼接读数与原文无损一致，
// 且 _v3_backup/ 保留原文 blob 副本（非破坏）。
func TestKnife8_MigrateV3ToV4_NonDestructive(t *testing.T) {
	a, pm := stateNewApp(t)
	// project.Create 默认写 v4 标记 → 摘掉标记模拟真实 v3 工程
	if err := os.Remove(filepath.Join(pm.Dir, ".gaea", "v4")); err != nil {
		t.Fatalf("摘除 v4 标记失败: %v", err)
	}
	if pm.IsV4() {
		t.Fatalf("前置条件：应为 v3 工程（无 v4 标记）")
	}

	// v3 blob 章内容 X（多行，验证迁移逐字节无损）
	X := "第一章旧稿：叶辰背剑下山。\n风雪满肩，山门在身后合拢。\n他在雪里走出第一条路。"
	if err := pm.WriteChapter(1, X); err != nil {
		t.Fatalf("写 v3 blob 章失败: %v", err)
	}

	// chapter_handler 的迁移入口（底层 pm.MigrateV3ToV4）
	if err := a.MigrateProjectToV4(); err != nil {
		t.Fatalf("MigrateProjectToV4 不应报错: %v", err)
	}
	if !pm.IsV4() || !a.IsProjectV4() {
		t.Fatalf("迁移后应为 v4 结构")
	}

	// 场景 ≥1，且绑定读到的场景正文即原文
	scenes, err := a.GetChapterScenes(1)
	if err != nil {
		t.Fatalf("GetChapterScenes 不应报错: %v", err)
	}
	if len(scenes) < 1 {
		t.Fatalf("迁移后应至少有 1 个场景，得到 %d 个", len(scenes))
	}
	if got, _ := scenes[0]["content"].(string); got != X {
		t.Errorf("迁移场景正文应无损等于原文，得到 %q", got)
	}

	// 拼接视图读取 == X（单场景 Stitch 无分隔线注入）
	stitch, err := pm.ReadChapterAsStitch(1)
	if err != nil {
		t.Fatalf("ReadChapterAsStitch 不应报错: %v", err)
	}
	if stitch != X {
		t.Errorf("ReadChapterAsStitch(1) 应无损等于原文 X，得到 %q", stitch)
	}

	// 非破坏：_v3_backup/ 内含原文 blob 副本
	backup, err := os.ReadFile(filepath.Join(pm.Dir, "_v3_backup", "001.md"))
	if err != nil {
		t.Fatalf("备份文件应存在且可读: %v", err)
	}
	if string(backup) != X {
		t.Errorf("备份应等于原文 blob，得到 %q", string(backup))
	}
}
