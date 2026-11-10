package agent

// B2 失败即值——executeOne 收口测试（dsh-ops 蒸馏 2026-11-10）：
//   - 错误通道：结构化 Failure 按字段渲染（conflict 码不退化为 exec_error）；
//     未结构化错误按副作用账本默认分类（写族=unknown/只读=none）；
//   - 成功通道：内层失败信封塌缩（不再「成功包着失败」），成功信封不塌缩。
// 盘查实证：executeOne 信封形状此前零测试，本文件一并钉死。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// envelopeStub 可注入任意 result/err 的测试工具（fakeTool 只能回固定文案）。
type envelopeStub struct {
	name     string
	result   string
	err      error
	readOnly bool
}

func (s envelopeStub) Name() string            { return s.name }
func (s envelopeStub) Description() string     { return "" }
func (s envelopeStub) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (s envelopeStub) ReadOnly() bool          { return s.readOnly }
func (s envelopeStub) PersistWrite() bool      { return false }
func (s envelopeStub) Execute(context.Context, json.RawMessage) (string, error) {
	return s.result, s.err
}

func newEnvelopeRunner(reg *tool.Registry) *AgentRunner {
	return New(nil, reg, NewSession(""), Options{}, event.Discard)
}

func TestExecuteOneErrorEnvelopeFailureAsValue(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(envelopeStub{name: "edit_file", readOnly: false, err: fmt.Errorf("write blew up")})
	reg.Add(envelopeStub{name: "grep", readOnly: true, err: fmt.Errorf("bad pattern")})
	reg.Add(envelopeStub{name: "stub_cas", readOnly: false, err: &tool.Failure{
		Code:        tool.CodeConflict,
		Err:         fmt.Errorf("REVISION_CONFLICT: 已被外部修改"),
		Retryable:   true,
		EffectState: tool.EffectNone,
		NextAction:  "重读拿最新 revision 后再改",
	}})
	a := newEnvelopeRunner(reg)

	// 写族（IsDeliverableTool 账本名）任意失败 → effect=unknown + 先核实指引。
	env, ok := tool.ParseEnvelope(a.executeOne(context.Background(), provider.ToolCall{Name: "edit_file"}).output)
	if !ok {
		t.Fatal("edit_file 失败应产出信封")
	}
	if env.OK || env.Code != tool.CodeExecError || !env.Retryable {
		t.Fatalf("写族默认分类错: %+v", env)
	}
	if env.EffectState != tool.EffectUnknown || !strings.Contains(env.NextAction, "核实") {
		t.Fatalf("写族 effect 应 unknown+先核实: effect=%q next=%q", env.EffectState, env.NextAction)
	}

	// 只读族 → effect=none。
	env2, _ := tool.ParseEnvelope(a.executeOne(context.Background(), provider.ToolCall{Name: "grep"}).output)
	if env2.EffectState != tool.EffectNone {
		t.Fatalf("只读族 effect 应 none: %q", env2.EffectState)
	}

	// 结构化 Failure 不退化：conflict 码原样进信封。
	env3, _ := tool.ParseEnvelope(a.executeOne(context.Background(), provider.ToolCall{Name: "stub_cas"}).output)
	if env3.Code != tool.CodeConflict || env3.EffectState != tool.EffectNone || !env3.Retryable {
		t.Fatalf("Failure 应按字段渲染: %+v", env3)
	}
	if !strings.Contains(env3.Error, "REVISION_CONFLICT") {
		t.Fatalf("前缀兼容层缺失: %q", env3.Error)
	}
}

func TestExecuteOneInnerFailureEnvelopeCollapse(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(envelopeStub{name: "stub_json_bash", readOnly: false,
		result: tool.WrapError(tool.CodeValidationError, "param missing", nil)})
	reg.Add(envelopeStub{name: "stub_ok", readOnly: true,
		result: tool.WrapResult(tool.CodeOK, map[string]any{"value": 1})})
	a := newEnvelopeRunner(reg)

	// 内层失败信封塌缩：外层不再 ok:true 包壳，data 带 tool，errMsg 置位。
	o := a.executeOne(context.Background(), provider.ToolCall{Name: "stub_json_bash"})
	env, ok := tool.ParseEnvelope(o.output)
	if !ok {
		t.Fatalf("应产出信封: %q", o.output)
	}
	if env.OK || env.Code != tool.CodeValidationError {
		t.Fatalf("内层失败应塌缩为失败信封: %+v", env)
	}
	if m, isMap := env.Data.(map[string]any); !isMap || m["tool"] != "stub_json_bash" {
		t.Fatalf("塌缩信封 data.tool 缺失: %+v", env.Data)
	}
	if o.errMsg == "" {
		t.Fatal("塌缩后 errMsg 应置位（失败计数/UI 据此判失败）")
	}

	// 成功信封不塌缩（o.errMsg 空，正常走压缩+外壳路径）。
	o2 := a.executeOne(context.Background(), provider.ToolCall{Name: "stub_ok"})
	if o2.errMsg != "" {
		t.Fatalf("成功调用不应有 errMsg: %+v", o2)
	}
	env2, _ := tool.ParseEnvelope(o2.output)
	if !env2.OK {
		t.Fatalf("成功信封应保持成功: %q", o2.output)
	}
}

func TestExecuteOnePlainJSONResultNotCollapsed(t *testing.T) {
	// 守卫：非信封 JSON（无 ok/code 字段）不得误塌缩——整串解析+code 非空
	// 双条件缺一不可。
	reg := tool.NewRegistry()
	reg.Add(envelopeStub{name: "stub_data", readOnly: true,
		result: `{"rows": 3, "ok2": false}`})
	a := newEnvelopeRunner(reg)
	o := a.executeOne(context.Background(), provider.ToolCall{Name: "stub_data"})
	if o.errMsg != "" {
		t.Fatalf("普通 JSON 数据不应误判失败: %+v", o)
	}
	env, _ := tool.ParseEnvelope(o.output)
	if !env.OK {
		t.Fatalf("普通 JSON 应保持成功外壳: %q", o.output)
	}
}
