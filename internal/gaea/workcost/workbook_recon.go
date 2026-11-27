package workcost

// workbook_recon.go — 工作簿内对账（2026-11-27 AI 反馈 #15/#16）。
//
// 测算簿在生成阶段最常见的静默丢钱：明细行溢出覆盖了下一块的定额头、定额行 /
// 清单行的求和区间没覆盖新加的行、三费拆分漏续、清单表引用行号错位——公式不
// 报错，金额悄悄偏低，人工勾稽才发现（实测一次丢 13.8 万）。
//
// 这里拿工作簿**自己的缓存值**按金额链逐级复算，对不上就点名。全是确定性判据，
// 不碰库，导入演练（import_project dry_run=true）时跑：
//
//	定额行金额 H        = Σ 明细行金额 H
//	清单行金额 H        = Σ(定额行金额 × 换算系数)
//	清单行金额 H        = 按明细行「含量 × 工料机价格」三层 ROUND 重算
//	清单行三费 K+L+M    = 清单行金额 H（表带三费列时才核）
//	综合单价「清单工程量」= 清单表该行的工程量
//	清单表综合单价        = 综合单价表该清单行的综合单价；清单表合价 = 工程量 × 综合单价
//	清单表直接费合计      = Σ 清单表合价 = 费用汇总直接费
//	工料机价格表          填了编码却没有单价的行
//
// 没重算的簿缓存值恒 0：要看缓存值的那几条跳过（另有「公式无缓存值」告警兜着）。

import (
	"fmt"
	"math"
)

// ReconIssue 工作簿内对账的一处不平。
type ReconIssue struct {
	Kind   string  `json:"kind"`   // reconKindLabel 的键
	Code   string  `json:"code"`   // 清单编码 / 定额编码 / 工料机编码（合计类为空）
	Row    int     `json:"row"`    // 出问题的行（综合单价表或清单表；0=没有单一行）
	Shown  float64 `json:"shown"`  // 表内缓存值
	Expect float64 `json:"expect"` // 按链复算应有的值
	Detail string  `json:"detail"` // 哪里断了、怎么修
}

var reconKindLabel = map[string]string{
	"quota_range": "定额行金额 ≠ 明细行金额合计",
	"item_range":  "清单行金额 ≠ Σ(定额行金额 × 换算系数)",
	"recompute":   "清单行金额 ≠ 按明细行重算",
	"fee_split":   "清单行三费拆分 K+L+M ≠ 金额",
	"qty_ref":     "综合单价「清单工程量」≠ 清单表工程量",
	"bill_ref":    "清单表综合单价 ≠ 综合单价表",
	"bill_amount": "清单表合价 ≠ 工程量 × 综合单价",
	"bill_total":  "清单表直接费合计 ≠ 各行合价之和",
	"total":       "费用汇总直接费 ≠ 清单表合价合计",
	"empty_price": "工料机价格表有编码没有单价",
}

// Text 回执里的一行。
func (is ReconIssue) Text() string {
	head := fmt.Sprintf("[%s] %s", reconKindLabel[is.Kind], is.Code)
	if is.Row > 0 {
		head += fmt.Sprintf("（第 %d 行）", is.Row)
	}
	if is.Kind == "empty_price" {
		return head + "——" + is.Detail
	}
	return fmt.Sprintf("%s：表内 %.2f / 应为 %.2f（差 %+.2f）——%s", head, is.Shown, is.Expect, is.Expect-is.Shown, is.Detail)
}

// reconTol 每行一分钱的容差：工作簿省略某层 ROUND 时逐行最多差半分，不算断链。
func reconTol(rows int) float64 {
	if rows < 1 {
		rows = 1
	}
	return 0.011*float64(rows) + 0.004
}

func blockFactor(f float64) float64 {
	if f <= 0 {
		return 1 // 空 / 0 = 跟随清单工程量（按 1 计）
	}
	return f
}

// WorkbookRecon 按金额链逐级复算工作簿自己的缓存值，返回对不上的地方（空=全部一致）。
func WorkbookRecon(b *ProjectBundle) []ReconIssue {
	var issues []ReconIssue
	priceByCode := make(map[string]float64, len(b.Resources))
	for _, r := range b.Resources {
		switch {
		case r.Code == "":
			continue
		case r.Price > 0:
			if _, dup := priceByCode[r.Code]; !dup {
				priceByCode[r.Code] = r.Price
			}
		case r.Title != "":
			// 填了编码与名称却没有单价：空价在综合单价里按 0 计，金额静默偏小。
			issues = append(issues, ReconIssue{Kind: "empty_price", Code: r.Code,
				Detail: fmt.Sprintf("「%s」的单价是空的——在工料机价格表 G 列填不含税单价（库里有就查库：cost_project action=resources）", r.Title)})
		}
	}

	linesByBlock := make(map[string][]ProjectConsumption)
	for _, l := range b.Lines {
		linesByBlock[l.ItemCode+"\x00"+l.QuotaCode] = append(linesByBlock[l.ItemCode+"\x00"+l.QuotaCode], l)
	}
	blocksByItem := make(map[string][]ProjectQuotaBlock)
	for _, blk := range b.QuotaBlocks {
		blocksByItem[blk.ItemCode] = append(blocksByItem[blk.ItemCode], blk)
	}

	var billSum float64
	billRows := 0
	for i := range b.Items {
		it := &b.Items[i]
		blocks := blocksByItem[it.Code]
		var (
			lineCount   int
			itemShown   float64 // Σ(定额行金额 × 系数)
			recomputed  float64 // 按明细行「含量 × 价格」三层 ROUND 重算
			firstRow    int
			quotaAmount = map[string]float64{} // 块金额：定额行缓存值，没有就取明细行合计
		)
		for _, blk := range blocks {
			lines := linesByBlock[it.Code+"\x00"+blk.Code]
			lineCount += len(lines)
			if firstRow == 0 {
				firstRow = blk.Row
			}
			var detailSum, blockRecompute float64
			for _, l := range lines {
				detailSum += l.Amount
				p := l.Price
				if p <= 0 {
					p = priceByCode[l.ResourceCode] // 取价公式没出数：按价格表里的同一编码重算
				}
				blockRecompute += round2(l.Quantity * p)
			}
			if blk.Amount != 0 || detailSum != 0 {
				if d := blk.Amount - detailSum; math.Abs(d) > reconTol(len(lines)) {
					issues = append(issues, ReconIssue{Kind: "quota_range", Code: it.Code, Row: blk.Row, Shown: blk.Amount, Expect: detailSum,
						Detail: fmt.Sprintf("定额 %s 行的金额公式没有覆盖它的全部明细行——H 列改成 ROUND(SUM(H<明细起>:H<明细止>),2)，区间要含新加的明细行", blk.Code)})
				}
			}
			quotaAmount[blk.Code] = blk.Amount
			if blk.Amount == 0 {
				quotaAmount[blk.Code] = detailSum
			}
			itemShown += blockFactor(blk.Factor) * quotaAmount[blk.Code]
			recomputed += blockFactor(blk.Factor) * round2(blockRecompute)
		}
		if it.Amount > 0 && len(blocks) > 0 {
			if d := it.Amount - itemShown; math.Abs(d) > reconTol(len(blocks)) {
				issues = append(issues, ReconIssue{Kind: "item_range", Code: it.Code, Row: firstRow, Shown: it.Amount, Expect: itemShown,
					Detail: "清单行金额公式没有覆盖全部定额块——H 列每个块一段 IF(F<定额行>=\"\",1,F<定额行>)*SUM(H<明细起>:H<明细止>)，新加的块要续写"})
			}
			if d := it.Amount - recomputed; math.Abs(d) > reconTol(lineCount)+0.02 {
				issues = append(issues, ReconIssue{Kind: "recompute", Code: it.Code, Row: firstRow, Shown: it.Amount, Expect: recomputed,
					Detail: "按明细行「含量 × 工料机价格」三层 ROUND 重算对不上——明细行的取价公式没出数、含量或层级填错位，或有明细行被跳过（看上方跳过行明细与疑似溢出）"})
			}
		}
		if b.HasFeeSplit && it.Amount > 0 {
			if sum := it.Labor + it.Material + it.Machine; math.Abs(it.Amount-sum) > reconTol(2*len(blocks)) {
				issues = append(issues, ReconIssue{Kind: "fee_split", Code: it.Code, Row: firstRow, Shown: sum, Expect: it.Amount,
					Detail: "清单行三费公式没有覆盖全部定额块——K / M 各续一段 SUMIF（人工 / 机械），L 续 SUMIF(材料)+SUMIF(外委)"})
			}
		}
		// 综合单价「清单工程量」列引的是清单表行号；引错了别的清单项，工程量静默错配。
		if it.SheetQuantity > 0 && it.Quantity > 0 && math.Abs(it.SheetQuantity-it.Quantity) > 1e-6*math.Max(1, it.Quantity) {
			issues = append(issues, ReconIssue{Kind: "qty_ref", Code: it.Code, Shown: it.SheetQuantity, Expect: it.Quantity,
				Detail: "综合单价「清单工程量」列引错了清单表的行——它引的是清单表行号（固定骨架下 = 3+组号），不是综合单价表自己的行号；改完 xlsx_recalc"})
		}
		if it.BillUnitPrice > 0 && it.ListPrice > 0 && math.Abs(it.BillUnitPrice-it.ListPrice) > 0.011 {
			issues = append(issues, ReconIssue{Kind: "bill_ref", Code: it.Code, Row: it.BillRow, Shown: it.BillUnitPrice, Expect: it.ListPrice,
				Detail: "清单表 H 列引的不是该组的清单行——清单表综合单价 = ROUND(综合单价!H<组清单行>,2)"})
		}
		if it.BillUnitPrice > 0 && it.Quantity > 0 && it.BillAmount != 0 {
			if want := it.BillUnitPrice * it.Quantity; math.Abs(it.BillAmount-want) > reconTol(1)+0.02 {
				issues = append(issues, ReconIssue{Kind: "bill_amount", Code: it.Code, Row: it.BillRow, Shown: it.BillAmount, Expect: want,
					Detail: "清单表合价 ≠ 工程量 × 综合单价（I 列公式被手填数值覆盖，或未重算）"})
			}
		}
		if it.BillAmount != 0 {
			billSum += it.BillAmount
			billRows++
		}
	}
	if billRows > 0 {
		if b.BillSheetTotal != 0 && math.Abs(b.BillSheetTotal-billSum) > reconTol(billRows) {
			issues = append(issues, ReconIssue{Kind: "bill_total", Shown: b.BillSheetTotal, Expect: billSum,
				Detail: "清单表「直接费合计」不等于各行合价之和——I1 的 SUM 区间没跟着清单表的行数改"})
		}
		if b.Fee.DirectAmount > 0 && math.Abs(b.Fee.DirectAmount-billSum) > reconTol(billRows) {
			issues = append(issues, ReconIssue{Kind: "total", Shown: b.Fee.DirectAmount, Expect: billSum,
				Detail: "费用汇总直接费不等于清单表合价合计——费用汇总 E16 的 SUM 区间没跟着清单表的行数改"})
		}
	}
	return issues
}
