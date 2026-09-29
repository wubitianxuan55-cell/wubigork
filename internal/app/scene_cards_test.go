package app

// 长篇刀2：场景卡与场景级生成测试（规格 进度计划/gaea-novel-scene-cards-20260929.md §2）。
// 覆盖：卡区段渲染三态 / 逐场景生成流（卡闸点名+跳过/逐场落盘+done 事件/
// Stitch→blob/SceneRefs 回写）/ 卡字段进 prompt / 单场景重写他场字节不动+版本留痕。

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// mkCardScene 在章下建一个带卡字段的场景（直写 scene.Manager，绕过 SaveSceneMeta
// 的 IsV4 门——测试聚焦生成链不测 v3/v4 边界）。
func mkCardScene(t *testing.T, pm *project.Manager, chapterNum int, title string, card bool) string {
	t.Helper()
	sm := pm.SceneManager(chapterNum)
	sc, err := sm.Create("s"+title, title)
	if err != nil {
		t.Fatalf("建场景 %s: %v", title, err)
	}
	if card {
		sc.Meta.Goal = title + "的目标"
		sc.Meta.Conflict = title + "的阻挡"
		sc.Meta.Turn = "价值从负到正"
		sc.Meta.Outcome = title + "后状态改变"
		sc.Meta.ExitHook = title + "的钩子"
	}
	if err := sm.Write(sc); err != nil {
		t.Fatalf("写场景 %s: %v", title, err)
	}
	return sc.Meta.ID
}

// TestSceneCardSection 卡区段纯函数三态：无卡空串（prompt 零变化底线）/ 全卡渲染 /
// 前场衔接独立成段。
func TestSceneCardSection(t *testing.T) {
	if got := sceneCardSection(&types.SceneMeta{Title: "无卡"}, nil); got != "" {
		t.Fatalf("无卡场景应返回空串（prompt 零回归底线），得到 %q", got)
	}
	got := sceneCardSection(&types.SceneMeta{
		Goal: "拿到账本", Conflict: "管家守着", Turn: "从被动到主动", Outcome: "账本到手", ExitHook: "门外脚步声",
	}, nil)
	for _, want := range []string{"场景目标（这场戏要什么）：拿到账本", "冲突（谁·什么在阻挡）：管家守着", "退出钩子（拉住读者进下一场）：门外脚步声"} {
		if !strings.Contains(got, want) {
			t.Fatalf("卡区段应含 %q，得到：\n%s", want, got)
		}
	}
	// 只有余波的前场也能单独成衔接段
	prev := sceneCardSection(nil, &types.SceneMeta{Outcome: "账本到手", ExitHook: "脚步声逼近"})
	if !strings.Contains(prev, "上一场结果：账本到手") || !strings.Contains(prev, "上一场退出钩子：脚步声逼近") {
		t.Fatalf("前场衔接段渲染不对：%s", prev)
	}
}

// TestNovelChapterScenesGenerate_Gate 卡闸：缺 Goal/Conflict 拒绝并点名场景与缺失
// 字段；allowMissingCard=true 跳过缺卡场、只生成有卡场。
func TestNovelChapterScenesGenerate_Gate(t *testing.T) {
	a, pm, _ := newChapterGateLLMAppReply(t, "生成的正文内容。")
	a.setPM(pm)
	mkCardScene(t, pm, 1, "有卡场", true)
	mkCardScene(t, pm, 1, "无卡场", false)

	// 闸拒绝：点名「无卡场」与缺失字段
	_, err := a.NovelChapterScenesGenerate(1, false)
	if err == nil || !strings.Contains(err.Error(), "无卡场") || !strings.Contains(err.Error(), "目标") {
		t.Fatalf("缺卡应拒绝并点名场景/字段，得到: %v", err)
	}

	// 覆盖路径：跳过缺卡场，有卡场照生成，done 事件带 skipped=1
	snap := subscribeCreateChapterStream(t, "scene-gen-stream")
	if _, err := a.NovelChapterScenesGenerate(1, true); err != nil {
		t.Fatalf("allowMissing 生成应放行: %v", err)
	}
	waitFor(t, 10*time.Second, "逐场景生成协程退出", func() bool {
		a.chapterGenMu.Lock()
		defer a.chapterGenMu.Unlock()
		return len(a.chapterGenCancels) == 0
	})
	// done 帧经 SSE flush 有延迟（协程退出≠订阅端已收到）：轮询事件快照。
	waitFor(t, 10*time.Second, "done 事件到达", func() bool {
		for _, ev := range snap() {
			if ev["type"] == "error" {
				t.Fatalf("不应有 error 事件: %v", ev["error"])
			}
			if ev["type"] == "done" {
				if ev["done"] != float64(1) || ev["skipped"] != float64(1) {
					t.Fatalf("done 事件应 done=1 skipped=1，得到 %v", ev)
				}
				return true
			}
		}
		return false
	})
}

// TestNovelChapterScenesGenerate_FlowAndSceneRefs 两卡两场：逐场落盘（正文含
// 生成内容）→ Stitch→blob（ReadChapter 可读）→ 大纲节点 SceneRefs 回写。
func TestNovelChapterScenesGenerate_FlowAndSceneRefs(t *testing.T) {
	a, pm, _ := newChapterGateLLMAppReply(t, "逐场生成的正文。")
	a.setPM(pm)
	id1 := mkCardScene(t, pm, 1, "第一场", true)
	id2 := mkCardScene(t, pm, 1, "第二场", true)
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 1, Title: "第一章", Branch: ""},
	}}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	snap := subscribeCreateChapterStream(t, "scene-gen-stream")
	if _, err := a.NovelChapterScenesGenerate(1, false); err != nil {
		t.Fatalf("生成应放行: %v", err)
	}
	waitFor(t, 10*time.Second, "逐场景生成协程退出", func() bool {
		a.chapterGenMu.Lock()
		defer a.chapterGenMu.Unlock()
		return len(a.chapterGenCancels) == 0
	})
	waitFor(t, 10*time.Second, "done 事件到达", func() bool {
		for _, ev := range snap() {
			if ev["type"] == "error" {
				t.Fatalf("不应有 error 事件: %v", ev["error"])
			}
			if ev["type"] == "done" {
				if ev["done"] != float64(2) {
					t.Fatalf("done 应为 2 场，得到 %v", ev["done"])
				}
				return true
			}
		}
		return false
	})

	// 两场正文已落盘（每场即落）
	sm := pm.SceneManager(1)
	for _, id := range []string{id1, id2} {
		sc, err := sm.Read(id)
		if err != nil {
			t.Fatalf("读场景 %s: %v", id, err)
		}
		if !strings.Contains(sc.Content, "逐场生成的正文") {
			t.Fatalf("场景 %s 正文未落盘: %q", id, sc.Content)
		}
	}
	// Stitch→blob 可读
	full, err := pm.ReadChapter(1)
	if err != nil || !strings.Contains(full, "逐场生成的正文") {
		t.Fatalf("整章 blob 应含生成正文: %q err=%v", full, err)
	}
	// SceneRefs 回写
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		t.Fatalf("读大纲: %v", err)
	}
	if got := of.Nodes[0].SceneRefs; len(got) != 2 || got[0] != id1 || got[1] != id2 {
		t.Fatalf("SceneRefs 应回写场景 ID 序，得到 %v", got)
	}
}

// TestSceneCardInjectedIntoPrompt 卡字段与上一场衔接必须进入生成 prompt
// （requests 通道捕获 HTTP body 断言——刀2 核心验收「场景卡字段参与生成」）。
func TestSceneCardInjectedIntoPrompt(t *testing.T) {
	a, pm, requests := newChapterGateLLMAppReply(t, "正文。")
	a.setPM(pm)
	mkCardScene(t, pm, 1, "前场", true)
	mkCardScene(t, pm, 1, "主场", true)
	metas, merr := pm.SceneManager(1).List()
	if merr != nil || len(metas) != 2 {
		t.Fatalf("列场景: %v (%d)", merr, len(metas))
	}

	if _, err := a.GenerateScene(1, metas[1].ID, "写紧凑一点", 100); err != nil {
		t.Fatalf("GenerateScene: %v", err)
	}
	select {
	case body := <-requests:
		text := string(body)
		for _, want := range []string{"主场的目标", "主场的阻挡", "上一场结果：前场后状态改变", "上一场退出钩子：前场的钩子"} {
			if !strings.Contains(text, want) {
				t.Fatalf("prompt 应含卡区段 %q（卡字段/衔接参与生成）", want)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("未捕获生成请求")
	}
}

// TestNovelSceneRewrite_OthersUntouched 单场景重写：目标场正文被替换，他场字节
// 不动；版本库留痕 mode=scene。
func TestNovelSceneRewrite_OthersUntouched(t *testing.T) {
	a, pm, _ := newChapterGateLLMAppReply(t, "重写后的这场戏。")
	a.setPM(pm)
	sm := pm.SceneManager(1)
	mkCardScene(t, pm, 1, "前场", true)
	midID := mkCardScene(t, pm, 1, "中场", true)
	mkCardScene(t, pm, 1, "后场", true)
	// 预置三场正文
	metas, _ := sm.List()
	for i, m := range metas {
		sc, _ := sm.Read(m.ID)
		sc.Content = "第" + strings.Repeat("一二三", i+1) + "场原正文"
		if err := sm.Write(sc); err != nil {
			t.Fatalf("预置正文: %v", err)
		}
	}
	syncBlobFromScenes(pm, 1)
	beforeOthers := map[string]string{}
	for _, m := range metas {
		if m.ID == midID {
			continue
		}
		sc, _ := sm.Read(m.ID)
		beforeOthers[m.ID] = sc.Content
	}

	if _, err := a.NovelSceneRewrite(1, midID, "把冲突改成暗中试探"); err != nil {
		t.Fatalf("NovelSceneRewrite: %v", err)
	}

	// 目标场已替换
	mid, _ := sm.Read(midID)
	if mid.Content != "重写后的这场戏。" {
		t.Fatalf("中场应被重写，得到 %q", mid.Content)
	}
	// 他场字节不动
	for id, want := range beforeOthers {
		sc, _ := sm.Read(id)
		if sc.Content != want {
			t.Fatalf("他场 %s 被改动：before=%q after=%q", id, want, sc.Content)
		}
	}
	// 版本库 mode=scene 留痕
	versions, err := pm.ListRewriteVersions(1)
	if err != nil || len(versions) == 0 {
		t.Fatalf("应留重写版本: %v", err)
	}
	found := false
	for _, v := range versions {
		if v.Mode == types.RewriteModeScene {
			found = true
		}
	}
	if !found {
		t.Fatalf("版本应含 mode=scene，得到 %+v", versions)
	}
}
