package app

// office_cost_library_schema_test.go — cost_library 参数说明两层结构的守卫
// （2026-11-25 办公工具梳理批）。
//
// 钉三件事：
//   - quota 花括号回归：旧 schema 的 quota 少一个右花括号，31 个顶层参数在模型
//     眼里嵌进了 quota（铜陵反馈 #13「把 dict/classify 塞进 quota」的根因）；
//   - 契约不丢键：模型最终能看到的每个键，要么在常驻 schema，要么在回执携带的
//     行契约文本里——两层缺一键，模型就有一个字段无处可查；
//   - 常驻面预算：行契约不常驻是为了省每轮上下文，常驻面回胖即红。

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// clSchemaKeys 递归收集一段 JSON Schema 里出现过的全部属性名。
func clSchemaKeys(raw json.RawMessage, into map[string]bool) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return
	}
	var walk func(x any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			if props, ok := t["properties"].(map[string]any); ok {
				for k, pv := range props {
					into[k] = true
					walk(pv)
				}
			}
			if items, ok := t["items"].(any); ok {
				walk(items)
			}
		}
	}
	walk(v)
}

func TestCostLibrarySchemaQuotaBraceRegression(t *testing.T) {
	var resident struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(costLibraryResidentSchema(), &resident); err != nil {
		t.Fatalf("常驻 schema 不是合法 JSON: %v", err)
	}
	var quota map[string]json.RawMessage
	if err := json.Unmarshal(resident.Properties["quota"], &quota); err != nil {
		t.Fatalf("quota 不是对象: %v", err)
	}
	if len(quota) != 3 {
		t.Fatalf("quota 只应有 type/description/properties 三个键（实际 %d 个：%v）——多出来的键是花括号又少了，后续参数被嵌进了 quota", len(quota), clKeysOf(quota))
	}
	// 这批参数曾被嵌进 quota，现在必须是顶层。
	for _, k := range []string{"resource", "rows", "classify", "from_code", "import_id", "project", "controls", "resources", "quotas", "bills"} {
		if _, ok := resident.Properties[k]; !ok {
			t.Errorf("参数 %s 应是常驻 schema 的顶层键", k)
		}
	}
}

func clKeysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestCostLibrarySchemaContractCoversDocumentedKeys 两层加起来不丢键。
func TestCostLibrarySchemaContractCoversDocumentedKeys(t *testing.T) {
	full := map[string]bool{}
	clSchemaKeys(costLibraryDocumentedSchema(), full)
	resident := map[string]bool{}
	clSchemaKeys(costLibraryResidentSchema(), resident)
	contract := clImportRowsContract + "\n" + clClassifyContract
	missing := []string{}
	for k := range full {
		if resident[k] {
			continue
		}
		// 契约里字段名后面跟类型（缩进清单）、或紧跟 { [ （压行小对象、对象数组）。
		if !regexp.MustCompile(`(^|[\s{,])` + regexp.QuoteMeta(k) + `[\s\[{]`).MatchString(contract) {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("这些键既不在常驻 schema 也不在回执契约里，模型无处可查: %v", missing)
	}
	// 行级字段不该回流常驻面（否则白省）。
	residentText := string(costLibraryResidentSchema())
	for _, k := range []string{"price_override", "quantity_expr", "quota_codes", "list_price", "sheet", "derived", "division"} {
		if strings.Contains(residentText, `"`+k+`"`) {
			t.Errorf("行级字段 %s 不该出现在常驻 schema", k)
		}
	}
}

// TestCostLibraryContractRidesReceipts 行契约在流程必经节点随身发放。
func TestCostLibraryContractRidesReceipts(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	tl := costLibraryTool{a: a}

	// import_open 回执带 import_add 行契约。
	out := clMust(t, tl, map[string]any{"action": "import_open", "project": map[string]any{"name": "p"}}, "import_add 行字段")
	t.Logf("行契约 %d 字符：\n%s", utf8.RuneCountInString(clImportRowsContract), clImportRowsContract)
	for _, k := range []string{"resource_code", "quantity_expr", "price_override", "loss_rate"} {
		if !strings.Contains(out, k) {
			t.Errorf("import_open 回执的行契约缺字段 %s", k)
		}
	}
	// dims 回执带 classify 契约。
	out = clMust(t, tl, map[string]any{"action": "dims"}, "classify 字段")
	for _, k := range []string{"price_period", "division", "chapter", "category_path"} {
		if !strings.Contains(out, k) {
			t.Errorf("dims 回执的 classify 契约缺字段 %s", k)
		}
	}
	// classify 缺参：错误里直接给契约，不用再跑一趟 help。
	clMustFail(t, tl, map[string]any{"action": "classify"}, "category_path")
	// help 不碰库也拿得到（零 App 实例）。
	out, err := (costLibraryTool{}).Execute(nil, json.RawMessage(`{"action":"help"}`))
	if err != nil {
		t.Fatalf("help 应成功: %v", err)
	}
	for _, k := range []string{"resource_code", "quantity_expr", "category_path", `"action":"dict_add"`} {
		if !strings.Contains(out, k) {
			t.Errorf("help 回执缺 %s", k)
		}
	}
	// import_add 失败时 UsageHint 给契约，其余动作给骨架。
	hint := (costLibraryTool{}).UsageHint(json.RawMessage(`{"action":"import_add"}`))
	if !strings.Contains(hint, "quantity_expr") {
		t.Errorf("import_add 的 UsageHint 应带行契约: %s", hint)
	}
	if hint := (costLibraryTool{}).UsageHint(json.RawMessage(`{"action":"delete_quota"}`)); !strings.Contains(hint, `"codes"`) {
		t.Errorf("delete_quota 的 UsageHint 应带骨架: %s", hint)
	}
}

// clResidentBudget 常驻 schema 的字符预算（rune，源文本口径含缩进）。2026-11-25
// 梳理前 13091；行契约与 classify 契约改随回执发放、嵌套 dict 对象退场后实测
// 8541，预算留一成余量。超了=有人把行级字段又写回了常驻面。
const clResidentBudget = 9400

func TestCostLibraryResidentSchemaBudget(t *testing.T) {
	n := utf8.RuneCountInString(string(costLibraryResidentSchema()))
	if n > clResidentBudget {
		t.Fatalf("cost_library 常驻 schema %d 字符，超预算 %d（行级字段请走回执契约）", n, clResidentBudget)
	}
	t.Logf("cost_library 常驻 schema %d 字符（完整说明 %d）", n, utf8.RuneCountInString(string(costLibraryDocumentedSchema())))
}
