package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagramGenFramework(t *testing.T) {
	out := filepath.Join(t.TempDir(), "framework.png")
	args := map[string]interface{}{
		"title": "gaea 三层架构框架图",
		"kind":  "framework",
		"nodes": []map[string]interface{}{
			{"id": "app", "label": "应用层：项目管理 / 办公助手 / 知识库", "level": 0, "group": "应用层"},
			{"id": "svc", "label": "服务层：模型调度 / 文档引擎 / 记忆中枢", "level": 1, "group": "服务层"},
			{"id": "data", "label": "数据层：本地模型 / SQLite / 文件存储", "level": 2, "group": "数据层"},
		},
		"output": out,
	}
	raw, _ := json.Marshal(args)
	res, err := (diagramGen{}).Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var r struct {
		OK   bool   `json:"ok"`
		Path string `json:"output"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if !r.OK {
		t.Fatalf("result not ok: %s", res)
	}
	info, err := os.Stat(r.Path)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if info.Size() < 5000 {
		t.Fatalf("output suspiciously small: %d bytes", info.Size())
	}
}

func TestDiagramGenFlow(t *testing.T) {
	out := filepath.Join(t.TempDir(), "flow.png")
	args := map[string]interface{}{
		"title": "项目周报流程",
		"kind":  "flow",
		"nodes": []map[string]interface{}{
			{"id": "a", "label": "收集本周数据"},
			{"id": "b", "label": "汇总完成事项"},
			{"id": "c", "label": "输出周报文档"},
		},
		"edges": []map[string]interface{}{
			{"from": "a", "to": "b", "label": "整理"},
			{"from": "b", "to": "c", "label": "生成"},
		},
		"output": out,
	}
	raw, _ := json.Marshal(args)
	res, err := (diagramGen{}).Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var r struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if !r.OK {
		t.Fatalf("result not ok: %s", res)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}

func TestDiagramGenValidation(t *testing.T) {
	if _, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(`{"nodes":[]}`)); err == nil {
		t.Fatal("expected empty-nodes error")
	}
	if _, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(`{"nodes":[{"id":"a","label":"x"}],"kind":"bad"}`)); err == nil {
		t.Fatal("expected bad-kind error")
	}
}

// TestDiagramGenFlowValidation flow 模式参数校验：环检测、layout 取值。
func TestDiagramGenFlowValidation(t *testing.T) {
	// 循环依赖 → 参数错误
	_, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(
		`{"kind":"flow","nodes":[{"id":"a","label":"甲"},{"id":"b","label":"乙"},{"id":"c","label":"丙"}],` +
			`"edges":[{"from":"a","to":"b"},{"from":"b","to":"c"},{"from":"c","to":"a"}]}`))
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("错误信息应含 循环依赖，实际: %v", err)
	}
	// 自环也是环
	if _, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(
		`{"kind":"flow","nodes":[{"id":"a","label":"甲"}],"edges":[{"from":"a","to":"a"}]}`)); err == nil {
		t.Fatal("expected self-loop cycle error")
	} else if !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("错误信息应含 循环依赖，实际: %v", err)
	}
	// 非法 layout
	if _, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(
		`{"kind":"flow","nodes":[{"id":"a","label":"甲"}],"layout":"diagonal"}`)); err == nil {
		t.Fatal("expected bad-layout error")
	} else if !strings.Contains(err.Error(), "layout") {
		t.Fatalf("错误信息应含 layout，实际: %v", err)
	}
}

// TestComputeFlowLevels 拓扑分层纯函数：链式/分支汇合/最长路径/边界/环检测。
func TestComputeFlowLevels(t *testing.T) {
	assertLevels := func(t *testing.T, got map[string]int, want map[string]int) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("levels 数量不符: got %v want %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("节点 %q level = %d, want %d (got %v)", k, got[k], v, got)
			}
		}
	}

	// 链式：a→b→c
	levels, err := computeFlowLevels([]string{"a", "b", "c"}, [][2]string{{"a", "b"}, {"b", "c"}})
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	assertLevels(t, levels, map[string]int{"a": 0, "b": 1, "c": 2})

	// 分支/汇合：a→b, a→c, b→d, c→d
	levels, err = computeFlowLevels([]string{"a", "b", "c", "d"},
		[][2]string{{"a", "b"}, {"a", "c"}, {"b", "d"}, {"c", "d"}})
	if err != nil {
		t.Fatalf("diamond: %v", err)
	}
	assertLevels(t, levels, map[string]int{"a": 0, "b": 1, "c": 1, "d": 2})

	// 多前驱取 max：a→b→c→e, a→d→e → e=3
	levels, err = computeFlowLevels([]string{"a", "b", "c", "d", "e"},
		[][2]string{{"a", "b"}, {"b", "c"}, {"c", "e"}, {"a", "d"}, {"d", "e"}})
	if err != nil {
		t.Fatalf("longest-path: %v", err)
	}
	assertLevels(t, levels, map[string]int{"a": 0, "b": 1, "c": 2, "d": 1, "e": 3})

	// 空 edges：全部 level 0
	levels, err = computeFlowLevels([]string{"a", "b", "c"}, nil)
	if err != nil {
		t.Fatalf("empty edges: %v", err)
	}
	assertLevels(t, levels, map[string]int{"a": 0, "b": 0, "c": 0})

	// 单节点
	levels, err = computeFlowLevels([]string{"solo"}, nil)
	if err != nil {
		t.Fatalf("single node: %v", err)
	}
	assertLevels(t, levels, map[string]int{"solo": 0})

	// 引用未知节点的边忽略
	levels, err = computeFlowLevels([]string{"a"}, [][2]string{{"x", "y"}, {"a", "ghost"}})
	if err != nil {
		t.Fatalf("unknown-node edges: %v", err)
	}
	assertLevels(t, levels, map[string]int{"a": 0})

	// 环：a→b→c→a
	if _, err = computeFlowLevels([]string{"a", "b", "c"},
		[][2]string{{"a", "b"}, {"b", "c"}, {"c", "a"}}); err == nil {
		t.Fatal("expected cycle error")
	} else if !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("错误信息应含 循环依赖，实际: %v", err)
	}

	// 自环
	if _, err = computeFlowLevels([]string{"a", "b"}, [][2]string{{"a", "a"}}); err == nil {
		t.Fatal("expected self-loop cycle error")
	}

	// 多连通分量（各自独立成层）
	levels, err = computeFlowLevels([]string{"a", "b", "x", "y"},
		[][2]string{{"a", "b"}, {"x", "y"}})
	if err != nil {
		t.Fatalf("multi-component: %v", err)
	}
	assertLevels(t, levels, map[string]int{"a": 0, "b": 1, "x": 0, "y": 1})
}

// TestWrapLabel 折行纯函数： rune 计数、边界值。
func TestWrapLabel(t *testing.T) {
	cases := []struct {
		in       string
		maxChars int
		want     string
	}{
		{"短标签", 14, "短标签"},
		{"", 14, ""},
		{"正好十四个字符的标签哦", 14, "正好十四个字符的标签哦"},                    // 11 字
		{"一二三四五六七八九十一二三四", 14, "一二三四五六七八九十一二三四"}, // 恰 14 字不折
		{"一二三四五六七八九十一二三四五六", 14, "一二三四五六七八九十一二三四\n五六"},     // 16 字 → 14+2
		{"一二三四五六七八九十一二三四五六七", 14, "一二三四五六七八九十一二三四\n五六七"},    // 17 字 → 14+3
		{"abcdefgh", 3, "abc\ndef\ngh"},
		{"超长中文标签自动折行处理验证用例字符串", 14, "超长中文标签自动折行处理验证\n用例字符串"}, // 19 字 → 14+5
	}
	for _, tc := range cases {
		if got := wrapLabel(tc.in, tc.maxChars); got != tc.want {
			t.Errorf("wrapLabel(%q, %d) = %q, want %q", tc.in, tc.maxChars, got, tc.want)
		}
	}
	// 行数验证：20 字 → 2 行
	if got := wrapLabel("并行任务A：数据清洗、格式转换与质量校验", 14); strings.Count(got, "\n") != 1 {
		t.Errorf("20 字标签应折为 2 行，got %q", got)
	}
}

// diagramRenderSkip 环境（Python/matplotlib）不可用时跳过渲染冒烟。
func diagramRenderSkip(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "Python") || strings.Contains(msg, "matplotlib") {
		t.Skipf("python/matplotlib 不可用，跳过渲染冒烟: %v", err)
	}
}

// TestDiagramGenFlowBranchLayouts 冒烟：分支/汇合 flow 的 vertical + horizontal 各渲染一张。
func TestDiagramGenFlowBranchLayouts(t *testing.T) {
	if _, err := lookPathFirst([]string{"python", "python3"}); err != nil {
		t.Skip("python 不可用，跳过渲染冒烟")
	}
	for _, layout := range []string{"vertical", "horizontal"} {
		t.Run(layout, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "flow_"+layout+".png")
			args := map[string]interface{}{
				"title":  "分支汇合流程（" + layout + "）",
				"kind":   "flow",
				"layout": layout,
				"nodes": []map[string]interface{}{
					{"id": "start", "label": "开始：领取任务"},
					{"id": "clean", "label": "并行任务A：数据清洗、格式转换与质量校验"},
					{"id": "model", "label": "并行任务B：模型推理"},
					{"id": "end", "label": "汇总结果并归档"},
				},
				"edges": []map[string]interface{}{
					{"from": "start", "to": "clean", "label": "分流"},
					{"from": "start", "to": "model", "label": "分流"},
					{"from": "clean", "to": "end"},
					{"from": "model", "to": "end", "label": "汇合"},
				},
				"output": out,
			}
			raw, _ := json.Marshal(args)
			res, err := (diagramGen{}).Execute(context.Background(), raw)
			diagramRenderSkip(t, err)
			if err != nil {
				t.Fatalf("Execute(%s): %v", layout, err)
			}
			var r struct {
				OK        bool   `json:"ok"`
				Path      string `json:"output"`
				SizeBytes int64  `json:"size_bytes"`
			}
			if err := json.Unmarshal([]byte(res), &r); err != nil {
				t.Fatalf("parse result: %v", err)
			}
			if !r.OK {
				t.Fatalf("result not ok: %s", res)
			}
			info, err := os.Stat(r.Path)
			if err != nil {
				t.Fatalf("output missing: %v", err)
			}
			if info.Size() < 5000 || info.Size() != r.SizeBytes {
				t.Fatalf("output size abnormal: file=%d reported=%d", info.Size(), r.SizeBytes)
			}
		})
	}
}
