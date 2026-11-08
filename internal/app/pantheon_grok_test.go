package app

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	gaeaProvider "github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/pantheon"
)

// grokChatModel 解析口径（2026-10-29 实弹：本机 xai 目录把 grok-imagine-video
// 判成 llm，名字级防御必须把生成档挡在议事席外）。
func TestGrokChatModelResolution(t *testing.T) {
	eng := &modelengine.EngineConfig{
		ID:           "xai",
		DefaultModel: "grok-4.7",
		Models: []modelengine.ModelInfo{
			{ID: "grok-imagine-video", Kind: "llm"},
			{ID: "grok-4.3", Kind: "llm"},
			{ID: "grok-tts", Kind: "other"},
		},
	}
	if m, ok := grokChatModel(eng); !ok || m != "grok-4.7" {
		t.Fatalf("default model = %q ok=%v, want grok-4.7", m, ok)
	}

	eng.DefaultModel = "grok-imagine-video"
	if m, ok := grokChatModel(eng); !ok || m != "grok-4.3" {
		t.Fatalf("fallback = %q ok=%v, want grok-4.3 (skip imagine)", m, ok)
	}

	eng.DefaultModel = ""
	eng.Models = []modelengine.ModelInfo{{ID: "grok-tts", Kind: "other"}}
	if _, ok := grokChatModel(eng); ok {
		t.Fatal("no chat model should resolve from a tts-only catalog")
	}
}

type pantheonOfficeProbeTool struct{}

func (pantheonOfficeProbeTool) Name() string        { return "office_probe" }
func (pantheonOfficeProbeTool) Description() string { return "读取办公资料做验证" }
func (pantheonOfficeProbeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}
func (pantheonOfficeProbeTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "办公资料显示应继续验证", nil
}
func (pantheonOfficeProbeTool) ReadOnly() bool { return true }

func TestRunPantheonGrokAgentUsesOfficeToolLoop(t *testing.T) {
	prov := testutil.NewMock("grok-office",
		testutil.Turn{ToolCalls: []gaeaProvider.ToolCall{{
			ID: "call_1", Name: "office_probe", Arguments: `{"scope":"docs"}`,
		}}},
		testutil.Turn{Text: "基于办公工具结果，建议继续验证。"},
	)
	reg := tool.NewRegistry()
	reg.Add(pantheonOfficeProbeTool{})
	var streamed strings.Builder
	var traces []pantheon.ToolTrace

	final, err := runPantheonGrokAgent(
		context.Background(),
		prov,
		reg,
		t.TempDir(),
		"请先核验办公资料，再给出结论。",
		nil,
		func(text string) { streamed.WriteString(text) },
		func(trace pantheon.ToolTrace) { traces = append(traces, trace) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(final, "继续验证") || !strings.Contains(streamed.String(), "继续验证") {
		t.Fatalf("final=%q streamed=%q", final, streamed.String())
	}
	if prov.CallCount() != 2 {
		t.Fatalf("provider calls = %d, want tool round + final", prov.CallCount())
	}
	if len(traces) < 2 || traces[0].Name != "office_probe" ||
		traces[0].Status != "started" || traces[1].Status != "completed" {
		t.Fatalf("tool traces = %+v", traces)
	}
	req := prov.Requests()[0]
	if len(req.Tools) != 1 || req.Tools[0].Name != "office_probe" {
		t.Fatalf("office tool schema was not sent to Grok: %+v", req.Tools)
	}
}

func TestPantheonGrokOfficeRegistryIsReadOnly(t *testing.T) {
	seat := &pantheonGrokSeat{}
	reg := seat.officeToolRegistry(t.TempDir())
	for _, name := range []string{"read_file", "grep", "ls"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("read-only workspace tool %q missing", name)
		}
	}
	for _, name := range []string{"write_file", "edit_file", "bash", "task", "install_skill"} {
		if _, ok := reg.Get(name); ok {
			t.Fatalf("forbidden tool %q exposed to Grok", name)
		}
	}
	prompt := pantheonGrokSystemPrompt(reg.Names())
	if !strings.Contains(prompt, "Gaea 办公 AgentRunner") ||
		!strings.Contains(prompt, "禁止写文件") {
		t.Fatalf("Grok office system prompt is incomplete:\n%s", prompt)
	}
}

// T0 漂移锁（2026-11-08 众神殿结论 pan_1794098356822 第一步）：doc_render 带
// ReadOnly 标签却删/建工作区产物、写系统临时目录、起 LibreOffice 子进程——
// 元数据声明不构成隔离，在 effect 隔离审计完成前不得回到任何议事 T0 名单。
func TestPantheonGrokT0ExcludesDocRender(t *testing.T) {
	for _, name := range pantheonGrokWorkspaceTools {
		if name == "doc_render" {
			t.Fatal("doc_render 不得留在议事 T0（ReadOnly 标签≠无副作用，待 effect 隔离审计）")
		}
	}
	for name := range pantheonGrokOfficeReadTools {
		if name == "doc_render" {
			t.Fatal("doc_render 不得留在议事 T0 办公只读名单")
		}
	}
	seat := &pantheonGrokSeat{}
	if _, ok := seat.officeToolRegistry(t.TempDir()).Get("doc_render"); ok {
		t.Fatal("装配后的注册表不得包含 doc_render")
	}
}

func TestPantheonGrokProbeAdvertisesOfficeTools(t *testing.T) {
	status := (&pantheonGrokSeat{}).ProbeGrok(context.Background())
	joined := strings.Join(status.Capabilities, ",")
	for _, capability := range []string{"gaea-office-tools", "read-only-tools", "tool-trajectory"} {
		if !strings.Contains(joined, capability) {
			t.Fatalf("Grok capabilities %v missing %q", status.Capabilities, capability)
		}
	}
	if status.Kind != "Gaea 办公席位 · xAI" {
		t.Fatalf("Grok kind = %q", status.Kind)
	}
}

// T0 精确集漂移锁（2026-11-08 众神殿结论第二步·最小切片）：议事 T0 两个名单
// 与登记表逐字钉死——新增任何工具必须显式改本测试并过 effect 隔离审计，防止
// 「顺手加一个」式的静默扩面。
func TestPantheonGrokT0ExactSets(t *testing.T) {
	wantWorkspace := map[string]bool{
		"read_file": true, "grep": true, "ls": true,
		"cad_ask": true, "vision": true,
		"schedule_get": true, "schedule_analyze": true,
	}
	gotWorkspace := map[string]bool{}
	for _, name := range pantheonGrokWorkspaceTools {
		if gotWorkspace[name] {
			t.Fatalf("T0 工作区名单有重复项 %q", name)
		}
		gotWorkspace[name] = true
	}
	if len(gotWorkspace) != len(wantWorkspace) {
		t.Fatalf("T0 工作区名单漂移：got %v, want %v（新增须过 effect 审计并改本测试）", keysOf(gotWorkspace), keysOf(wantWorkspace))
	}
	for name := range wantWorkspace {
		if !gotWorkspace[name] {
			t.Fatalf("T0 工作区名单缺 %q", name)
		}
	}

	wantOffice := map[string]bool{
		"cost_search": true, "cost_indicators": true, "cost_compose": true,
		"knowledge_search": true, "memory_search": true, "fact_list": true,
		"web_search": true, "web_fetch": true, "read_skill": true,
		"genui_validate": true, "ocr": true,
	}
	gotOffice := map[string]bool{}
	for name := range pantheonGrokOfficeReadTools {
		gotOffice[name] = true
	}
	if len(gotOffice) != len(wantOffice) {
		t.Fatalf("T0 办公只读名单漂移：got %v, want %v", keysOf(gotOffice), keysOf(wantOffice))
	}
	for name := range wantOffice {
		if !gotOffice[name] {
			t.Fatalf("T0 办公只读名单缺 %q", name)
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
