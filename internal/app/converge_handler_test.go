package app

// 长篇刀4测试：质量收敛闭环（规格 进度计划/gaea-novel-converge-20260930.md §1）。
// 覆盖：判据矩阵 / 预检 dry-run / 已收敛零轮直答 / 无改善回滚（磁盘回轮前+
// stopped 事件）/ 定向指令拼装。

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/project"
)

// TestConvergeCheck 判据纯函数矩阵：S3 不阻断、高分不收敛、空正文 S1、达标收敛。
func TestConvergeCheck(t *testing.T) {
	// 好文本：低 AI 味 + 零确定性信号
	good := "他推门进来，把伞收在墙角。雨还没停。她没抬头，手里的针线也没停。"
	ck := convergeCheck(good, 40)
	if !ck.Converged {
		t.Fatalf("低分零信号应收敛：score=%d s1s2=%d %+v", ck.TasteScore, len(ck.S1S2), ck.S1S2)
	}
	// 空正文：chapter_empty 是 S1 → 不收敛
	ck = convergeCheck("   \n  ", 40)
	if ck.Converged || len(ck.S1S2) == 0 {
		t.Fatalf("空正文应因 S1 不收敛：%+v", ck.S1S2)
	}
	// 高分文本不收敛（分数维度）
	bad := strings.Repeat("他感到无比的震惊。此外，值得注意的是，这一切仿佛命运的安排。", 20)
	ck = convergeCheck(bad, 5) // 目标 5 分：几乎必然未达
	if ck.Converged {
		t.Fatalf("高分文本按严格目标不应收敛：score=%d", ck.TasteScore)
	}
	// S3 不阻断：超长段落是 S3——构造单段超长但低 AI 味
	longPara := strings.Repeat("她说了一句话。", 400) // 无 AI 味高频词、单段超长
	ck = convergeCheck(longPara, 60)
	hasS3 := false
	for _, is := range ck.S3 {
		if is.Code == "paragraph_too_long" {
			hasS3 = true
		}
	}
	if !hasS3 {
		t.Fatalf("超长段落应落 S3 提示（不阻断）：s3=%+v", ck.S3)
	}
	if len(ck.S1S2) != 0 && ck.TasteScore <= 60 {
		t.Fatalf("S3 不应计入阻断集：s1s2=%+v", ck.S1S2)
	}
}

// TestConvergePreview 预检 dry-run：返回判据状态与计划轮次；空章诚实报错。
func TestConvergePreview(t *testing.T) {
	a, pm, _ := newChapterGateLLMAppReply(t, "无关。")
	a.setPM(pm)
	if err := pm.WriteChapter(1, "他推门进来。雨还没停。她没抬头。"); err != nil {
		t.Fatal(err)
	}
	pv, err := a.NovelChapterConvergePreview(1, 0)
	if err != nil {
		t.Fatalf("预检: %v", err)
	}
	if pv["converged"] != true || pv["planRounds"] != convergeDefaultMaxRounds {
		t.Fatalf("好文本预检应 converged+计划轮次：%v", pv)
	}
	// 无正文章节：诚实报错
	a2, pm2, _ := newChapterGateLLMAppReply(t, "无关。")
	a2.setPM(pm2)
	if _, err := a2.NovelChapterConvergePreview(9, 0); err == nil {
		t.Fatal("无正文章节应报错（收敛修补需要已有正文）")
	}
}

// TestConvergeAlreadyConverged 已收敛章零轮直答：不烧模型、不写盘、无流事件。
func TestConvergeAlreadyConverged(t *testing.T) {
	snap := subscribeCreateChapterStream(t, convergeStreamChannel)
	a, pm, _ := newChapterGateLLMAppReply(t, "不应被调用。")
	a.setPM(pm)
	if err := pm.WriteChapter(1, "他推门进来。雨还没停。她没抬头，手里的针线也没停。"); err != nil {
		t.Fatal(err)
	}
	res, err := a.NovelChapterConverge(1, 2, 40)
	if err != nil {
		t.Fatalf("启动: %v", err)
	}
	if res["alreadyConverged"] != true || res["rounds"] != 0 {
		t.Fatalf("已收敛应零轮直答：%v", res)
	}
	time.Sleep(150 * time.Millisecond) // 无后台协程；留窗口确认无事件
	if evs := snap(); len(evs) != 0 {
		t.Fatalf("零轮直答不应有流事件：%v", evs)
	}
}

// TestConvergeNoImproveRollback 无改善回滚：坏文本 + 假站返回不改善内容 →
// rewriteUnit 安全闸不落盘 → 本轮无变化 → no-improve 停 + 磁盘回轮前（字节不变）。
func TestConvergeNoImproveRollback(t *testing.T) {
	snap := subscribeCreateChapterStream(t, convergeStreamChannel)
	// 假站返回与原文同劣的内容（安全闸必然拒绝，本轮零落盘）
	a, pm, _ := newChapterGateLLMAppReply(t, "他感到无比的震惊。此外，这一切仿佛命运的安排。")
	a.setPM(pm)
	original := strings.Repeat("他感到无比的震惊。此外，值得注意的是，这一切仿佛命运的安排。", 8)
	if err := pm.WriteChapter(1, original); err != nil {
		t.Fatal(err)
	}
	beforeBytes, _ := readChapterBytes(pm, 1)

	res, err := a.NovelChapterConverge(1, 2, 1) // 目标 1 分：必然未达 → 走轮次
	if err != nil {
		t.Fatalf("启动: %v", err)
	}
	if res["started"] != true {
		t.Fatalf("应启动：%v", res)
	}
	// 等终态事件
	deadline := time.Now().Add(30 * time.Second)
	var stopped map[string]interface{}
	for time.Now().Before(deadline) {
		for _, ev := range snap() {
			if ev["type"] == "stopped" || ev["type"] == "converged" || ev["type"] == "error" {
				stopped = ev
			}
		}
		if stopped != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if stopped == nil {
		t.Fatal("未收到终态事件")
	}
	if stopped["type"] == "error" {
		t.Fatalf("不应 error：%v", stopped["error"])
	}
	if stopped["type"] != "stopped" || stopped["reason"] != "no-improve" {
		t.Fatalf("本轮无改善应 no-improve 停：%v", stopped)
	}
	afterBytes, _ := readChapterBytes(pm, 1)
	if string(afterBytes) != string(beforeBytes) {
		t.Fatalf("no-improve 回滚后磁盘应与轮前逐字节一致（before=%dB after=%dB）", len(beforeBytes), len(afterBytes))
	}
}

// TestConvergeIssuesInstruction 定向指令拼装：S1/S2 逐条列出 + AI 味要点 + 空清单。
func TestConvergeIssuesInstruction(t *testing.T) {
	ck := ConvergeCheck{
		TasteScore: 72,
		S1S2: []novelgate.Issue{
			{Code: "telegraph_style", Severity: "S2", Message: "疑似电报体", Evidence: "例句"},
		},
	}
	got := convergeIssuesInstruction(ck, 40)
	for _, want := range []string{"[telegraph_style] 疑似电报体（例句）", "AI 味 72 分（目标 ≤40）"} {
		if !strings.Contains(got, want) {
			t.Fatalf("指令应含 %q：%s", want, got)
		}
	}
	if got := convergeIssuesInstruction(ConvergeCheck{}, 40); got != "" {
		t.Fatalf("无问题清单应返回空指令：%q", got)
	}
}

// readChapterBytes 读章字节（回滚断言用）。
func readChapterBytes(pm *project.Manager, num int) ([]byte, error) {
	return os.ReadFile(pm.ChapterPath(num))
}

// TestConvergeWriteBackUnits 回滚直测：写脏磁盘 → 按轮前单元快照回滚 → 字节还原。
// （no-improve 集成场景里安全闸常使回滚成为 no-op，此直测钉住回滚函数本身。）
func TestConvergeWriteBackUnits(t *testing.T) {
	a, pm, _ := newChapterGateLLMAppReply(t, "无关。")
	a.setPM(pm)
	original := "他推门进来。雨还没停。"
	if err := pm.WriteChapter(1, original); err != nil {
		t.Fatal(err)
	}
	before, _ := readChapterBytes(pm, 1)
	// 写脏（模拟修补落盘）
	if err := pm.WriteChapter(1, "被磨坏的稿子。"); err != nil {
		t.Fatal(err)
	}
	dirty, _ := readChapterBytes(pm, 1)
	if string(dirty) == string(before) {
		t.Fatal("前置失败：写脏未生效")
	}
	// 回滚到轮前单元
	if err := convergeWriteBackUnits(pm, 1, []convergeUnit{{id: "", isScene: false, text: original}}); err != nil {
		t.Fatalf("回滚: %v", err)
	}
	after, _ := readChapterBytes(pm, 1)
	if string(after) != string(before) {
		t.Fatalf("回滚后应逐字节还原轮前（%q != %q）", after, before)
	}
}
