// Package promptstore t6 提示词工坊的纯函数层（线 A）：模板覆盖行（Override）
// 的键归一、模板保存校验（六规则矩阵）、覆盖表的 upsert/remove 与生效覆盖查询。
// 覆盖状态文件的读写（<DataRoot>/prompt_overrides.json，taskinbox 同款容错读 +
// temp+rename 原子写）归 App 层（线 B gaea_prompt_store.go），工坊面板归前端
// （线 C）。
//
// 设计纪律（先例 internal/taskinbox、internal/skilldistill）：零 IO、零外部
// 依赖——全部数据由调用方组装，表驱动测试。时间戳由调用方传入（nowMs 参数），
// 本包不取时钟（Upsert 行为与真实时间解耦，可确定性测试）。
//
// 规格来源：进度计划/gaea-prompt-workshop-t6-20260916.md §3.2；
// 蒸馏依据 docs/distill/06-prompt-workshop.md（§2.1 覆盖表语义 / §9.3 保存
// 零校验教训 / §11.3 validate.go 六规则矩阵的 gaea 化）。
package promptstore

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/prompt"
)

// 校验规则 code（规格 §3.2 矩阵，钉死 wire 值防漂移）。
const (
	CodeEmptySystem     = "empty-system"     // System trim 后为空
	CodeEmptyTask       = "empty-task"       // Task trim 后为空
	CodeBraceUnbalanced = "brace-unbalanced" // {{ 与 }} 计数不等或存在未闭合 {{
	CodeTooLong         = "too-long"         // 单字段或全模板序列化超长
	CodeUndeclaredVar   = "undeclared-var"   // {{name}} 未在 Parameters 声明
	CodeLegacyBrace     = "legacy-brace"     // 单层 {word_count} 旧语法残渣
)

// Severity 两档（规格 §3.2）：error 阻断落盘；warn 落盘但带回提示。
const (
	SeverityError = "error"
	SeverityWarn  = "warn"
)

// 长度上限（规格 §3.2，rune 口径——中文模板不得按字节腰斩）。
const (
	// MaxFieldRunes System/Task 单字段上限。
	MaxFieldRunes = 20000
	// MaxTemplateRunes 全模板 JSON 序列化上限（整体预算，防 Inputs/Constraints
	// 等未单列上限的字段无限膨胀绕过单字段闸）。
	MaxTemplateRunes = 65536
	// MaxKeyRunes 覆盖键上限：模板名即键（= prompts/*.json 文件名形态），
	// 100 与 MuMu template_key 同宽。
	MaxKeyRunes = 100
)

// NormalizeKey 的固定错误（App 层 errors.Is 判定，先例 taskinbox.ErrEmptyTitle）。
var (
	ErrEmptyKey   = errors.New("promptstore: 模板键为空（trim 后）")
	ErrKeyTooLong = errors.New("promptstore: 模板键超长（>100 rune）")
	ErrKeyCharset = errors.New("promptstore: 模板键含非法字符（仅 A-Za-z0-9._-）")
)

// legacyPlaceholderRe 旧单层占位符样式：{word_count} 形态即 Python format 占位
// 字符集（[a-zA-Z_][a-zA-Z0-9_]*，MuMu §3.2）。中文/含斜杠的花括号（如
// 「第N卷·{阶段名}」「{起/承/转/合/终}」）是字面格式示例，不是变量占位。
var legacyPlaceholderRe = regexp.MustCompile(`\{[A-Za-z_][A-Za-z0-9_]*\}`)

// Issue 一条保存校验问题。任一 SeverityError → 调用方不得落盘；warn 不阻断
// （规格 §3.2「存在任一 error → 调用方不得落盘」）。
type Issue struct {
	Code     string `json:"code"`     // 六规则 code 之一（见上常量）
	Severity string `json:"severity"` // "error" | "warn"
	Message  string `json:"message"`
}

// Override 一行全局覆盖（<DataRoot>/prompt_overrides.json 的 templates[] 元素）。
// json 标签 camelCase 与状态文件契约一致，线 B 视图同字段同标签——测试钉死
// 防漂移（taskinbox.Task 先例）。V1 只做全局覆盖，不带 scope 字段（Q2 裁决）。
type Override struct {
	Key         string          `json:"key"`                // 模板名（= Template.Name；NormalizeKey 把关）
	Category    string          `json:"category,omitempty"` // 工坊分组（列表态优先于内置字段，线 B 合并口径）
	Description string          `json:"description,omitempty"`
	Content     prompt.Template `json:"content"`   // 完整模板正文（整模板覆盖——MuMu §2.1 同款，无字段级合并）
	IsActive    bool            `json:"isActive"`  // false = 禁用 = 引擎 Get 回落磁盘/embed（MuMu is_active 同款语义）
	Version     int             `json:"version"`   // 保存次数（Upsert 自增，非用户输入——MuMu 无版本号教训，Q5 裁决落库）
	CreatedAt   int64           `json:"createdAt"` // unix ms（首存定死，后续保存不可变）
	UpdatedAt   int64           `json:"updatedAt"` // unix ms（每次保存刷新）
}

// NormalizeKey 归一并校验覆盖键：TrimSpace；非空；≤ MaxKeyRunes rune；字符集
// [A-Za-z0-9._-]。模板名即文件名式键（20 个 prompts/*.json 同款形态），不
// 放行路径分隔符/空白/中文（残渣防御，先例 routesuggest.ParseSuggestionID）。
func NormalizeKey(key string) (string, error) {
	k := strings.TrimSpace(key)
	if k == "" {
		return "", ErrEmptyKey
	}
	if r := []rune(k); len(r) > MaxKeyRunes {
		return "", ErrKeyTooLong
	}
	for _, c := range k {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return "", ErrKeyCharset
		}
	}
	return k, nil
}

// fieldText 一段待校验正文及其定位标签（Issue 消息可定位到段）。
type fieldText struct {
	label string
	text  string
}

// templateTextFields 列出会被 BuildSystemPrompt 拼进 system 段的全部正文字段
// （与 internal/prompt BuildSystemPrompt 同源口径）：占位符只有进了渲染链路
// 才有校验意义，纯元数据（name/category/description/parameters）与 input 节
// 标签（进 user 段，V1 不做 {{}} 渲染）不参与花括号/占位校验。
func templateTextFields(t prompt.Template) []fieldText {
	fields := make([]fieldText, 0, 4+len(t.Constraints.Must)+len(t.Constraints.Forbidden)+len(t.Constraints.Style))
	fields = append(fields,
		fieldText{"system", t.System},
		fieldText{"task", t.Task},
		fieldText{"output.format", t.Output.Format},
		fieldText{"output.description", t.Output.Description},
	)
	for i, m := range t.Constraints.Must {
		fields = append(fields, fieldText{label: "constraints.must[" + strconv.Itoa(i) + "]", text: m})
	}
	for i, f := range t.Constraints.Forbidden {
		fields = append(fields, fieldText{label: "constraints.forbidden[" + strconv.Itoa(i) + "]", text: f})
	}
	for i, s := range t.Constraints.Style {
		fields = append(fields, fieldText{label: "constraints.style[" + strconv.Itoa(i) + "]", text: s})
	}
	return fields
}

// braceBalanced 单字段双层花括号平衡（规格 §3.2：{{ 与 }} 计数不等，或存在
// 其后无 }} 的未闭合 {{，任一即 false）。按字段独立判定——BuildSystemPrompt
// 在字段间插入固定标题行（「## 任务」等），跨字段的 {{…}} 不构成合法占位，
// 宁报 error 不放过（保守口径）。
func braceBalanced(s string) bool {
	if strings.Count(s, "{{") != strings.Count(s, "}}") {
		return false
	}
	for i := 0; i < len(s); {
		next := strings.Index(s[i:], "{{")
		if next < 0 {
			return true
		}
		start := i + next
		if strings.Index(s[start+2:], "}}") < 0 {
			return false // 未闭合 {{（其后无 }}）
		}
		i = start + 2
	}
	return true
}

// maskDoubleBrace 把 {{…}} 区间替换为等长空格：legacy-brace 只查真正的单层
// 占位——{{word_count}} 内部子串 {word_count} 不是旧语法残渣（先迁移的模板
// 不能被旧语法告警误伤）。
func maskDoubleBrace(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); {
		next := strings.Index(s[i:], "{{")
		if next < 0 {
			sb.WriteString(s[i:])
			break
		}
		start := i + next
		sb.WriteString(s[i:start])
		closeRel := strings.Index(s[start+2:], "}}")
		if closeRel < 0 {
			sb.WriteString(strings.Repeat(" ", len(s)-start)) // 未闭合 {{：一并遮蔽（brace-unbalanced 已拦）
			break
		}
		end := start + 2 + closeRel + 2
		sb.WriteString(strings.Repeat(" ", end-start))
		i = end
	}
	return sb.String()
}

// Validate 模板保存校验六规则矩阵（规格 §3.2；MuMu §9.3 保存零校验 → 用户写错
// 一个占位符、下一次生成整链 500 的教训修正）。返回全部命中问题（error 与
// warn 可并存），空切片 = 干净模板。规则明细：
//
//	empty-system/empty-task/brace-unbalanced/too-long → error（阻断落盘）；
//	undeclared-var（仅 Parameters 非空时报——Parameters 空视作未维护声明，
//	不追溯旧模板）/legacy-brace → warn（落盘带回提示）。
func Validate(t prompt.Template) []Issue {
	issues := make([]Issue, 0, 4)
	fields := templateTextFields(t)

	// 1. 空段（trim 后为空即不可用——BuildSystemPrompt 拼不出有效角色/任务）。
	if strings.TrimSpace(t.System) == "" {
		issues = append(issues, Issue{Code: CodeEmptySystem, Severity: SeverityError,
			Message: "system 段为空（trim 后）"})
	}
	if strings.TrimSpace(t.Task) == "" {
		issues = append(issues, Issue{Code: CodeEmptyTask, Severity: SeverityError,
			Message: "task 段为空（trim 后）"})
	}

	// 2. 花括号平衡（逐字段；见 braceBalanced 的保守口径说明）。
	for _, f := range fields {
		if !braceBalanced(f.text) {
			issues = append(issues, Issue{Code: CodeBraceUnbalanced, Severity: SeverityError,
				Message: f.label + " 段双层花括号不平衡（{{ 与 }} 计数不等或存在未闭合 {{）"})
		}
	}

	// 3. 超长（rune 口径；全模板序列化是整体预算）。
	if utf8.RuneCountInString(t.System) > MaxFieldRunes {
		issues = append(issues, Issue{Code: CodeTooLong, Severity: SeverityError,
			Message: "system 段超长（>20000 rune）"})
	}
	if utf8.RuneCountInString(t.Task) > MaxFieldRunes {
		issues = append(issues, Issue{Code: CodeTooLong, Severity: SeverityError,
			Message: "task 段超长（>20000 rune）"})
	}
	if b, err := json.Marshal(t); err == nil && utf8.RuneCountInString(string(b)) > MaxTemplateRunes {
		issues = append(issues, Issue{Code: CodeTooLong, Severity: SeverityError,
			Message: "全模板 JSON 序列化超长（>65536 rune）"})
	}

	// 4. 未声明变量（warn；复用 RenderPlaceholders 的扫描器提取 {{name}} 全集
	// ——校验器与渲染器同一套识别逻辑，永不各说各话）。仅 Parameters 非空时
	// 报：空声明视作未维护，不追溯存量模板（规格 §3.2「仅 Parameters 非空时报」）。
	if len(t.Parameters) > 0 {
		declared := make(map[string]bool, len(t.Parameters))
		for _, p := range t.Parameters {
			declared[p] = true
		}
		seen := make(map[string]bool)
		for _, f := range fields {
			_, names := prompt.RenderPlaceholders(f.text, nil)
			for _, n := range names {
				if declared[n] || seen[n] {
					continue
				}
				seen[n] = true
				issues = append(issues, Issue{Code: CodeUndeclaredVar, Severity: SeverityWarn,
					Message: "占位符 {{" + n + "}} 未在 parameters 声明（" + f.label + " 段）"})
			}
		}
	}

	// 5. 旧单层语法（warn：提示 {word_count} 在 {{}} 渲染下不会被替换，
	// substituteWordCount 旧链路已兼容但新预览/工坊语义下应迁移）。
	{
		seen := make(map[string]bool)
		for _, f := range fields {
			for _, m := range legacyPlaceholderRe.FindAllString(maskDoubleBrace(f.text), -1) {
				if seen[m] {
					continue
				}
				seen[m] = true
				issues = append(issues, Issue{Code: CodeLegacyBrace, Severity: SeverityWarn,
					Message: "旧单层语法 " + m + " 不会被 {{}} 渲染替换（" + f.label + " 段），建议迁移为双花括号"})
			}
		}
	}
	return issues
}

// ActiveOverride 找生效覆盖：Key 精确匹配且 IsActive。无行/未启用/键不匹配
// 一律 nil——引擎 Get 据此回落磁盘/embed（三级解析，规格 §3.1）。返回命中行
// 的**副本**指针：线 B 装配闭包会长期持有 &Content，副本隔离切片内部地址，
// 覆盖表重载后旧指针不被原地改写。
func ActiveOverride(entries []Override, key string) *Override {
	for i := range entries {
		if entries[i].Key == key && entries[i].IsActive {
			ov := entries[i]
			return &ov
		}
	}
	return nil
}

// Upsert 覆盖表写入（纯函数，不改写入参切片）：
//   - 同 Key 替换：Version=旧+1、CreatedAt 沿用旧值、UpdatedAt=nowMs，
//     Category/Description/Content/IsActive 取新行；
//   - 新键追加：Version=1、CreatedAt=UpdatedAt=nowMs；
//   - 结果按 Key 字典序稳定（状态文件可 diff、列表序确定）。
//
// ov 的 Version/CreatedAt/UpdatedAt 由本函数重新裁定（版本=保存次数，是系统
// 事实而非用户输入）。键的归一（NormalizeKey）与「引擎存在该键」的存在性
// 把关归调用方（线 B Save 流程），本函数不做 IO 也不猜键。
func Upsert(entries []Override, ov Override, nowMs int64) []Override {
	out := make([]Override, len(entries))
	copy(out, entries)
	for i := range out {
		if out[i].Key != ov.Key {
			continue
		}
		ov.Version = out[i].Version + 1
		ov.CreatedAt = out[i].CreatedAt
		ov.UpdatedAt = nowMs
		out[i] = ov
		return sortByKey(out)
	}
	ov.Version = 1
	ov.CreatedAt = nowMs
	ov.UpdatedAt = nowMs
	out = append(out, ov)
	return sortByKey(out)
}

// Remove 删除覆盖行（恢复内置）：按 Key 精确匹配移除（正常唯一；残渣重复行
// 一并清掉）。返回（新表, 是否删过）；入参切片不被改写。「不存在也幂等成功」
// 的 Reset 口径由线 B 决定，本函数如实报告删没删。
func Remove(entries []Override, key string) ([]Override, bool) {
	out := make([]Override, 0, len(entries))
	removed := false
	for _, e := range entries {
		if e.Key == key {
			removed = true
			continue
		}
		out = append(out, e)
	}
	return out, removed
}

// sortByKey 按 Key 字典序稳定排序（Key 正常唯一；重复键保原相对序）。
func sortByKey(entries []Override) []Override {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries
}
