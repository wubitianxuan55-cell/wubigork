package agent

// subagent_files_note_test.go — 2026-11-28 黄甲反馈回归钉：子代理没跑完时，失败
// 回执带上它已经动过的文件。实测四路子代理断流后，主控只能手工比对章节文件的
// 字节数确认有没有半成品写入。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	builtin "github.com/gaea/gaea/internal/gaea/tool/builtin"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func TestSubagentFilesNote(t *testing.T) {
	if subagentFilesNote(nil) != "" || subagentFilesNote(&TurnResult{}) != "" {
		t.Fatal("没动过文件不该有清单")
	}
	note := subagentFilesNote(&TurnResult{
		FilesCreated:  []string{"a.md", "a.md"},
		FilesModified: []string{"a.md", "b.md"},
	})
	for _, want := range []string{"[已动过的文件]", "新建 a.md", "改写/移动/删除 b.md"} {
		if !strings.Contains(note, want) {
			t.Errorf("清单应含 %q：%s", want, note)
		}
	}
	if strings.Contains(note, "改写/移动/删除 a.md") || strings.Count(note, "新建 a.md") != 1 {
		t.Errorf("新建后又改的文件只报一次（归新建）：%s", note)
	}

	many := make([]string, 15)
	for i := range many {
		many[i] = "f" + string(rune('a'+i)) + ".md"
	}
	note = subagentFilesNote(&TurnResult{FilesCreated: many})
	if strings.Count(note, "新建 ") != 12 || !strings.Contains(note, "…等共 15 件") {
		t.Errorf("超过 12 件应折叠并给出总数：%s", note)
	}
}

// 端到端：子代理用 write_file 写了一个文件之后模型流失败——失败回执带上文件清单
// （实测形态：读到一半断流，主控不知道章节文件有没有被写成半成品）。
func TestSubagentFailureReceiptListsFilesWritten(t *testing.T) {
	ws := t.TempDir()
	reg := tool.NewRegistry()
	for _, tl := range (builtin.Workspace{Dir: ws}).Tools("write_file", "edit_file") {
		reg.Add(tl)
	}
	// 第二步：改写另一个文件（成功）；第三步：写第三个文件被拦（没有写策略时
	// 一律拒）；第四步：模型流失败。
	mp := testutil.NewMock("mock",
		testutil.Turn{ToolCalls: []provider.ToolCall{{
			ID: "c1", Name: "write_file",
			Arguments: `{"path":"ch03.md","content":"# 第 3 章\n\n正文一段。\n"}`,
		}}},
		testutil.Turn{ToolCalls: []provider.ToolCall{{
			ID: "c2", Name: "edit_file",
			Arguments: `{"path":"ch03.md","old_string":"正文一段。","new_string":"正文一段（扩写）。"}`,
		}}},
		testutil.Turn{StreamError: errors.New("流读取错误: read tcp 172.19.0.1:64659->198.18.0.89:443: wsarecv: An existing connection was forcibly closed by the remote host.")},
	)
	sess := NewSession("sys")
	sess.SetSpace(SpaceFromContext(context.Background())) // 防穿越闸：子会话空间须与运行上下文一致
	_, err := runSubAgentInternal(context.Background(), mp, reg, sess, "写第 3 章",
		Options{AccessPolicy: tool.NewPathAccessPolicy(ws, nil, nil, nil)}, event.Discard, nil)
	if err == nil {
		t.Fatal("子代理流失败应报错")
	}
	target := filepath.Join(ws, "ch03.md")
	for _, want := range []string{"sub-agent error before finishing: 流读取错误", "[已动过的文件]", "新建 ch03.md"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("失败回执应含 %q：\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "改写/移动/删除 ch03.md") {
		t.Errorf("先建后改的文件只报一次（归新建）：%v", err)
	}
	if raw, rerr := os.ReadFile(target); rerr != nil || !strings.Contains(string(raw), "扩写") {
		t.Fatalf("断流前写入并改过的内容应在盘上：%q %v", raw, rerr)
	}
}

// 没写策略时 write_file 一律被拦：被拦的调用不算「动过的文件」，失败回执不带清单。
func TestSubagentFailureReceiptOmitsBlockedWrites(t *testing.T) {
	ws := t.TempDir()
	reg := tool.NewRegistry()
	for _, tl := range (builtin.Workspace{Dir: ws}).Tools("write_file") {
		reg.Add(tl)
	}
	mp := testutil.NewMock("mock",
		testutil.Turn{ToolCalls: []provider.ToolCall{{
			ID: "c1", Name: "write_file", Arguments: `{"path":"ch03.md","content":"x"}`,
		}}},
		testutil.Turn{StreamError: errors.New("boom")},
	)
	sess := NewSession("sys")
	sess.SetSpace(SpaceFromContext(context.Background()))
	_, err := runSubAgentInternal(context.Background(), mp, reg, sess, "写第 3 章", Options{}, event.Discard, nil)
	if err == nil {
		t.Fatal("子代理流失败应报错")
	}
	if strings.Contains(err.Error(), "[已动过的文件]") {
		t.Fatalf("被拦的写入不该进文件清单：%v", err)
	}
	if _, rerr := os.Stat(filepath.Join(ws, "ch03.md")); !os.IsNotExist(rerr) {
		t.Fatalf("被拦的写入不该落盘：%v", rerr)
	}
}
