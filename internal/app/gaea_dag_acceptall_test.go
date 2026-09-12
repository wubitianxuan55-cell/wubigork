package app

// GaeaDagAcceptAll 一键验收测试（成品直出首刀，DeliverableRegistry 用户面）：
// 全 done 一键全收、部分已验收后只收余量、无可验收提示非错误。验收语义
// （状态翻转+AcceptedAt+记忆回流）与单验收共用 dagAcceptMemoryWrite 同真源。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/dag"
)

func TestDagAcceptAll(t *testing.T) {
	journalDir := injectDagEnv(t)
	id := planChain(t)

	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		var node string
		switch {
		case strings.Contains(prompt, "读取三份月度 xlsx"):
			node = "read"
		case strings.Contains(prompt, "透视汇总出 summary.xlsx"):
			node = "pivot"
		case strings.Contains(prompt, "嵌图表出报告 docx"):
			node = "report"
		}
		appendCard(t, journalDir, "sa_"+node, "docs/"+node+".xlsx")
		return "ok: " + node, "sa_" + node, nil
	})

	a := &App{}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	waitDagDone(t, id)

	// 全 done：一键全收，产物计数与记忆回流逐节点到位。
	out, err := a.GaeaDagAcceptAll(id)
	if err != nil {
		t.Fatalf("GaeaDagAcceptAll: %v", err)
	}
	if !strings.Contains(out, "已一键验收 3 个节点") || !strings.Contains(out, "3 件产物") {
		t.Fatalf("一键验收文案: %q", out)
	}
	r := waitDagDone(t, id)
	for _, nid := range []string{"read", "pivot", "report"} {
		if n := dagNode(t, r, nid); n.Status != dag.StatusAccepted || n.AcceptedAt == "" {
			t.Fatalf("节点 %s 应 accepted+AcceptedAt: %+v", nid, n)
		}
		if m, ok := officeStoreOverride.Get(nid); !ok || m.Title == "" {
			t.Fatalf("节点 %s 记忆回流缺失", nid)
		}
	}

	// 已全验收后再点：无可验收，明确提示非错误。
	out, err = a.GaeaDagAcceptAll(id)
	if err != nil {
		t.Fatalf("重复一键验收应 nil error: %v", err)
	}
	if !strings.Contains(out, "没有可一键验收") {
		t.Fatalf("空验收文案: %q", out)
	}

	// 混合态：重跑 read（accepted 退回 done→再完成），一键只收余量 1 个。
	if _, err := a.GaeaDagNodeRun(id, "read"); err != nil {
		t.Fatalf("重跑 read: %v", err)
	}
	waitDagDone(t, id)
	out, err = a.GaeaDagAcceptAll(id)
	if err != nil {
		t.Fatalf("混合态一键验收: %v", err)
	}
	if !strings.Contains(out, "已一键验收 1 个节点") {
		t.Fatalf("混合态只应收 done 余量: %q", out)
	}
	r = waitDagDone(t, id)
	if n := dagNode(t, r, "read"); n.Status != dag.StatusAccepted {
		t.Fatalf("read 应重新 accepted: %+v", n)
	}
}

// TestDagAcceptAllSingleAcceptRegression 单验收回归：重构共用记忆回写核心后
// 单节点路径语义不变（状态/记忆/文案）。
func TestDagAcceptAllSingleAcceptRegression(t *testing.T) {
	journalDir := injectDagEnv(t)
	id := planChain(t)
	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		_ = journalDir
		return "ok", "sa_x", nil
	})
	a := &App{}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	waitDagDone(t, id)
	// 非法状态拒绝语义不变。
	if _, err := a.GaeaDagNodeAccept(id, "no-such"); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("不存在节点应报错: %v", err)
	}
	if out, err := a.GaeaDagNodeAccept(id, "read"); err != nil || !strings.Contains(out, "已验收") {
		t.Fatalf("单验收文案: %q %v", out, err)
	}
	r := waitDagDone(t, id)
	if n := dagNode(t, r, "read"); n.Status != dag.StatusAccepted {
		t.Fatalf("单验收状态: %+v", n)
	}
	if _, err := a.GaeaDagNodeAccept(id, "read"); err == nil || !strings.Contains(err.Error(), "只验收已完成") {
		t.Fatalf("重复验收应拒绝: %v", err)
	}
}
