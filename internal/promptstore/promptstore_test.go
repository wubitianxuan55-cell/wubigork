// promptstore 表驱动测试（规格 进度计划/gaea-prompt-workshop-t6-20260916.md §3.4
// 清单逐项落地）：Validate 六规则矩阵（error/warn 分级、长度边界、花括号不平衡
// 含未闭合）/ Upsert 自增·CreatedAt 保留·字典序 / Remove / ActiveOverride 含
// IsActive=false 回 nil / NormalizeKey 全防御 / JSON camelCase 标签钉死。
// 先例 taskinbox/taskinbox_test.go：白盒同包、零 IO、纯表驱动。
package promptstore

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/prompt"
)

// cleanT 干净模板夹具：非空 system/task、声明了 word_count 且正文真用它。
func cleanT() prompt.Template {
	return prompt.Template{
		Name:       "create-chapter",
		System:     "你是正在写这本书的作者。",
		Task:       "写出本章正文，不少于{{word_count}}字。",
		Output:     prompt.OutputDef{Format: "markdown", Description: "正文，不少于{{word_count}}字"},
		Parameters: []string{"word_count"},
	}
}

// issueKeys 把 Issue 列表折叠为 "code|severity" 序列（顺序敏感——Validate 输出
// 按固定规则序，测试同时钉住这个顺序）。
func issueKeys(issues []Issue) []string {
	keys := make([]string, len(issues))
	for i, iss := range issues {
		keys[i] = iss.Code + "|" + iss.Severity
	}
	return keys
}

// TestValidateRuleMatrix 六规则主矩阵：每用例断言完整 (code|severity) 序列。
func TestValidateRuleMatrix(t *testing.T) {
	cases := []struct {
		name string
		tmpl func() prompt.Template
		want []string
	}{
		{"干净模板（已声明占位符）", cleanT, nil},
		{"system 为空", func() prompt.Template {
			t := cleanT()
			t.System = ""
			return t
		}, []string{CodeEmptySystem + "|" + SeverityError}},
		{"system 纯空白（trim 后为空）", func() prompt.Template {
			t := cleanT()
			t.System = " \n\t　"
			return t
		}, []string{CodeEmptySystem + "|" + SeverityError}},
		{"task 为空", func() prompt.Template {
			t := cleanT()
			t.Task = "   "
			return t
		}, []string{CodeEmptyTask + "|" + SeverityError}},
		{"system 与 task 双空（两条 error 并存）", func() prompt.Template {
			t := cleanT()
			t.System, t.Task = "", ""
			return t
		}, []string{
			CodeEmptySystem + "|" + SeverityError,
			CodeEmptyTask + "|" + SeverityError,
		}},
		{"未闭合 {{（其后无 }}）", func() prompt.Template {
			t := cleanT()
			t.Task = "写出本章，不少于{{word_count 字。"
			return t
		}, []string{CodeBraceUnbalanced + "|" + SeverityError}},
		{"{{ 与 }} 计数不等（多一个 {{）——合法 {{a}} 部分仍报未声明", func() prompt.Template {
			t := cleanT()
			t.Task = "{{a}}{{"
			return t
		}, []string{CodeBraceUnbalanced + "|" + SeverityError, CodeUndeclaredVar + "|" + SeverityWarn}},
		{"计数相等但存在未闭合 {{（}} 在前）", func() prompt.Template {
			t := cleanT()
			t.Task = "}} 尾部才有 {{"
			return t
		}, []string{CodeBraceUnbalanced + "|" + SeverityError}},
		{"output.description 段不平衡也报（全正文口径）", func() prompt.Template {
			t := cleanT()
			t.Output.Description = "输出示例 {{unclosed"
			return t
		}, []string{CodeBraceUnbalanced + "|" + SeverityError}},
		{"单层 JSON 字面量花括号不误报", func() prompt.Template {
			t := cleanT()
			t.Output.Description = `输出 {"nodes":[{"id":"ch_1"}]} 数组`
			return t
		}, nil},
		{"中文单层花括号不是旧语法（字面格式示例）", func() prompt.Template {
			t := cleanT()
			t.Task = "卷标题格式「第N卷·{阶段名}·{卷名}」"
			t.Parameters = nil
			return t
		}, nil},
		{"undeclared-var：占位符未声明（warn）", func() prompt.Template {
			t := cleanT()
			t.Parameters = []string{"other"}
			return t
		}, []string{CodeUndeclaredVar + "|" + SeverityWarn}},
		{"undeclared-var 同名多段只报一次", func() prompt.Template {
			t := cleanT()
			t.Task = "写出 {{word_count}} 字"
			t.Output.Description = "不少于 {{word_count}} 字"
			t.Parameters = []string{"other"}
			return t
		}, []string{CodeUndeclaredVar + "|" + SeverityWarn}},
		{"undeclared-var：Parameters 为 nil 不报（空声明=未维护）", func() prompt.Template {
			t := cleanT()
			t.Parameters = nil
			return t
		}, nil},
		{"legacy-brace：单层 {word_count} 旧语法（warn）", func() prompt.Template {
			t := cleanT()
			t.Task = "写出本章正文，不少于{word_count}字。"
			t.Output.Description = "正文"
			t.Parameters = nil
			return t
		}, []string{CodeLegacyBrace + "|" + SeverityWarn}},
		{"双层 {{word_count}} 不触发 legacy-brace", func() prompt.Template {
			t := cleanT()
			t.Parameters = nil
			return t
		}, nil},
		{"error 与 warn 并存（空 system + 未声明 + 旧语法）", func() prompt.Template {
			t := cleanT()
			t.System = ""
			t.Task = "写出本章，不少于{word_count}字，含{{tone}}。"
			t.Parameters = []string{"word_count"}
			return t
		}, []string{
			CodeEmptySystem + "|" + SeverityError,
			CodeUndeclaredVar + "|" + SeverityWarn,
			CodeLegacyBrace + "|" + SeverityWarn,
		}},
		{"constraints/style 各段同样受检（标签定位）", func() prompt.Template {
			t := cleanT()
			t.Task = "重写整章。"
			t.Output.Description = "正文"
			t.Parameters = []string{"word_count"} // 非空声明：{{style_ref}} 才会被比对出未声明
			t.Constraints.Must = []string{"保持 {{style_ref}} 一致"}
			t.Constraints.Style = []string{"干净克制", "节奏参考 {pace} 口径"}
			return t
		}, []string{
			CodeUndeclaredVar + "|" + SeverityWarn,
			CodeLegacyBrace + "|" + SeverityWarn,
		}},
	}
	for _, c := range cases {
		got := issueKeys(Validate(c.tmpl()))
		if c.want == nil {
			c.want = []string{}
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s: 校验结果 got %v want %v", c.name, got, c.want)
		}
	}
}

// TestValidateSeverityMatrix error/warn 分级钉死：六规则全走查（防未来错配
// 档位——error 阻断落盘、warn 不阻断）+ wire 值逐字校验。
func TestValidateSeverityMatrix(t *testing.T) {
	severityByCode := map[string]string{
		CodeEmptySystem:     SeverityError,
		CodeEmptyTask:       SeverityError,
		CodeBraceUnbalanced: SeverityError,
		CodeTooLong:         SeverityError,
		CodeUndeclaredVar:   SeverityWarn,
		CodeLegacyBrace:     SeverityWarn,
	}
	// 四份夹具合起来触发全部六规则（empty-system/brace 由第一份、empty-task
	// 由第二份、too-long 由第三份、两 warn 由第四份触发）。
	all := append(
		append(
			append(
				Validate(prompt.Template{Name: "x", Task: "{{a}}{{"}),
				Validate(prompt.Template{Name: "x", System: "s"})..., // task 空
			),
			Validate(prompt.Template{Name: "x", System: "s", Task: strings.Repeat("字", MaxFieldRunes+1)})...,
		),
		Validate(prompt.Template{Name: "x", System: "s", Task: "用 {word_count} 字", Parameters: []string{"other2"}, Output: prompt.OutputDef{Description: "见 {{other3}}"}})...,
	)
	hit := map[string]bool{}
	for _, iss := range all {
		want, ok := severityByCode[iss.Code]
		if !ok {
			t.Fatalf("未知规则 code %q（矩阵外新规则需补分级表）", iss.Code)
		}
		if iss.Severity != want {
			t.Fatalf("规则 %q 分级 got %q want %q", iss.Code, iss.Severity, want)
		}
		hit[iss.Code] = true
	}
	for code := range severityByCode {
		if !hit[code] {
			t.Fatalf("规则 %q 未被夹具触发（分级表走查不全）", code)
		}
	}
	if SeverityError != "error" || SeverityWarn != "warn" {
		t.Fatal("severity wire 值必须为 error|warn")
	}
}

// TestValidateTooLongBoundaries 长度边界（rune 口径）：
// System/Task 恰 20000 过、20001 报；全模板序列化恰 65536 过、65537 报。
func TestValidateTooLongBoundaries(t *testing.T) {
	// System 恰界通过。
	t0 := cleanT()
	t0.System = strings.Repeat("字", MaxFieldRunes)
	if issues := Validate(t0); len(issues) != 0 {
		t.Fatalf("System 恰 %d rune 应通过，得到 %v", MaxFieldRunes, issueKeys(issues))
	}
	// System 越界 1 rune 报 too-long。
	t1 := cleanT()
	t1.System = strings.Repeat("字", MaxFieldRunes+1)
	if got := issueKeys(Validate(t1)); !reflect.DeepEqual(got, []string{CodeTooLong + "|" + SeverityError}) {
		t.Fatalf("System %d rune 应报 too-long，得到 %v", MaxFieldRunes+1, got)
	}
	// Task 越界（System 合法）。
	t2 := cleanT()
	t2.Task = strings.Repeat("字", MaxFieldRunes+1)
	if got := issueKeys(Validate(t2)); !reflect.DeepEqual(got, []string{CodeTooLong + "|" + SeverityError}) {
		t.Fatalf("Task 越界应报 too-long，得到 %v", got)
	}
	// 全模板序列化：用 Output.Description（无单字段上限）灌到恰界/越界。
	mk := func(extra int) prompt.Template {
		t := cleanT()
		t.Output.Description = strings.Repeat("字", 60000+extra)
		return t
	}
	b, err := json.Marshal(mk(0))
	if err != nil {
		t.Fatalf("marshal 夹具失败: %v", err)
	}
	have := utf8.RuneCount(b)
	if have >= MaxTemplateRunes {
		t.Fatalf("夹具基线已越界（%d rune），测试夹具失控", have)
	}
	exact := mk(MaxTemplateRunes - have) // 补齐到恰 65536（每个中文字 rune 恰贡献 1 rune 输出）
	b, err = json.Marshal(exact)
	if err != nil {
		t.Fatalf("marshal 夹具失败: %v", err)
	}
	if got := utf8.RuneCount(b); got != MaxTemplateRunes {
		t.Fatalf("夹具应恰 %d rune，实际 %d（夹具假设失效需修正）", MaxTemplateRunes, got)
	}
	if issues := Validate(exact); len(issues) != 0 {
		t.Fatalf("全模板恰 %d rune 应通过，得到 %v", MaxTemplateRunes, issueKeys(issues))
	}
	over := mk(MaxTemplateRunes - have + 1)
	if got := issueKeys(Validate(over)); !reflect.DeepEqual(got, []string{CodeTooLong + "|" + SeverityError}) {
		t.Fatalf("全模板 %d rune 应报 too-long，得到 %v", MaxTemplateRunes+1, got)
	}
}

// TestValidateMessages 消息可定位性抽查：段名/变量名/旧样式进消息（工坊面板
// 逐条渲染 Issues，消息要能指到段）。
func TestValidateMessages(t *testing.T) {
	tmpl := prompt.Template{
		Name:   "x",
		System: "s",
		Task:   "写出 {{word_count}} 字",
		Output: prompt.OutputDef{Format: "markdown", Description: "正文，不少于{word_count}字"},
		Constraints: prompt.ConstraintDef{
			Must:  []string{"保持 {{style_ref}} 一致"},
			Style: []string{"参考 {pace} 口径"},
		},
		Parameters: []string{"other"},
	}
	issues := Validate(tmpl)
	parts := make([]string, len(issues))
	for i, iss := range issues {
		parts[i] = iss.Code + ": " + iss.Message
	}
	joined := strings.Join(parts, "\n")
	for _, want := range []string{
		"undeclared-var", "{{word_count}}", "task 段", // 未声明变量带名字与段名
		"undeclared-var", "{{style_ref}}", "constraints.must[0]", // constraints 段定位
		"legacy-brace", "{word_count}", "output.description 段", // 旧样式带样子与段名
		"legacy-brace", "{pace}", "constraints.style[0]",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("校验消息应包含 %q：\n%s", want, joined)
		}
	}
}

// TestUpsert 同键替换自增 / 新键 Version=1 / CreatedAt 保留 / 结果按 Key 字典序 /
// 入参切片不被改写 / ov 自带的版本与时间戳被重新裁定。
func TestUpsert(t *testing.T) {
	// 新键：Version=1、CreatedAt=UpdatedAt=now（无视调用方乱填的版本与时间戳）。
	entries := Upsert(nil, Override{
		Key:      "create-chapter",
		Content:  prompt.Template{Name: "create-chapter", System: "s", Task: "t"},
		IsActive: true,
		Version:  999, CreatedAt: 1, UpdatedAt: 1, // 必须被重新裁定
	}, 1000)
	if len(entries) != 1 {
		t.Fatalf("新键应 1 条，得到 %d", len(entries))
	}
	if entries[0].Version != 1 || entries[0].CreatedAt != 1000 || entries[0].UpdatedAt != 1000 {
		t.Fatalf("新键应 Version=1/CreatedAt=UpdatedAt=now，得到 %+v", entries[0])
	}
	// 同键再存：Version=旧+1、CreatedAt 保留、UpdatedAt=now2、内容取新值。
	entries = Upsert(entries, Override{
		Key:         "create-chapter",
		Category:    "chapter",
		Description: "改后",
		Content:     prompt.Template{Name: "create-chapter", System: "s2", Task: "t2"},
		IsActive:    false,
		Version:     777, CreatedAt: 1, UpdatedAt: 1, // 同上，被重新裁定
	}, 2000)
	if len(entries) != 1 {
		t.Fatalf("同键应替换为 1 条，得到 %d", len(entries))
	}
	got := entries[0]
	if got.Version != 2 || got.CreatedAt != 1000 || got.UpdatedAt != 2000 {
		t.Fatalf("替换应 Version=旧+1/CreatedAt 保留/UpdatedAt=now，得到 %+v", got)
	}
	if got.Category != "chapter" || got.Description != "改后" || got.Content.System != "s2" || got.IsActive {
		t.Fatalf("替换应取新行内容与启停位，得到 %+v", got)
	}
}

// TestUpsertSortAndPurity 字典序稳定 + 纯函数纪律（入参切片不被改写）。
func TestUpsertSortAndPurity(t *testing.T) {
	mk := func(key string) Override {
		return Override{Key: key, Content: prompt.Template{Name: key, System: "s", Task: "t"}}
	}
	// 乱序插入 z→a→m：结果恒按 Key 字典序。
	entries := Upsert(nil, mk("z-chapter"), 1)
	entries = Upsert(entries, mk("a-chapter"), 2)
	entries = Upsert(entries, mk("m-chapter"), 3)
	wantKeys := []string{"a-chapter", "m-chapter", "z-chapter"}
	if len(entries) != len(wantKeys) {
		t.Fatalf("应 %d 条，得到 %d", len(wantKeys), len(entries))
	}
	for i, k := range wantKeys {
		if entries[i].Key != k {
			t.Fatalf("Upsert 结果应按 Key 字典序：第 %d 位 %q ≠ %q（全序 %v）", i, entries[i].Key, k, keysOf(entries))
		}
	}
	// 替换后仍保字典序。
	entries = Upsert(entries, mk("a-chapter"), 4)
	if keysOf(entries)[0] != "a-chapter" || len(entries) != 3 {
		t.Fatalf("替换不应增删条目且保序：%v", keysOf(entries))
	}
	// 纯函数：入参切片保持原序原值，不被就地改写。
	snapshot := []Override{mk("b"), mk("a")}
	before := append([]Override{}, snapshot...)
	_ = Upsert(snapshot, mk("c"), 5)
	if !reflect.DeepEqual(snapshot, before) {
		t.Fatalf("Upsert 不得改写入参：%v vs %v", snapshot, before)
	}
}

// keysOf 测试辅助：取键序列，便于报错信息可读。
func keysOf(entries []Override) []string {
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}
	return keys
}

// TestRemove 删除命中 / 未命中如实报告 / 重复键残渣一并清 / 入参不被改写。
func TestRemove(t *testing.T) {
	entries := []Override{
		{Key: "a-chapter", IsActive: true, Version: 1},
		{Key: "m-chapter", IsActive: true, Version: 1},
		{Key: "z-chapter", IsActive: false, Version: 2},
	}
	got, ok := Remove(entries, "m-chapter")
	if !ok || !reflect.DeepEqual(keysOf(got), []string{"a-chapter", "z-chapter"}) {
		t.Fatalf("删除 m-chapter 应剩 [a z]：%v ok=%v", keysOf(got), ok)
	}
	// 未命中：内容不变、false。
	same, ok := Remove(got, "no-such")
	if ok || !reflect.DeepEqual(keysOf(same), keysOf(got)) {
		t.Fatalf("未命中应原样返回 false：%v ok=%v", keysOf(same), ok)
	}
	// 空表删除：false 且恒非 nil 空表。
	empty, ok := Remove(nil, "a-chapter")
	if ok || empty == nil || len(empty) != 0 {
		t.Fatalf("空表删除应 (空表,false)，得到 (%v,%v)", empty, ok)
	}
	// 残渣重复键：一并清掉。
	dup := []Override{{Key: "d"}, {Key: "e"}, {Key: "d"}}
	got, ok = Remove(dup, "d")
	if !ok || !reflect.DeepEqual(keysOf(got), []string{"e"}) {
		t.Fatalf("重复键应一并清掉：%v ok=%v", keysOf(got), ok)
	}
	// 纯函数：入参不被改写。
	if !reflect.DeepEqual(keysOf(dup), []string{"d", "e", "d"}) {
		t.Fatalf("Remove 不得改写入参：%v", keysOf(dup))
	}
}

// TestActiveOverride 生效覆盖查询：IsActive 且 Key 匹配才命中；IsActive=false /
// 键不匹配 / 空表一律 nil；返回的是副本（改它不污染表）。
func TestActiveOverride(t *testing.T) {
	entries := []Override{
		{Key: "create-chapter", IsActive: false, Version: 3}, // 停用行：不得命中
		{Key: "rewrite-chapter", IsActive: true, Version: 2},
	}
	if got := ActiveOverride(entries, "create-chapter"); got != nil {
		t.Fatalf("IsActive=false 应回 nil（引擎回落内置），得到 %+v", got)
	}
	got := ActiveOverride(entries, "rewrite-chapter")
	if got == nil || got.Key != "rewrite-chapter" || got.Version != 2 || !got.IsActive {
		t.Fatalf("激活行应命中，得到 %+v", got)
	}
	// 键不匹配 / 空表 → nil。
	if ActiveOverride(entries, "worldview-agent") != nil {
		t.Fatal("键不匹配应回 nil")
	}
	if ActiveOverride(nil, "rewrite-chapter") != nil {
		t.Fatal("空表应回 nil")
	}
	// 副本语义：改返回值不污染入参表。
	got.Version = 99
	if entries[1].Version != 2 {
		t.Fatalf("返回值应为副本（改写不得透传入参表）：entries=%+v", entries[1])
	}
	// 同键残渣：跳过停用行、命中后面的激活行。
	mixed := []Override{
		{Key: "k", IsActive: false},
		{Key: "k", IsActive: true, Version: 7},
	}
	if got := ActiveOverride(mixed, "k"); got == nil || got.Version != 7 {
		t.Fatalf("应跳过停用行命中激活行，得到 %+v", got)
	}
}

// TestNormalizeKey 全防御矩阵：trim / 非空 / 恰 100 rune 过 101 拒 / 字符集
// 白名单（路径分隔符、空白、中文、重音字符一律拒）；哨兵错误 errors.Is 可判。
func TestNormalizeKey(t *testing.T) {
	okCases := []struct{ in, want string }{
		{"create-chapter", "create-chapter"},       // 常规模板名
		{"worldview-agent", "worldview-agent"},     // 带形容词后缀
		{"a", "a"},                                 // 单字符
		{"A.b_C-9", "A.b_C-9"},                     // 大小写+全部白名单符号
		{"  create-chapter\t\n", "create-chapter"}, // 首尾空白 trim
		{strings.Repeat("k", MaxKeyRunes), strings.Repeat("k", MaxKeyRunes)}, // 恰 100 过
	}
	for _, c := range okCases {
		got, err := NormalizeKey(c.in)
		if err != nil || got != c.want {
			t.Fatalf("NormalizeKey(%q) = (%q,%v)，want (%q,nil)", c.in, got, err, c.want)
		}
	}
	errCases := []struct {
		in    string
		want  error
		label string
	}{
		{"", ErrEmptyKey, "空串"},
		{"   \t\n ", ErrEmptyKey, "纯空白 trim 后为空"},
		{strings.Repeat("k", MaxKeyRunes+1), ErrKeyTooLong, "101 rune 超长"},
		{"create chapter", ErrKeyCharset, "内部空格"},
		{"a\\b", ErrKeyCharset, "反斜杠（路径分隔符残渣）"},
		{"a/b", ErrKeyCharset, "正斜杠（路径分隔符残渣）"},
		{"a:b", ErrKeyCharset, "冒盘符（Windows 残渣）"},
		{"小说章节", ErrKeyCharset, "中文"},
		{"café", ErrKeyCharset, "重音字符"},
		{"a b", ErrKeyCharset, "中间空白"},
		{"a{b", ErrKeyCharset, "花括号"},
		{"a+b", ErrKeyCharset, "加号"},
		{"中文键.json", ErrKeyCharset, "中文+合法扩展名（整体仍拒）"},
	}
	for _, c := range errCases {
		got, err := NormalizeKey(c.in)
		if err == nil || !errors.Is(err, c.want) {
			t.Fatalf("%s: NormalizeKey(%q) = (%q,%v)，want 哨兵 %v", c.label, c.in, got, err, c.want)
		}
	}
}

// TestOverrideJSONShape JSON 标签逐字钉死（与 <DataRoot>/prompt_overrides.json
// 状态文件契约一致，防漂移）：camelCase、key/content/isActive/version/createdAt/
// updatedAt 六键恒出、category/description omitempty 缺省不出。
func TestOverrideJSONShape(t *testing.T) {
	full := Override{
		Key:         "create-chapter",
		Category:    "chapter",
		Description: "章节创作",
		Content:     prompt.Template{Name: "create-chapter", System: "s", Task: "t"},
		IsActive:    true,
		Version:     3,
		CreatedAt:   1757900000000,
		UpdatedAt:   1757900001000,
	}
	b, err := json.Marshal(full)
	if err != nil {
		t.Fatalf("marshal 失败：%v", err)
	}
	want := `{"key":"create-chapter","category":"chapter","description":"章节创作",` +
		`"content":{"name":"create-chapter","system":"s","task":"t","input_sections":null,` +
		`"output":{"format":"","description":""},"constraints":{"must":null,"forbidden":null}},` +
		`"isActive":true,"version":3,"createdAt":1757900000000,"updatedAt":1757900001000}`
	if string(b) != want {
		t.Fatalf("全字段 JSON 形状漂移：\n got  %s\n want %s", b, want)
	}
	minimal := Override{Key: "x", Content: prompt.Template{}}
	b2, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("marshal 失败：%v", err)
	}
	want2 := `{"key":"x","content":{"name":"","system":"","task":"","input_sections":null,` +
		`"output":{"format":"","description":""},"constraints":{"must":null,"forbidden":null}},` +
		`"isActive":false,"version":0,"createdAt":0,"updatedAt":0}`
	if string(b2) != want2 {
		t.Fatalf("最小行 JSON 形状漂移（category/description omitempty 不得出现）：\n got  %s\n want %s", b2, want2)
	}
	// Issue 标签钉死（保存结果 Issues 逐条回传前端）。
	ib, _ := json.Marshal(Issue{Code: CodeLegacyBrace, Severity: SeverityWarn, Message: "m"})
	if string(ib) != `{"code":"legacy-brace","severity":"warn","message":"m"}` {
		t.Fatalf("Issue JSON 形状漂移：%s", ib)
	}
}
