package outline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// ── 故事主线写入路径（规格 §1.11 / §7.1-1）─────────────────────
//
// 回归对象：Continue/ExpandNode 解析模型回复后只回收 .Nodes，把 story_thread
// 丢弃 → 五卷规划的【故事主线 P0】输入恒为空。以下用例锁定两条不变式：
//  1) 回复含 story_thread → 回收并随 outline.json 落盘往返；
//  2) 回复不含（或为空）story_thread → 已有主线保持不变（绝不擦除）。

// stubLLM 最小 ai.LLMClient：无论入参一律返回固定回复。
type stubLLM struct {
	reply string
	err   error
}

func (s stubLLM) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	ch := make(chan ai.SSEChunk, 1)
	ch <- ai.SSEChunk{Done: true}
	close(ch)
	return ch, nil
}

func (s stubLLM) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return s.reply, s.err
}

func (s stubLLM) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	return s.reply, s.err
}

func (s stubLLM) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

// testTemplates 写入最小 RTCO 模板：Continue/ExpandNode 需要 eng.Get 命中模板名。
func testTemplates(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := map[string]string{
		"outline-continue": `{"name":"outline-continue","system":"sys","task":"task",` +
			`"input_sections":{"story_thread":{"priority":"P0","label":"故事主线"}},` +
			`"output":{"format":"json"}}`,
		"outline-expand": `{"name":"outline-expand","system":"sys","task":"task",` +
			`"input_sections":{"story_thread":{"priority":"P0","label":"故事主线"}},` +
			`"output":{"format":"json"}}`,
	}
	for name, body := range tmpl {
		if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// newTestAgent 构造指向真实临时项目的 Agent（pm 走既有 Create 目录口径）。
func newTestAgent(t *testing.T, reply string) (*Agent, *project.Manager) {
	t.Helper()
	root := t.TempDir()
	pm, err := project.Create(filepath.Join(root, "novel"), "测试书", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	eng := prompt.NewEngine(testTemplates(t))
	return New(stubLLM{reply: reply}, pm, &config.Config{}, eng), pm
}

// TestContinue_CollectsStoryThread 回复含 story_thread（多行）→ 回收 + 落盘往返。
func TestContinue_CollectsStoryThread(t *testing.T) {
	const thread = "核心冲突：旧神复苏 vs 人类文明存续\n叙事方向：群像视角，双线收束"
	reply := `{"story_thread":"核心冲突：旧神复苏 vs 人类文明存续\n叙事方向：群像视角，双线收束",` +
		`"nodes":[` +
		`{"id":"vol_1","title":"第一卷·起·残响","summary":"s1","children":[{"id":"ch_1_1","title":"第1章"}]},` +
		`{"id":"vol_2","title":"第二卷·承·裂痕","summary":"s2"},` +
		`{"id":"vol_3","title":"第三卷·转·逆潮","summary":"s3"},` +
		`{"id":"vol_4","title":"第四卷·合·归墟","summary":"s4"},` +
		`{"id":"vol_5","title":"第五卷·终·新火","summary":"s5"}]}`

	a, pm := newTestAgent(t, reply)
	of, err := a.Continue(context.Background(), 5)
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	if of.StoryThread != thread {
		t.Fatalf("主线应原样回收（含多行原文）:\n got %q\nwant %q", of.StoryThread, thread)
	}
	if len(of.Nodes) != 5 {
		t.Fatalf("五卷应保留 5 卷: got %d", len(of.Nodes))
	}

	// 落盘往返：走既有 OutlineFile 持久化路径（outline.json），不新增文件。
	if err := a.Save(of); err != nil {
		t.Fatalf("Save: %v", err)
	}
	back, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("ReadOutlines: %v", err)
	}
	if back.StoryThread != thread {
		t.Fatalf("落盘往返主线丢失:\n got %q\nwant %q", back.StoryThread, thread)
	}
	if len(back.Nodes) != 5 {
		t.Fatalf("落盘往返节点丢失: got %d", len(back.Nodes))
	}
	raw, err := os.ReadFile(filepath.Join(pm.Dir, "outline.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"story_thread"`) {
		t.Fatalf("outline.json 应含 story_thread 键: %s", raw)
	}
}

// TestContinue_KeepsStoryThreadWhenReplyOmits 回复不含 story_thread
// → 已有主线保持不变（模型漏输出不得把主线清空）。
func TestContinue_KeepsStoryThreadWhenReplyOmits(t *testing.T) {
	const existing = "既有主线：不要被擦除"
	a, pm := newTestAgent(t, `{"nodes":[{"id":"vol_6","title":"第六卷","summary":"s6"}]}`)
	if err := a.Save(&types.OutlineFile{StoryThread: existing, Nodes: []types.OutlineNode{}}); err != nil {
		t.Fatal(err)
	}

	of, err := a.Continue(context.Background(), 1) // 非 5 = 追加模式
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	if of.StoryThread != existing {
		t.Fatalf("回复缺字段应保留已有主线: got %q", of.StoryThread)
	}
	if err := a.Save(of); err != nil {
		t.Fatal(err)
	}
	back, err := pm.ReadOutlines()
	if err != nil {
		t.Fatal(err)
	}
	if back.StoryThread != existing {
		t.Fatalf("落盘后主线被擦除: got %q", back.StoryThread)
	}
}

// TestContinue_KeepsStoryThreadWhenReplyEmpty 回复显式给出空/空白 story_thread
// → 同样不擦除已有主线（空串与缺字段同口径）。
func TestContinue_KeepsStoryThreadWhenReplyEmpty(t *testing.T) {
	const existing = "既有主线：空白值也不许清空"
	for name, empty := range map[string]string{"empty": "", "blank": "     "} {
		t.Run(name, func(t *testing.T) {
			reply := `{"story_thread":"` + empty + `","nodes":[{"id":"vol_7","title":"第七卷"}]}`
			a, _ := newTestAgent(t, reply)
			if err := a.Save(&types.OutlineFile{StoryThread: existing, Nodes: []types.OutlineNode{}}); err != nil {
				t.Fatal(err)
			}
			of, err := a.Continue(context.Background(), 1)
			if err != nil {
				t.Fatalf("Continue: %v", err)
			}
			if of.StoryThread != existing {
				t.Fatalf("空白主线不应覆盖: got %q", of.StoryThread)
			}
		})
	}
}

// TestExpandNode_CollectsStoryThread 展开回复携带顶层 story_thread → 回收，
// 且 children 追加行为不变。
func TestExpandNode_CollectsStoryThread(t *testing.T) {
	const thread = "新主线：展开后细化"
	a, pm := newTestAgent(t, `{"story_thread":"新主线：展开后细化",`+
		`"children":[{"id":"ch_1_1","title":"第1章"},{"id":"ch_1_2","title":"第2章"}]}`)
	if err := a.Save(&types.OutlineFile{
		StoryThread: "旧主线",
		Nodes:       []types.OutlineNode{{ID: "vol_1", Title: "第一卷"}},
	}); err != nil {
		t.Fatal(err)
	}

	of, err := a.ExpandNode(context.Background(), "vol_1", 2)
	if err != nil {
		t.Fatalf("ExpandNode: %v", err)
	}
	if of.StoryThread != thread {
		t.Fatalf("展开应回收主线: got %q", of.StoryThread)
	}
	if len(of.Nodes) != 1 || len(of.Nodes[0].Children) != 2 {
		t.Fatalf("children 追加行为被破坏: %+v", of.Nodes)
	}
	if err := a.Save(of); err != nil {
		t.Fatal(err)
	}
	back, err := pm.ReadOutlines()
	if err != nil {
		t.Fatal(err)
	}
	if back.StoryThread != thread {
		t.Fatalf("落盘往返主线丢失: got %q", back.StoryThread)
	}
}

// TestExpandNode_KeepsStoryThreadWhenReplyOmits 展开回复不含 story_thread
// → 已有主线保持不变。
func TestExpandNode_KeepsStoryThreadWhenReplyOmits(t *testing.T) {
	const existing = "既有主线：展开不擦除"
	a, _ := newTestAgent(t, `{"children":[{"id":"ch_2_1","title":"第1章"}]}`)
	if err := a.Save(&types.OutlineFile{
		StoryThread: existing,
		Nodes:       []types.OutlineNode{{ID: "vol_2", Title: "第二卷"}},
	}); err != nil {
		t.Fatal(err)
	}
	of, err := a.ExpandNode(context.Background(), "vol_2", 1)
	if err != nil {
		t.Fatalf("ExpandNode: %v", err)
	}
	if of.StoryThread != existing {
		t.Fatalf("回复缺字段应保留已有主线: got %q", of.StoryThread)
	}
}

// TestMergeStoryThread_NonEmptyOnly 直接锁定合并语义（含 dst=nil 防御）。
func TestMergeStoryThread_NonEmptyOnly(t *testing.T) {
	mergeStoryThread(nil, "主线") // 不得 panic

	cases := []struct {
		name   string
		thread string
		want   string
	}{
		{"non-empty overrides", "新主线", "新主线"},
		{"empty keeps", "", "旧主线"},
		{"blank keeps", "  \n\t ", "旧主线"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			of := &types.OutlineFile{StoryThread: "旧主线"}
			mergeStoryThread(of, c.thread)
			if of.StoryThread != c.want {
				t.Fatalf("merged = %q, want %q", of.StoryThread, c.want)
			}
		})
	}
}
