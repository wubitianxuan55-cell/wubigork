// Package workcost 工料法成本核算内核（工料机 × 消耗定额 → 综合单价）。
//
// 用户定调（2026-10）：工料机（人材机）是**基本数据**，综合单价分析表是靠
// 「工料机 × 消耗定额」核算出来的——即建立工料法成本数据库。
//
// 本包承载三层骨架中的核算面（纯函数，无 IO、无 LLM）：
//
//	① 工料机资源库 gf_resources       —— 基准价/现行价，独立主数据
//	② 消耗定额库   gf_quotas/items    —— 每单位子目消耗多少工料机（全局标准）
//	③ 综合单价     Σ(消耗量×资源价) + 管理费/利润/税金   —— 核算**输出**
//
// 关键方向性：综合单价不再是输入。cost_entries.price 降级为核算结果缓存
// （price_derived=1），成本库的检索/分位/对标等消费方读到的仍是同一个
// price 列，因此零改动。
//
// 存储访问（gf_* 表的读写）在 store.go；本文件只放类型与纯函数，便于
// 单测逐条钉住口径。
package workcost

import (
	"math"
	"strings"
)

// Kind 工料机资源类别 + 外委。
//
// 前三个与 cost.Component.Kind 及 cost_entries 的 labor_fee/material_fee/
// machine_fee 三列口径一致：工=人工、料=材料、机=机械。
//
// KindOutsourced（外委）来自**实测最新模版**（旺平矿业 2026-10-07）的行类列。
// 该表汇总行口径为：
//
//	人工 = SUMIF(行类,"人工")
//	材料 = SUMIF(行类,"材料") + SUMIF(行类,"外委")   ← 外委并入材料桶
//	机械 = SUMIF(行类,"机械")
//
// 即「外委」是独立的**行类标签**（水泥窑处置、外购土、监测化验这类不在现场
// 发生人材机的项），但三费归集上算作材料。故它是一等类别，不是材料的别名。
const (
	KindLabor      = "人工"
	KindMaterial   = "材料"
	KindMachine    = "机械"
	KindOutsourced = "外委"
)

// KindOrder 类别标准顺序（人工→材料→机械→外委），列表与明细行统一按此序输出，
// 保证「人工费/材料费/机械费」在分析表里恒定位次。
var KindOrder = []string{KindLabor, KindMaterial, KindMachine, KindOutsourced}

// 含量/价格来源标记（项目级覆盖留痕；对齐 SchemaV27 的 *_source 列）。
const (
	SourceGlobal  = "global"  // 取自消耗定额库的标准含量/资源基准价
	SourceProject = "project" // 项目级覆盖（调量/调价）
	SourceManual  = "manual"  // 手工录入（无定额来源）
	SourceDerived = "derived" // 由低层核算派生
)

// Resource 工料机资源（gf_resources 一行）。
//
// 身份 = (Kind, Title, Spec, Unit)；Code 是定额引用的稳定锚点（改址不改引用）。
// BasePrice 为编制基准价，CurrentPrice 为现行价（信息价更新后）——两者分离
// 才能做「基准价 vs 现行价」调差；核算默认取 CurrentPrice（现行价优先）。
type Resource struct {
	ID           int64    `json:"id"`
	Code         string   `json:"code"`
	Kind         string   `json:"kind"`
	Title        string   `json:"title"`
	Spec         string   `json:"spec"`
	Unit         string   `json:"unit"`
	BasePrice    float64  `json:"basePrice"`
	CurrentPrice float64  `json:"currentPrice"`
	CategoryPath string   `json:"categoryPath"`
	Source       string   `json:"source"`
	Supplier     string   `json:"supplier"`
	Region       string   `json:"region"`
	PriceDate    string   `json:"priceDate"`
	PriceType    string   `json:"priceType"`
	ValidUntil   string   `json:"validUntil"`
	LossRate     float64  `json:"lossRate"`
	Note         string   `json:"note"`
	Tags         []string `json:"tags"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

// EffectivePrice 返回核算取用价：现行价优先，为 0 时回退基准价。
// 现行价与基准价都是「未录入=0」语义（SQLite 默认 0），因此 0 必须回退而非
// 当作真价 0 参与核算——否则一次信息价导入的漏项会把综合单价静默清零。
func (r Resource) EffectivePrice() float64 {
	if r.CurrentPrice > 0 {
		return r.CurrentPrice
	}
	return r.BasePrice
}

// ResourcePrice 资源调价历史（gf_resource_prices 一行）。信息价按期发布，
// 每次发布留一条，旧价保留可回看环比。
type ResourcePrice struct {
	ID         int64   `json:"id"`
	ResourceID int64   `json:"resourceId"`
	Price      float64 `json:"price"`
	Period     string  `json:"period"`
	Region     string  `json:"region"`
	PriceType  string  `json:"priceType"`
	Source     string  `json:"source"`
	FetchedAt  string  `json:"fetchedAt"`
	Note       string  `json:"note"`
}

// Quota 消耗定额（gf_quotas 一行）：某清单子目每单位消耗的工料机标准含量。
// BaseLabor/BaseMaterial/BaseMachine 是定额本本身的基价拆分（编制口径留痕），
// 与实际核算价无关——实际价由 QuotaItem × 资源现行价重算得出。
type Quota struct {
	ID           int64       `json:"id"`
	Code         string      `json:"code"`
	// OrigCode 工作簿里的原始清单编码（编码统一可能改写 Code，溯源靠它）。
	OrigCode     string      `json:"origCode"`
	Title        string      `json:"title"`
	Specialty    string      `json:"specialty"`
	Chapter      string      `json:"chapter"`
	Unit         string      `json:"unit"`
	CategoryPath string      `json:"categoryPath"`
	BaseLabor    float64     `json:"baseLabor"`
	BaseMaterial float64     `json:"baseMaterial"`
	BaseMachine  float64     `json:"baseMachine"`
	Source       string      `json:"source"`
	Region       string      `json:"region"`
	PriceDate    string      `json:"priceDate"`
	Note         string      `json:"note"`
	Status       string      `json:"status"`
	Items        []QuotaItem `json:"items"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

// QuotaItem 定额子目的一条工料机消耗行。
//
// Quantity 是标准含量（每单位子目的消耗量）；ResourceCode 引用 gf_resources.code。
// Amount 刻意不入列：金额 = Quantity × 资源价 是核算派生量，读时用**当前**资源价
// 重算——这正是「资源价一涨、所有引用该子目的综合单价同步重算」的落点。
type QuotaItem struct {
	ID            int64   `json:"id"`
	QuotaCode     string  `json:"quotaCode"`
	ResourceCode  string  `json:"resourceCode"`
	Kind          string  `json:"kind"`
	Title         string  `json:"title"`
	Spec          string  `json:"spec"`
	Unit          string  `json:"unit"`
	Quantity      float64 `json:"quantity"`
	ResourcePrice float64 `json:"resourcePrice"` // 快照价（资源缺失时的兜底追溯）
	LossRate      float64 `json:"lossRate"`      // 行级损耗率，优先级高于资源级
	Note          string  `json:"note"`
	Sort          int     `json:"sort"`
}

// RateSet 取费参数（项目级，对齐实测产物「费用汇总」下半段的取费区）。
//
// ⚠️ 口径铁律（实测产物原话）：「综合单价只含人材机，管理费/利润/税金不进
// 综合单价（在费用汇总下半单列）」。因此取费**不作用于单条综合单价**，只在
// 项目合计层跑一次：Σ(合价) = 直接费 → 才依次推管理费/利润/规费/税。
//
// 取费基数链（百锦路地块土壤修复成本测算表「费用汇总」52-66 行实测）：
//
//	直接费    = Σ(清单合价)              —— 合价 = 工程量 × 综合单价(人材机)
//	企业管理费 = 直接费 × ManagementRate
//	利润      = (直接费 + 企业管理费) × ProfitRate
//	规费      = 直接费 × RegulatoryRate   —— 各专业口径不同，可为 0（不计列）
//	税前合计   = 直接费 + 企业管理费 + 利润 + 规费 + 措施费 + 暂列金额
//	增值税    = 税前合计 × TaxRate
//	含税总造价 = 税前合计 + 增值税
//
// 注意「利润」只累计到管理费（不含规费）——这是与另一份产物（市政道路改造
// 工程成本测算表，利润按「直接费+企管+规费」）的差异点，故基数链做成可插拔
// 策略而非写死（见 FeePolicy.ProfitBaseIncludesRegulatory）。
//
// 零值语义：任一费率 ≤0 视为不计取该段（显式不取费），而非「用默认值」——
// 实测产物里「规费：本测算不计列」「利润：0」是真实存在的口径，必须能被表达。
type RateSet struct {
	ManagementRate float64 `json:"managementRate"`
	RegulatoryRate float64 `json:"regulatoryRate"`
	ProfitRate     float64 `json:"profitRate"`
	TaxRate        float64 `json:"taxRate"`
}

// DefaultRateSet 默认取费口径（市政道路改造工程成本测算表实测值：企管 10%、
// 规费 2%、利润 7%、增值税 9%）。仅作新建项目的初始值——项目建档后费率是
// 项目自己的数据，核算层永远按项目实存费率算，不回落默认值。
func DefaultRateSet() RateSet {
	return RateSet{ManagementRate: 0.10, RegulatoryRate: 0.02, ProfitRate: 0.07, TaxRate: 0.09}
}

// ComposeLine 一条工料机组成行（核算输入：资源 + 含量 [+ 损耗]）。
type ComposeLine struct {
	Kind     string  `json:"kind"`
	Title    string  `json:"title"`
	Unit     string  `json:"unit"`
	Quantity float64 `json:"quantity"` // 消耗量/含量
	Price    float64 `json:"price"`    // 资源单价（现行价优先）
	LossRate float64 `json:"lossRate"` // 损耗率（0=无损耗）
}

// lineAmount 单行资源费 = 含量 × 单价 × (1+损耗率)。负数含量/单价按 0 计
// （invalid 行不计入合计，与 cost 包「含量≤0 或单价≤0 该行非法」口径一致）。
func (l ComposeLine) lineAmount() float64 {
	if l.Quantity <= 0 || l.Price <= 0 {
		return 0
	}
	return l.Quantity * l.Price * (1 + l.LossRate)
}

// Compose 综合单价分析结果（核算**输出**）——**只含人材机**。
//
// 这是「综合单价分析表」一条清单项汇总行的数据形态：人工/材料/机械三费 +
// 小计（= 人材机合价 = 综合单价）。管理费/利润/规费/税金**不在**本结构里——
// 它们只在项目合计层由 ComposeProjectFees 推出（实测产物口径）。
type Compose struct {
	LaborFee float64 `json:"laborFee"`
	// MaterialFee 已含外委（口径：材料 = SUMIF(材料) + SUMIF(外委)）。
	MaterialFee float64 `json:"materialFee"`
	MachineFee  float64 `json:"machineFee"`
	// OutsourcedFee 外委小计（已包含在 MaterialFee 内，此处供分析表拆分展示）。
	OutsourcedFee float64 `json:"outsourcedFee"`
	// OtherFee 类别未识别的资源费（源文件合并段标签等）：计入小计但不进三费
	// 拆分，并留 warn 显形，绝不静默归入某一类。
	OtherFee float64 `json:"otherFee"`
	// Subtotal 人材机合价小计 = 综合单价（人材机口径）。
	Subtotal float64 `json:"subtotal"`
	// CompositePrice 综合单价（人材机口径）= Subtotal，逐项一一对应。
	CompositePrice float64 `json:"compositePrice"`
	// ZeroLines 含量或单价为 0 的组成行数——**不算错误**。实测模版会保留计划
	// 资源的零含量行（如「技术工/带班 ×0」：工序模板列了但本项未用），这是合法
	// 占位；只有负值才是真非法。
	ZeroLines int `json:"zeroLines"`
	// Lines 逐行资源费（Amount 已按含量×单价×(1+损耗) 算出）。
	Lines []ComposeLineView `json:"lines"`
	// Warnings 核算期发现的**真问题**（负含量/负单价/资源缺失），显形而非静默吞掉。
	Warnings []string `json:"warnings"`
}

// ComposeLineView 组成行的核算视图（含派生的资源费与占比）。
type ComposeLineView struct {
	ComposeLine
	Amount float64 `json:"amount"`
	// SharePct 占人材机小计比（%），小计为 0 时为 0。
	SharePct float64 `json:"sharePct"`
}

// Subtotal 是 Compose 的旧字段名兼容别名（DirectFee 在旧版本里表示「直接费」，
// 本口径下它就是人材机小计）。保留访问器避免调用方写错语义。

// ComposeUnitPrice 按工料机组成核算**综合单价（只含人材机）**（纯函数）。
//
// 实测产物「综合单价」表的口径：K = Σ(I×J) = 人材机合价，P = K = 综合单价；
// L/M/N = SUMIF(行类) 拆分三费。本函数就是这三步，**不含任何取费**。
func ComposeUnitPrice(lines []ComposeLine) Compose {
	var c Compose
	byKind := map[string]float64{}
	outsourced := 0.0
	for _, l := range lines {
		amt := l.lineAmount()
		// 零含量/零单价 = 合法占位（模版里「技术工/带班 ×0」是工序模板列了但本项
		// 未用），计数而不告警；**负值**才是真非法，必须显形。
		if l.Quantity == 0 || l.Price == 0 {
			c.ZeroLines++
		} else if l.Quantity < 0 || l.Price < 0 {
			c.Warnings = append(c.Warnings, "含量/单价不得为负："+labelOf(l))
		}
		switch strings.TrimSpace(l.Kind) {
		case KindLabor:
			byKind[KindLabor] += amt
		case KindMaterial:
			byKind[KindMaterial] += amt
		case KindMachine:
			byKind[KindMachine] += amt
		case KindOutsourced:
			// 外委独立成行类，但三费归集并入材料桶——对齐实测模版汇总行
			// 「材料 = SUMIF(材料) + SUMIF(外委)」。outsourced 同时保留外委
			// 小计，供分析表拆分展示。
			byKind[KindMaterial] += amt
			outsourced += amt
		default:
			// 类别未识别：计入直接费但不进三费拆分，并留告警——绝不静默归类。
			if amt > 0 {
				c.Warnings = append(c.Warnings, "类别未识别（不计入人材机拆分）："+labelOf(l))
			}
			byKind["未分类"] += amt
		}
		c.Lines = append(c.Lines, ComposeLineView{ComposeLine: l, Amount: amt})
	}
	c.LaborFee = round2(byKind[KindLabor])
	c.MaterialFee = round2(byKind[KindMaterial])
	c.MachineFee = round2(byKind[KindMachine])
	c.OutsourcedFee = round2(outsourced)
	c.OtherFee = round2(byKind["未分类"])
	c.Subtotal = round2(c.LaborFee + c.MaterialFee + c.MachineFee + byKind["未分类"])
	c.CompositePrice = c.Subtotal
	for i := range c.Lines {
		if c.Subtotal > 0 {
			c.Lines[i].SharePct = round2(c.Lines[i].Amount / c.Subtotal * 100)
		}
	}
	return c
}

// FeePolicy 取费基数链的可插拔口径。
//
// 两份实测产物的差异只有一处：利润基数算不算规费。
//
//	百锦路（土壤修复）：利润 =（直接费+管理费）× 利润率，规费不计列
//	市政道路（模版）  ：利润 =（直接费+管理费+规费）× 利润率
//
// 故做成策略而非写死；零值 = 上面第一种（不累计规费），与最新产物一致。
type FeePolicy struct {
	// ProfitBaseIncludesRegulatory 利润基数是否含规费（默认 false）。
	ProfitBaseIncludesRegulatory bool
}

// FeeResult 项目合计层的取费结果（实测产物「费用汇总」52-66 行）。
type FeeResult struct {
	// DirectFee 分部分项人材机直接费合计 = Σ(合价)。
	DirectFee     float64 `json:"directFee"`
	ManagementFee float64 `json:"managementFee"`
	ProfitFee     float64 `json:"profitFee"`
	RegulatoryFee float64 `json:"regulatoryFee"`
	// MeasuresFee 总价措施费；Contingency 暂列金额（实测产物「总价措施费已列入
	// 分部分项，此处不重复」→ 通常为 0，但口径上必须留位，否则两类费用无处安放）。
	MeasuresFee float64 `json:"measuresFee"`
	Contingency float64 `json:"contingency"`
	// PreTaxTotal 税前合计（不含税造价）= 直接费+企管+利润+规费+措施+暂列。
	PreTaxTotal float64 `json:"preTaxTotal"`
	TaxFee      float64 `json:"taxFee"`
	// Total 含税总造价。
	Total float64 `json:"total"`
	// ControlPrice 招标控制价与差额/利用率（实测产物 63-66 行；无控制价时为 0）。
	ControlPrice   float64 `json:"controlPrice"`
	ControlDiff    float64 `json:"controlDiff"`
	ControlUtilPct float64 `json:"controlUtilPct"`
}

// ComposeProjectFees 项目合计层取费（纯函数）：直接费 → 企管 → 利润 → 规费 →
// 税前合计 → 增值税 → 含税总造价。**不作用于单条综合单价**。
func ComposeProjectFees(directFee float64, rates RateSet, policy FeePolicy, measures, contingency, controlPrice float64) FeeResult {
	var f FeeResult
	f.DirectFee = round2(directFee)
	f.MeasuresFee = round2(measures)
	f.Contingency = round2(contingency)
	f.ManagementFee = round2(f.DirectFee * rates.ManagementRate)
	profitBase := f.DirectFee + f.ManagementFee
	if policy.ProfitBaseIncludesRegulatory {
		f.RegulatoryFee = round2(f.DirectFee * rates.RegulatoryRate)
		profitBase += f.RegulatoryFee
	} else {
		f.RegulatoryFee = round2(f.DirectFee * rates.RegulatoryRate)
	}
	f.ProfitFee = round2(profitBase * rates.ProfitRate)
	f.PreTaxTotal = round2(f.DirectFee + f.ManagementFee + f.ProfitFee + f.RegulatoryFee + f.MeasuresFee + f.Contingency)
	f.TaxFee = round2(f.PreTaxTotal * rates.TaxRate)
	f.Total = round2(f.PreTaxTotal + f.TaxFee)
	f.ControlPrice = round2(controlPrice)
	if f.ControlPrice > 0 {
		f.ControlDiff = round2(f.Total - f.ControlPrice)
		f.ControlUtilPct = round2(f.Total / f.ControlPrice * 100)
	}
	return f
}

// labelOf 组成行的人话标签（告警文案用）。
func labelOf(l ComposeLine) string {
	t := strings.TrimSpace(l.Title)
	if t == "" {
		t = "（未命名资源）"
	}
	if u := strings.TrimSpace(l.Unit); u != "" {
		return t + "(" + u + ")"
	}
	return t
}

// round2 金额四舍五入到分。核算链路每段都收敛到分，避免浮点尾数在明细行与
// 合计行之间产生「差一分」的对不上（模版用 Excel 公式联动，展示口径也是分）。
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Round2 导出舍入（app/导出层复用同一口径，避免各写一份）。
func Round2(v float64) float64 { return round2(v) }
