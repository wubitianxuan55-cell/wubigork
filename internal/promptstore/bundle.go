package promptstore

// t6 提示词工坊·第二刀（规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md）：
// 模板包导出/导入——MuMu §6.4 三态导入算法的本地化（蒸馏依据
// docs/distill/06-prompt-workshop.md §6.4/§11.4）。本文件零 IO 纯函数：
// 包的组装（BuildBundle）与三态决策合并（ImportBundle）都在这里表驱动可测；
// 落盘/绑定归线 B（gaea_prompt_store.go），选文件/另存为归线 C。
//
// 三态口径（IsCustomized=包行是否自定义快照）：
//   false + 基线存在 + 内容相同 → 删本地覆盖行（回落内置）kept_system_default
//   false + 基线存在 + 内容不同 → 写覆盖行 converted_to_custom（内置升级对账）
//   true  + 已知键              → 写覆盖行 created_or_updated
// gaea 两道闸（MuMu 升级，单机无云端人工审核链）：
//   未知键不建行（工坊 V1 不支持自建键，与 Save「未知模板键」同口径）；
//   写行路径前置 Validate，error 级整行跳过不阻断整包（防坏模板毒化生成链）。
// 哈希与比对：SystemContentHash 只是导出侧冗余诊断（sha256(TrimSpace)[:16]，
// MuMu 同款算法）；导入对账用 canonical JSON 逐字比对（Template 是结构体，
// 序列化即规范化文本；MuMu 同口径用内容比对不用哈希）。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/gaea/gaea/internal/prompt"
)

// ── 导出侧（包组装，规格 §3）────────────────────────────────────────────

// BundleTemplate 包内单行：备份语义的全量快照行（内置行也入包，Q1 裁决——
// 换机/重装整包回魂；IsCustomized=导出时覆盖行是否存在（无论启停，Q2），
// 行内容如实取覆盖行 Content（保真备份，含停用行），否则基线 Content。
type BundleTemplate struct {
	Key               string          `json:"key"`
	Category          string          `json:"category,omitempty"`
	Description       string          `json:"description,omitempty"`
	Content           prompt.Template `json:"content"`
	IsActive          bool            `json:"isActive"`
	IsCustomized      bool            `json:"isCustomized"`
	SystemContentHash string          `json:"systemContentHash,omitempty"` // 当前本地基线哈希（诊断冗余，导入不依赖）
	Version           int             `json:"version"`                     // 快照版本（导入不信，Q3：本地 Upsert 自增）
}

// BundleExportStats 导出统计（MuMu statistics 同款口径）。
type BundleExportStats struct {
	Total         int `json:"total"`
	Customized    int `json:"customized"`
	SystemDefault int `json:"systemDefault"`
}

// ExportBundle 模板包：明文 JSON（仅模板正文无用户数据，Q5）；version=1。
type ExportBundle struct {
	Version    int               `json:"version"`
	ExportedAt int64             `json:"exportedAt"`
	Templates  []BundleTemplate  `json:"templates"`
	Statistics BundleExportStats `json:"statistics"`
}

// ContentHash 导出侧冗余诊断哈希：sha256(TrimSpace(s)) hex 前 16 位。
// MuMu prompt_templates.py:28-30 同款算法（长度截断同款：对账用逐字比对，
// 哈希只是包内可读的诊断信息）。
func ContentHash(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])[:16]
}

// canonicalJSON 规范化序列化：json.Marshal 对 map 按键排序、struct 按字段序，
// 同构模板字节稳定——「内容逐字比对」的比较基准。
func canonicalJSON(t prompt.Template) string {
	b, err := json.Marshal(t)
	if err != nil {
		return "" // Template 全基础类型字段，实际不可达；防御回落空串
	}
	return string(b)
}

// SameTemplate 内容逐字比对：canonical JSON 相等。导入三态的「内容 vs 系统
// 内容」判定（MuMu imported_content == system_content 的结构体版）。
func SameTemplate(a, b prompt.Template) bool {
	return canonicalJSON(a) == canonicalJSON(b)
}

// BuildBundle 组装模板包（纯函数）：names（引擎 Names()，调用方保证字典序）
// × 覆盖表 × 基线查找闭包。baseline(name)==nil 的键照入包（IsCustomized 取
// 覆盖行存在性，内容有覆盖行则覆盖行仍是完整正文）——全量备份不挑食。
func BuildBundle(names []string, entries []Override, baseline func(string) *prompt.Template, nowMs int64) ExportBundle {
	b := ExportBundle{Version: 1, ExportedAt: nowMs, Templates: []BundleTemplate{}}
	for _, name := range names {
		row := BundleTemplate{Key: name, IsActive: true}
		if base := baseline(name); base != nil {
			row.Content = *base
			row.Category = base.Category
			row.Description = base.Description
			row.SystemContentHash = ContentHash(canonicalJSON(*base))
		}
		if ov := findPromptOverrideRow(entries, name); ov != nil {
			row.Content = ov.Content
			row.Category = ov.Category
			row.Description = ov.Description
			row.IsActive = ov.IsActive
			row.IsCustomized = true
			row.Version = ov.Version
		}
		b.Templates = append(b.Templates, row)
		b.Statistics.Total++
		if row.IsCustomized {
			b.Statistics.Customized++
		} else {
			b.Statistics.SystemDefault++
		}
	}
	return b
}

// findPromptOverrideRow 在 promptstore 内的镜像（线 B gaea_prompt_store.go 有
// 同名私有函数；纯函数层不能依赖 app 包，本地复制一份三行遍历）。
func findPromptOverrideRow(entries []Override, key string) *Override {
	for i := range entries {
		if entries[i].Key == key {
			ov := entries[i]
			return &ov
		}
	}
	return nil
}

// ── 导入侧（三态决策+合并，规格 §3）────────────────────────────────────

// 导入 action 常量（统计名沿用 MuMu §6.4 表；skipped_* 三值为 gaea 闸口）。
const (
	ActionKeptSystemDefault = "kept_system_default" // 内容同基线 → 删覆盖行回落内置
	ActionConvertedToCustom = "converted_to_custom" // 内置升级后不同 → 转自定义行
	ActionCreatedOrUpdate   = "created_or_updated"  // 自定义快照 → 直接写行
	ActionSkippedInvalid    = "skipped_invalid"     // Validate error 级 → 跳行（gaea 闸）
	ActionSkippedUnknown    = "skipped_unknown"     // 未知键 → 不建行（gaea 闸）
	ActionSkippedDuplicate  = "skipped_duplicate"   // 包内重复键 → 首见生效（gaea 闸）
)

// BundleImportStats 导入统计（MuMu statistics 同名口径 + skipped_* 扩展）。
type BundleImportStats struct {
	Total             int `json:"total"`
	KeptSystemDefault int `json:"keptSystemDefault"`
	ConvertedToCustom int `json:"convertedToCustom"`
	CreatedOrUpdate   int `json:"createdOrUpdate"`
	SkippedInvalid    int `json:"skippedInvalid"`
	SkippedUnknown    int `json:"skippedUnknown"`
	SkippedDuplicate  int `json:"skippedDuplicate"`
}

// BundleImportOutcome 单行导入结果：action + 跳过原因（skipped_invalid 带校验
// 问题摘要，前端结果弹窗逐行展示）。
type BundleImportOutcome struct {
	Key    string `json:"key"`
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

// BundleImportResult 导入结果：Applied=至少一行变更（线 B 落盘判据）；空包/
// 全跳过时 false，不算错误。
type BundleImportResult struct {
	Applied    bool                  `json:"applied"`
	Statistics BundleImportStats     `json:"statistics"`
	Outcomes   []BundleImportOutcome `json:"outcomes"`
}

// ImportBundle 三态导入决策+合并（纯函数，不落盘）：entries 进 → 合并新表+
// 结果出。engineHas=引擎已知键判定（未知键闸）；baseline=基线查找（内容对账）。
// Version/CreatedAt/UpdatedAt 由 Upsert 重新裁定（Q3：不信包内版本防倒退）。
// 入参 entries 不被改写；返回表恒非 nil。
func ImportBundle(b ExportBundle, entries []Override,
	engineHas func(string) bool, baseline func(string) *prompt.Template,
	nowMs int64) ([]Override, BundleImportResult) {

	merged := make([]Override, len(entries))
	copy(merged, entries)
	res := BundleImportResult{Statistics: BundleImportStats{Total: len(b.Templates)}, Outcomes: []BundleImportOutcome{}}
	seen := make(map[string]bool, len(b.Templates))

	for _, item := range b.Templates {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: item.Key, Action: ActionSkippedInvalid, Reason: "空模板键"})
			res.Statistics.SkippedInvalid++
			continue
		}
		if seen[key] {
			res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: key, Action: ActionSkippedDuplicate, Reason: "包内重复键，首见生效"})
			res.Statistics.SkippedDuplicate++
			continue
		}
		seen[key] = true

		if !engineHas(key) {
			res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: key, Action: ActionSkippedUnknown, Reason: "未知模板键（工坊 V1 不支持自建新键）"})
			res.Statistics.SkippedUnknown++
			continue
		}

		// 写行路径统一过校验闸（error 级跳行不阻断整包；warn 放行——与 Save
		// 同分级口径）。
		content := item.Content
		content.Name = key // 覆盖行模板名与键对齐（Save 同款归位）
		if issues := Validate(content); hasErrorIssue(issues) {
			res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: key, Action: ActionSkippedInvalid, Reason: summarizeIssues(issues)})
			res.Statistics.SkippedInvalid++
			continue
		}

		if !item.IsCustomized {
			base := baseline(key)
			if base != nil && SameTemplate(content, *base) {
				// 内容与当前基线逐字相同 → 该行本就该走内置：删本地覆盖行
				//（即使存在），回落系统默认。
				next, removed := Remove(merged, key)
				if removed {
					merged = next
					res.Applied = true
				}
				res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: key, Action: ActionKeptSystemDefault})
				res.Statistics.KeptSystemDefault++
				continue
			}
			// 基线不同（内置升级或用户改过内置快照）→ 转自定义行保住包内容。
			merged = upsertImported(merged, key, item, content, nowMs)
			res.Applied = true
			res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: key, Action: ActionConvertedToCustom, Reason: "与当前内置基线不同，已转为自定义覆盖"})
			res.Statistics.ConvertedToCustom++
			continue
		}

		merged = upsertImported(merged, key, item, content, nowMs)
		res.Applied = true
		res.Outcomes = append(res.Outcomes, BundleImportOutcome{Key: key, Action: ActionCreatedOrUpdate})
		res.Statistics.CreatedOrUpdate++
	}
	return merged, res
}

// upsertImported 写入一行导入覆盖（Upsert 包装：元数据取包行，版本/时间由
// Upsert 裁定）。
func upsertImported(entries []Override, key string, item BundleTemplate, content prompt.Template, nowMs int64) []Override {
	return Upsert(entries, Override{
		Key:         key,
		Category:    item.Category,
		Description: item.Description,
		Content:     content,
		IsActive:    item.IsActive,
	}, nowMs)
}

// hasErrorIssue 校验问题里是否有 error 级（阻断口径与线 B Save 一致）。
func hasErrorIssue(issues []Issue) bool {
	for _, is := range issues {
		if is.Severity == SeverityError {
			return true
		}
	}
	return false
}

// summarizeIssues 校验问题摘要（skipped_invalid 的 reason；error 优先，至多
// 三条防刷屏）。
func summarizeIssues(issues []Issue) string {
	var msgs []string
	for _, is := range issues {
		if is.Severity != SeverityError {
			continue
		}
		msgs = append(msgs, is.Message)
		if len(msgs) == 3 {
			break
		}
	}
	if len(msgs) == 0 && len(issues) > 0 {
		msgs = append(msgs, issues[0].Message)
	}
	if len(msgs) == 0 {
		return "校验未通过"
	}
	return "校验未通过：" + strings.Join(msgs, "；")
}
