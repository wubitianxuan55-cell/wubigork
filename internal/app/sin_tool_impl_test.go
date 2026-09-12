package app

// 原罪工具集 v1 实现测试（一）：注册表接线 / 联网委托 / 角色卡渲染 / 线格式契约。
// 便签与大纲见 sin_notes_test.go，工具循环见 sin_tool_loop_test.go。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// sinTestHome 把用户配置目录重定向到临时 HOME（sinRoot 随之落临时目录，
// 便签/大纲测试不碰真机数据）。
func sinTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	return home
}

// TestSinToolRegistryWiring v1 五个工具全部在位、顺序稳定、schema 合法、
// 只读标记如实（写类工具不得声明只读——前端徽标与提示词据此）。
func TestSinToolRegistryWiring(t *testing.T) {
	a := &App{}
	tools := a.sinToolSet("sin_1_1")
	want := []string{sinToolWebSearch, sinToolWebFetch, sinToolCast, sinToolNotes, sinToolOutline, sinToolExport}
	if got := sinToolNames(tools); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("工具顺序 = %v, want %v", got, want)
	}
	wantReadOnly := map[string]bool{
		sinToolWebSearch: true,
		sinToolWebFetch:  true,
		sinToolCast:      true,
		sinToolNotes:     false, // 含 write/set/delete
		sinToolOutline:   false, // 含 write
		sinToolExport:    false, // 新建导出文件（不改正文）
	}
	for _, tl := range tools {
		if strings.TrimSpace(tl.Description()) == "" {
			t.Errorf("%s 缺少说明词（模型靠它决定用不用）", tl.Name())
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(tl.Schema(), &parsed); err != nil {
			t.Errorf("%s schema 不是合法 JSON: %v", tl.Name(), err)
		}
		if parsed["type"] != "object" {
			t.Errorf("%s schema 顶层 type = %v, want object", tl.Name(), parsed["type"])
		}
		if got := tl.ReadOnly(); got != wantReadOnly[tl.Name()] {
			t.Errorf("%s ReadOnly = %v, want %v", tl.Name(), got, wantReadOnly[tl.Name()])
		}
	}

	// tools 定义必须是 OpenAI 兼容的 {"type":"function",...}——缺 type 会被
	// Grok/DeepSeek 等以 400 拒绝（同 ai 包既有教训）。
	schemas := sinToolSchemas(tools)
	if len(schemas) != len(want) {
		t.Fatalf("schemas 数量 = %d, want %d", len(schemas), len(want))
	}
	for _, s := range schemas {
		if s.Type != "function" || s.Function.Name == "" || len(s.Function.Parameters) == 0 {
			t.Errorf("工具定义不完整: %+v", s)
		}
	}
	if sinToolSchemas(nil) != nil {
		t.Error("空工具集应返回 nil（请求体不带 tools 字段）")
	}
}

// TestSinWebToolsDelegateToBuiltin 联网工具委托办公 builtin：schema 与 builtin
// 逐字节同源（不维护第二份），且 builtin 确实注册（空白导入有效）。
func TestSinWebToolsDelegateToBuiltin(t *testing.T) {
	for _, kind := range []string{sinToolWebSearch, sinToolWebFetch} {
		b, ok := tool.LookupBuiltin(kind)
		if !ok || b == nil {
			t.Fatalf("内置 %s 未注册（原罪无法委托）", kind)
		}
		del := sinToolWebDelegate{kind: kind}
		if string(del.Schema()) != string(b.Schema()) {
			t.Errorf("%s schema 应与 builtin 同源", kind)
		}
		if !del.ReadOnly() {
			t.Errorf("%s 应声明只读", kind)
		}
		// 说明词是原罪自己的（故事语境），不是 builtin 的原文
		if strings.TrimSpace(del.Description()) == "" || del.Description() == b.Description() {
			t.Errorf("%s 应有原罪语境的说明词", kind)
		}
		if del.Name() != kind {
			t.Errorf("名字 = %q, want %q", del.Name(), kind)
		}
	}
}

// TestSinCastToolText 角色卡渲染矩阵：未选角色 / 列示 / 精确 / 模糊 / 未命中 /
// 空字段不占位（不编造）。
func TestSinCastToolText(t *testing.T) {
	cast := []*characterlib.Character{
		{
			Name: "林晚", Gender: "女", Age: "28", RoleType: "记者",
			Appearance: "短发，左眉有一道旧疤", Figure: "偏瘦",
			Personality: "克制，先看再说", Background: "地方台调查记者",
			Motivation: "查清三年前的旧案", Status: "Alive",
			VoiceGuide:      "短句，少形容词",
			DialogueSamples: []string{"先说事实。", "我不写我没见过的。", "你在藏什么？", "第四条不该出现"},
		},
		{Name: "顾城"},
		nil, // 脏数据：整条跳过
	}

	if got := sinCastToolText(nil, ""); !strings.Contains(got, "还没有选择角色卡") {
		t.Errorf("未选角色应如实说明: %q", got)
	}

	list := sinCastToolText(cast, "")
	if !strings.Contains(list, "林晚") || !strings.Contains(list, "顾城") {
		t.Errorf("列示应含全部角色: %q", list)
	}
	if strings.Contains(list, "说话样例") {
		t.Errorf("列示不该带说话样例（那是详情面）: %q", list)
	}

	full := sinCastToolText(cast, "林晚")
	for _, want := range []string{"角色卡：林晚", "外观", "旧疤", "背景", "动机", "口吻", "说话样例"} {
		if !strings.Contains(full, want) {
			t.Errorf("详情缺 %q: %q", want, full)
		}
	}
	if strings.Contains(full, "第四条不该出现") {
		t.Errorf("说话样例应封顶 3 条: %q", full)
	}
	// 空字段不占位：顾城只有名字，详情里不应出现空标签。
	bare := sinCastToolText(cast, "顾城")
	if strings.Contains(bare, "外观") || strings.Contains(bare, "性别") {
		t.Errorf("空字段不该占位: %q", bare)
	}

	if fuzzy := sinCastToolText(cast, " 林 "); !strings.Contains(fuzzy, "角色卡：林晚") {
		t.Errorf("去空白后的包含匹配应命中: %q", fuzzy)
	}
	miss := sinCastToolText(cast, "白墨")
	if !strings.Contains(miss, "没有叫「白墨」") || !strings.Contains(miss, "林晚") {
		t.Errorf("未命中应回列可查名字: %q", miss)
	}
}

// TestSinToolFramesContract 线格式契约：前端过程卡按这些键名取值，一旦漂移
// 卡片会静默变空，所以逐键锁定（与 sinToolTrace 的 JSON 标签同一套名字）。
func TestSinToolFramesContract(t *testing.T) {
	tr := sinToolTrace{
		ID: "call_1", Name: sinToolWebSearch, Args: `{"query":"唐末长安坊市"}`,
		Output: "结果", ElapsedMS: 1234, ReadOnly: true,
	}
	dispatch := sinToolDispatchFrame(tr)
	assertSinFrameKeys(t, dispatch, []string{"type", "id", "name", "args", "read_only"})
	if dispatch["type"] != "tool_dispatch" {
		t.Errorf("dispatch.type = %v", dispatch["type"])
	}
	result := sinToolResultFrame(tr)
	assertSinFrameKeys(t, result, []string{"type", "id", "name", "output", "error", "elapsed_ms"})
	if result["type"] != "tool_result" {
		t.Errorf("result.type = %v", result["type"])
	}

	// 轨迹落库形态：与 done.tools / extra.tools 同一个结构体，键名以 JSON 标签为准。
	raw, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var keys map[string]interface{}
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, k := range []string{"id", "name", "args", "output", "elapsed_ms", "read_only"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("轨迹 JSON 缺键 %q: %s", k, raw)
		}
	}
}

// TestSinClampToolOutput 超长结果截断并留可见标记（不静默切）。
func TestSinClampToolOutput(t *testing.T) {
	short := "短结果"
	if got := sinClampToolOutput(short); got != short {
		t.Errorf("短结果不应改写: %q", got)
	}
	long := strings.Repeat("字", sinToolOutputMaxRunes+100)
	got := sinClampToolOutput(long)
	if !strings.Contains(got, "已截断") {
		t.Error("超长结果应留截断标记")
	}
	if len([]rune(got)) > sinToolOutputMaxRunes+20 {
		t.Errorf("截断后仍过长: %d rune", len([]rune(got)))
	}
}

// assertSinFrameKeys 键集合必须恰好相等（多一个少一个都是契约漂移）。
func assertSinFrameKeys(t *testing.T, frame map[string]interface{}, want []string) {
	t.Helper()
	if len(frame) != len(want) {
		t.Fatalf("frame 键数 = %d, want %d（%v）", len(frame), len(want), frame)
	}
	for _, k := range want {
		if _, ok := frame[k]; !ok {
			t.Errorf("frame 缺键 %q: %v", k, frame)
		}
	}
}
