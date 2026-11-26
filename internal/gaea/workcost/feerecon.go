package workcost

// feerecon.go — 费用汇总对账：拿工作簿「费用汇总」表的缓存金额反推各段的
// 计算基数，与模版口径（导出、空白模版、库内现算三处同源）比对。
//
// 为什么要有它（2026-11-25 铜陵六表反馈 #15）：导入后库内含税 66,084,514.03
// 对工作簿 65,961,728.30 差 122,785.73 元，回执只报数不说性质，模型按「不等即
// 口径分叉先排查」手算逐行比了 5 轮，结论还是错的——它判成「单价舍位」，实际
// 98% 是利润基数口径分叉（工作簿 = 直接费 × 4%，库内 = (直接费+管理费) × 4%），
// 舍位只占约 1,752 元。差异必须由程序按性质拆开说。
//
// 纪律：这里只做「工作簿自己写的数 ↔ 模版口径」的确定性核对，不判断哪种
// 口径对——利润基数各项目实测产物本就不一（见 FeePolicy），回执把两边都摆出来。

import (
	"fmt"
	"math"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 取费行定段的关键词（按列序取第一个命中取费词的文本格）。顺序有讲究：
// 「招标控制价（含税）」含「含税」、「税前合计（不含税造价）」含「含税」，
// 所以 控制价、税前 必须排在含税总造价之前。
var feeRowKinds = []struct {
	kind     string
	keywords []string
}{
	{"control", []string{"控制价"}},
	{"pretax", []string{"税前"}},
	{"total", []string{"含税总造价", "含税工程造价", "含税合计", "总造价", "工程造价"}},
	{"direct", []string{"直接费"}},
	{"mgmt", []string{"管理费"}},
	{"profit", []string{"利润"}},
	{"regulatory", []string{"规费"}},
	{"tax", []string{"增值税", "税金"}},
}

// feeSeqRunes 序号格的字符集（「一」「十二」「3.1」「（一）」）。只由这些字符
// 组成的文本格是序号，定段时跳过。
const feeSeqRunes = "一二三四五六七八九十零〇0123456789.．、()（）"

func isFeeSeqCell(t string) bool {
	if t == "" {
		return false
	}
	for _, r := range t {
		if !strings.ContainsRune(feeSeqRunes, r) {
			return false
		}
	}
	return true
}

// feeRowKind 判断一行是哪一段取费：按列序找第一个「非序号、非公式、非数字」
// 的文本格，看它含哪个取费词。不是取费行返回空串——「分部小计」块的表头
// （含「占直接费」）与合计行因此不会被误认。
func feeRowKind(vals []string) string {
	for _, v := range vals {
		t := strings.TrimSpace(v)
		if t == "" || strings.HasPrefix(t, "=") || isFeeSeqCell(t) {
			continue
		}
		if _, isNum := parseNum(t); isNum {
			continue
		}
		for _, k := range feeRowKinds {
			for _, kw := range k.keywords {
				if strings.Contains(t, kw) {
					if k.kind == "control" && !isControlPriceLabel(t) {
						// 控制价的派生行（利用率 / 下浮率 / 占比）不是控制价本身：
						// 此前照样定段成 control，金额栏的比率被当成控制价
						// （2026-11-26 造价反馈，百锦路 #7）。
						continue
					}
					return k.kind
				}
			}
		}
		return ""
	}
	return ""
}

// feeRowAmount 取一行的金额（缓存值）：优先表头定位到的「金额」列；表头没
// 认出来时退一步取行内最大的正数（金额列通常是该行最大的数，费率恒 <1）。
func feeRowAmount(f *excelize.File, sheet string, r int, vals []string, headerKnown bool, amtCol int) float64 {
	if headerKnown && amtCol > 0 && amtCol <= len(vals) {
		cell, _ := excelize.CoordinatesToCellName(amtCol, r)
		return cellValue(f, sheet, cell)
	}
	best := 0.0
	for c := range vals {
		cell, _ := excelize.CoordinatesToCellName(c+1, r)
		if v := cellValue(f, sheet, cell); v > best {
			best = v
		}
	}
	return best
}

// setAmount 记一段的金额（首次为准）。直接费与含税总造价都有了才算「可对账」。
func (fee *ProjectFeeParams) setAmount(kind string, v float64) {
	switch kind {
	case "direct":
		if fee.DirectAmount == 0 {
			fee.DirectAmount = v
		}
	case "mgmt":
		if fee.ManagementAmount == 0 {
			fee.ManagementAmount = v
		}
	case "profit":
		if fee.ProfitAmount == 0 {
			fee.ProfitAmount = v
		}
	case "regulatory":
		if fee.RegulatoryAmount == 0 {
			fee.RegulatoryAmount = v
		}
	case "pretax":
		if fee.PreTaxAmount == 0 {
			fee.PreTaxAmount = v
		}
	case "tax":
		if fee.TaxAmount == 0 {
			fee.TaxAmount = v
		}
	case "total":
		if fee.TotalAmount == 0 {
			fee.TotalAmount = v
		}
	case "control":
		if fee.ControlPrice == 0 {
			fee.ControlPrice = v
		}
	}
	fee.HasAmounts = fee.DirectAmount > 0 && fee.TotalAmount > 0
}

// FeeChainFinding 取费链一处对不上的地方。
type FeeChainFinding struct {
	// Item 段名：管理费 / 利润 / 规费 / 增值税 / 税前合计 / 含税总造价。
	Item string `json:"item"`
	// Workbook 工作簿实际用的基数（人话），如「直接费 × 4%」。
	Workbook string `json:"workbook"`
	// Template 模版与库内现算用的基数（人话），如「（直接费+管理费）× 4%」。
	Template string `json:"template"`
	// Amount / Expected 工作簿该段金额 / 按模版口径应为；Diff = Expected − Amount。
	Amount   float64 `json:"amount"`
	Expected float64 `json:"expected"`
	Diff     float64 `json:"diff"`
}

type feeBase struct {
	name string
	v    float64
}

// feeTol 对账容差：相对万分之一、绝对不少于 1 元（缓存值经重算，正常不会
// 超出；超出即基数不同）。
func feeTol(expected float64) float64 {
	return math.Max(1, math.Abs(expected)*1e-4)
}

func feeNear(a, b float64) bool { return math.Abs(a-b) <= feeTol(b) }

func feePct(rate float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", rate*100), "0"), ".") + "%"
}

// feeBaseName 在候选基数里找出能复现 amount 的那一个，返回它的人话；都对不上
// 如实说对不上。
func feeBaseName(amount, rate float64, candidates []feeBase) string {
	for _, c := range candidates {
		if feeNear(amount, c.v*rate) {
			return c.name + " × " + feePct(rate)
		}
	}
	return "对不上任何常见基数（金额 " + fmt.Sprintf("%.2f", amount) + "）"
}

// CheckFeeChain 核对工作簿费用汇总的取费链：每一段拿「缓存金额」反推它用的
// 基数，再与模版口径比。全部一致或没有缓存金额时返回空切片。
//
// 模版口径（render.go writeFeeSheet 与 ComposeProjectFees 同源）：
//
//	管理费 = 直接费 × 管理费率；利润 =（直接费+管理费）× 利润率；规费 = 直接费 × 规费率
//	税前合计 = 直接费+管理费+利润+规费；增值税 = 税前合计 × 税率；含税总造价 = 税前合计+增值税
//
// 利润基数反推出的工作簿口径原样报出（「直接费 × 4%」），不判对错。
func CheckFeeChain(fee ProjectFeeParams) []FeeChainFinding {
	if !fee.HasAmounts {
		return nil
	}
	var out []FeeChainFinding
	direct, mgmt, profit := fee.DirectAmount, fee.ManagementAmount, fee.ProfitAmount
	reg, pretax, tax := fee.RegulatoryAmount, fee.PreTaxAmount, fee.TaxAmount

	rateRow := func(item string, amount, rate, base float64, template string, cands []feeBase) {
		if amount == 0 && rate == 0 {
			return // 该段不计取
		}
		expected := base * rate
		if feeNear(amount, expected) {
			return
		}
		out = append(out, FeeChainFinding{Item: item, Workbook: feeBaseName(amount, rate, cands),
			Template: template, Amount: amount, Expected: expected, Diff: expected - amount})
	}
	sumRow := func(item string, amount, expected float64, template string) {
		if amount == 0 && expected == 0 {
			return
		}
		if feeNear(amount, expected) {
			return
		}
		out = append(out, FeeChainFinding{Item: item, Workbook: fmt.Sprintf("工作簿写的 %.2f", amount),
			Template: template, Amount: amount, Expected: expected, Diff: expected - amount})
	}

	rateRow("管理费", mgmt, fee.ManagementRate, direct, "直接费 × "+feePct(fee.ManagementRate),
		[]feeBase{{"直接费", direct}, {"直接费+规费", direct + reg}, {"税前合计", pretax}})
	rateRow("利润", profit, fee.ProfitRate, direct+mgmt, "（直接费+管理费）× "+feePct(fee.ProfitRate),
		[]feeBase{{"直接费", direct}, {"直接费+规费", direct + reg}, {"直接费+管理费", direct + mgmt}, {"税前合计", pretax}})
	rateRow("规费", reg, fee.RegulatoryRate, direct, "直接费 × "+feePct(fee.RegulatoryRate),
		[]feeBase{{"直接费+管理费", direct + mgmt}, {"税前合计", pretax}})
	sumRow("税前合计", pretax, direct+mgmt+profit+reg, "直接费+管理费+利润+规费")
	rateRow("增值税", tax, fee.TaxRate, pretax, "税前合计 × "+feePct(fee.TaxRate),
		[]feeBase{{"直接费", direct}, {"直接费+管理费+利润", direct + mgmt + profit}})
	sumRow("含税总造价", fee.TotalAmount, pretax+tax, "税前合计+增值税")
	return out
}
