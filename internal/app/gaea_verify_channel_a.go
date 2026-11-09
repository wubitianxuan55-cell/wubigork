package app

// v4.9.1 Verifier 通道 A 引用级深化（审计欠账收口）：xlsx_apply 在「公式重算
// 零错误」之上增加「声明 ↔ 实况」引用级比对——证据卡 v4.9.1 起随卡落盘
// opsJson（机器可读 op 载荷），复核时逐条回读目标工作簿：
//   - set_value：单元格实际值与声明值一致（数值按浮点等值比较，容差 1e-9）；
//   - set_formula：单元格实际公式与声明公式一致（去空白后精确比较）；
//   - replace：替换单元格包含 Replace 且不再包含 Find。
// 其余 op 类型（fill_range/transform/split_column/set_style 等批量或样式类）
// 不可单格核对，计入 skipped（诚实降级，宁漏勿误）。旧证据卡无 opsJson →
// 声明比对不适用。全簿公式评估错误（含断链 #REF! 的求值结果）由既有 recalc
// 重算承担，本层不做全簿公式扫描（避免字符串级误报）。

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/docxedit"
	"github.com/gaea/gaea/internal/office/pptxedit"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/xuri/excelize/v2"
)

// verifyClaimMaxOps 声明比对逐条核对的 op 上限（防超卡拖慢复核）。
const verifyClaimMaxOps = 32

// verifyXlsxChannelADeep 重算零错误后的引用级「声明↔实况」比对。返回追加在
// 通道 A 文案之后的复核结论（fail 前缀会升级整条通道 A 结论）。
func verifyXlsxChannelADeep(f *excelize.File, rec evidence.ChangeRecord) string {
	if strings.TrimSpace(rec.OpsJSON) == "" {
		return "；旧证据卡无 opsJson，声明比对不适用"
	}
	var ops []xlsxedit.Op
	if err := json.Unmarshal([]byte(rec.OpsJSON), &ops); err != nil {
		return "；warn: opsJson 解析失败，跳过声明比对"
	}
	if len(ops) > verifyClaimMaxOps {
		ops = ops[:verifyClaimMaxOps]
	}
	checked, mismatched, skipped := 0, 0, 0
	var examples []string
	noteMismatch := func(s string) {
		if len(examples) < 3 {
			examples = append(examples, s)
		}
	}

	for _, op := range ops {
		if op.Sheet == "" {
			skipped++
			continue
		}
		switch op.Type {
		case "set_value":
			actual, err := f.GetCellValue(op.Sheet, op.Target)
			if err != nil {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 不可读（%v）", op.Sheet, op.Target, err))
				continue
			}
			if !cellValueEquals(fmt.Sprint(op.Value), actual) {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 预期「%s」实际「%s」", op.Sheet, op.Target, fmt.Sprint(op.Value), actual))
				continue
			}
			checked++
		case "set_formula":
			actual, err := f.GetCellFormula(op.Sheet, op.Target)
			if err != nil {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 公式不可读（%v）", op.Sheet, op.Target, err))
				continue
			}
			if compactFormula(actual) != compactFormula(fmt.Sprint(op.Formula)) {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 公式预期「%s」实际「%s」", op.Sheet, op.Target, op.Formula, actual))
				continue
			}
			checked++
		case "replace":
			if op.Target == "" || op.Find == "" {
				skipped++
				continue
			}
			actual, err := f.GetCellValue(op.Sheet, op.Target)
			if err != nil {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 不可读（%v）", op.Sheet, op.Target, err))
				continue
			}
			if !strings.Contains(actual, op.Replace) || strings.Contains(actual, op.Find) {
				mismatched++
				noteMismatch(fmt.Sprintf("%s!%s 替换未如实落盘（实际「%s」）", op.Sheet, op.Target, actual))
				continue
			}
			checked++
		default:
			skipped++
		}
	}

	switch {
	case mismatched > 0:
		return fmt.Sprintf("；fail: 引用级复核发现 %d 项声明与实况不符（%s）",
			mismatched, strings.Join(examples, "；"))
	case checked == 0:
		return fmt.Sprintf("；引用级声明比对无可核对项（跳过 %d 项批量/样式类）", skipped)
	default:
		suffix := ""
		if skipped > 0 {
			suffix = fmt.Sprintf("，跳过 %d 项批量/样式类", skipped)
		}
		return fmt.Sprintf("；引用级复核 %d 项声明与实况一致%s", checked, suffix)
	}
}

// cellValueEquals 声明值与单元格实际值等值比较：两侧都能解析为数值时按浮点
// 等值（42 ≈ 42.0）；否则按去空白字符串精确比较。
func cellValueEquals(expected, actual string) bool {
	e, eerr := strconv.ParseFloat(strings.TrimSpace(expected), 64)
	a, aerr := strconv.ParseFloat(strings.TrimSpace(actual), 64)
	if eerr == nil && aerr == nil {
		return math.Abs(e-a) <= 1e-9*math.Max(1, math.Abs(e))
	}
	return strings.TrimSpace(expected) == strings.TrimSpace(actual)
}

// compactFormula 公式归一（去前导 = 与空白）——applyOne 落盘时剥掉 "=" 前缀，
// excelize 存储规范化（空格、参数间隔）也可能与声明原文有无害差异。
func compactFormula(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "=")
	return strings.ReplaceAll(s, " ", "")
}

// ── P2 通道 A 引用级深化（2026-11-10，众神殿 pan_1794234593182 裁决 action-5
// 前半：docx/pptx 从「目标可读」升级为引用级比对；声明载荷走 opsJSON，
// 旧卡无载荷=诚实跳过声明比对，不冒充核对）。 ─────────────────────────

// docxApplyClaims docx_apply 证据卡的声明载荷（opsJSON）。
type docxApplyClaims struct {
	SelectedText string `json:"selected_text"`
	Replacement  string `json:"replacement"`
}

// docxApplyClaimsOf 取 docx_apply 声明：opsJSON 优先；旧卡回退解析摘要
// 「→ X（修订制写入）」（只有替换文本可取，选区未知）。
func docxApplyClaimsOf(rec evidence.ChangeRecord) (docxApplyClaims, bool) {
	var c docxApplyClaims
	if strings.TrimSpace(rec.OpsJSON) != "" {
		if err := json.Unmarshal([]byte(rec.OpsJSON), &c); err == nil && strings.TrimSpace(c.Replacement) != "" {
			return c, true
		}
	}
	s := rec.AfterSummary
	i := strings.Index(s, "→")
	j := strings.LastIndex(s, "（修订制写入）")
	if i >= 0 && j > i {
		c.Replacement = strings.TrimSpace(s[i+3 : j])
		return c, true
	}
	return c, false
}

// pptxApplyClaims pptx_apply 证据卡的声明载荷（opsJSON）。
type pptxApplyClaims struct {
	Slide       int    `json:"slide"`
	Target      string `json:"target"`
	Replacement string `json:"replacement"`
}

// pptxApplyClaimsOf 取 pptx_apply 声明：opsJSON 优先；旧卡回退解析摘要
// 「p1 "target"」/「p1 → "repl"（…）」。
func pptxApplyClaimsOf(rec evidence.ChangeRecord) (pptxApplyClaims, bool) {
	var c pptxApplyClaims
	if strings.TrimSpace(rec.OpsJSON) != "" {
		if err := json.Unmarshal([]byte(rec.OpsJSON), &c); err == nil && strings.TrimSpace(c.Replacement) != "" {
			return c, true
		}
	}
	re := regexp.MustCompile(`p(\d+)\s+"([^"]*)"`)
	if m := re.FindStringSubmatch(rec.BeforeSummary); m != nil {
		c.Slide, _ = strconv.Atoi(m[1])
		c.Target = m[2]
	}
	m2 := regexp.MustCompile(`p(\d+)\s+→\s+"([^"]*)"`)
	if mm := m2.FindStringSubmatch(rec.AfterSummary); mm != nil {
		c.Slide, _ = strconv.Atoi(mm[1])
		c.Replacement = mm[2]
		return c, true
	}
	return c, false
}

// verifyDocxApplyChannelADeep docx_apply 引用级：文档可解析 + 替换后文本在
// 文档文本流中 + 本作者待处理修订在册（修订制写入的物证）。
func verifyDocxApplyChannelADeep(target string, rec evidence.ChangeRecord) string {
	const author = "gaea AI"
	n, err := docxedit.PendingChangeCount(target, author)
	if err != nil {
		return "fail: 文档不可解析 " + err.Error()
	}
	claims, ok := docxApplyClaimsOf(rec)
	if !ok {
		return fmt.Sprintf("pass: 文档可解析，%d 处待处理修订在册（旧卡无声明载荷，引用级比对跳过）", n)
	}
	has, err := docxedit.DocumentContainsText(target, claims.Replacement)
	if err != nil {
		return "fail: 文档不可解析 " + err.Error()
	}
	if !has {
		return "fail: 替换后文本未出现在文档中（声明与实况不一致）"
	}
	if n == 0 {
		return "fail: 替换文本在文档中，但无本作者待处理修订（修订制写入未生效或已被接受/拒绝）"
	}
	return fmt.Sprintf("pass: 文档可解析，替换文本在文档中且 %d 处待处理修订在册（引用级）", n)
}

// verifyDocxAcceptChannelADeep docx_accept 引用级：文档可解析且无本引擎作者
// 的待处理修订（清除修订标记的实况；其他作者的修订不归本引擎管，不计）。
func verifyDocxAcceptChannelADeep(target string) string {
	const author = "gaea AI"
	n, err := docxedit.PendingChangeCount(target, author)
	if err != nil {
		return "fail: 文档不可解析 " + err.Error()
	}
	if n > 0 {
		return fmt.Sprintf("fail: 仍有本引擎待处理修订 %d 处（接受未生效）", n)
	}
	return "pass: 文档可解析，无本引擎待处理修订（引用级）"
}

// verifyPptxApplyChannelADeep pptx_apply 引用级：LocateText 读回目标页，
// 替换文本在、目标文本不在（多处出现仅替换首处=残留如实注记不判失败）。
func verifyPptxApplyChannelADeep(target string, rec evidence.ChangeRecord) string {
	slides, err := pptxedit.LocateText(target)
	if err != nil {
		return "fail: 文档不可解析 " + err.Error()
	}
	claims, ok := pptxApplyClaimsOf(rec)
	if !ok {
		return fmt.Sprintf("pass: 文档可解析，共 %d 页（旧卡无声明载荷，引用级比对跳过）", len(slides))
	}
	if claims.Slide < 1 || claims.Slide > len(slides) {
		return fmt.Sprintf("fail: 第 %d 页不存在（文档共 %d 页）", claims.Slide, len(slides))
	}
	joined := strings.Join(slides[claims.Slide-1].Paragraphs, "\n")
	if !strings.Contains(joined, claims.Replacement) {
		return fmt.Sprintf("fail: 替换后文本未出现在第 %d 页（声明与实况不一致）", claims.Slide)
	}
	if strings.Contains(joined, claims.Target) {
		return fmt.Sprintf("pass: 第 %d 页替换文本在（引用级；目标文本仍有残留——多处出现仅替换首处）", claims.Slide)
	}
	return fmt.Sprintf("pass: 第 %d 页替换文本在、目标文本已不在（引用级）", claims.Slide)
}

// verifyOfficeGenerateChannelA office_generate 引用级：交付文件存在且容器头
// 有效（docx/pptx/xlsx=OOXML zip 容器 PK 头；md/pdf=可读非空）。排版级复核
// 不在本通道（doc_render 预览 + 条件视觉属 P2 后续）。
func verifyOfficeGenerateChannelA(target string) string {
	st, err := os.Stat(target)
	if err != nil {
		return "fail: 交付文件不存在"
	}
	if st.Size() == 0 {
		return "fail: 交付文件为空"
	}
	switch strings.ToLower(filepath.Ext(target)) {
	case ".docx", ".pptx", ".xlsx":
		head := make([]byte, 2)
		f, err := os.Open(target)
		if err != nil {
			return "fail: 交付文件不可读"
		}
		defer func() { _ = f.Close() }()
		if _, err := io.ReadFull(f, head); err != nil {
			return "fail: 交付文件过短（非 OOXML 容器）"
		}
		if head[0] != 'P' || head[1] != 'K' {
			return "fail: 交付文件不是有效 OOXML 容器（缺 zip 头）"
		}
		return "pass: 交付文件存在且 OOXML 容器头有效（引用级；排版级复核走 doc_render 预览）"
	default:
		return "pass: 交付文件存在且非空（引用级）"
	}
}
