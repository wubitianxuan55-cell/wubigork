// seed.go —— 存量成本条目 → 工料机资源库的自动归类（纯函数，可单测）。
//
// 背景（实测 2026-10 用户库）：cost_entries 1590 条里约 1564 条是**纯资源价**
// （管材管件 273、电线电缆 226、市政绿化材料 184、阀门 89、钢材 76…），
// 全库没有一个字的工料机组成，name 与 title 完全相同、code 列全空。因此
// 「全量重新组合核算」没有输入数据可算，正确做法是把这批纯资源价归入新的
// 工料机资源库（工料法第①层），综合单价另从最近项目录入。
//
// 本文件的职责只有一件：**判定一条成本条目属于工/料/机哪一类、以及它作为
// 资源时的身份字段**。判定纯函数、无 IO——归类的正确性可以逐条钉住，入库
// 动作由 seed 的 apply 层（app 侧）负责。
package workcost

import (
	"strings"
)

// 顶层分类路径前缀（成本库既有口径，见 cost.EnsureDefaultCategories）。
const (
	pathMaterials = "材料"
	pathMachinery = "机械"
	pathLabor     = "人工"
)

// 分类关键词。按「先精确后宽泛」的顺序匹配；命中即定类。
// 口径来源：实测库里真实存在的一级/二级分类名（材料/安装材料/管材管件、
// 材料/土建材料/钢材、机械/土方机械、其他/处置 等）。
var (
	laborKeywords = []string{
		"人工", "工日", "普工", "技工", "劳务", "班组", "带班",
	}
	machineKeywords = []string{
		"机械", "台班", "挖掘机", "装载机", "自卸", "压路机", "摊铺机", "吊车",
		"汽车吊", "泵车", "搅拌车", "罐车", "钻机", "打桩", "破碎锤", "发电机",
		"空压机", "洒水车", "雾炮", "吸污车", "设备租赁", "脚手架", "筛分斗",
	}
	// 「外委」是**独立行类**——实测最新模版（旺平矿业 2026-10-07）把「水泥窑
	// 协同处置」「外购清洁土」「监测化验」单列为行类=外委，汇总行口径为
	// 材料 = SUMIF(材料) + SUMIF(外委)。故外委不再混进人工桶（早期百锦路表
	// 曾把外委标为人工，口径已随最新模版演进）。
	outsourcedKeywords = []string{
		"外委", "委外", "外购", "外包", "分包", "第三方",
		// 处置类外委：实测模版把「水泥窑协同处置」标为行类=外委（不在现场发生
		// 人材机，是委托处置服务），而标题里没有「外委」字样。
		"水泥窑", "协同处置", "委外处置", "危废处置", "污水厂接收",
	}
	// 现场发生的技术服务（编制/设计/论证/评估/检测）仍是人工——实测产物把
	// 「修复方案编制及评审」「深基坑专项设计论证」标为行类=人工。
	serviceKeywords = []string{
		"编制", "咨询", "设计", "论证", "评估", "评审", "测绘", "勘察", "专家",
		"检测", "监测", "鉴别", "化验",
	}
)

// ClassifyKind 判定一条成本条目应归入工料机哪一类。
//
// 判定优先级（越靠前越权威）：
//  1. 顶层分类路径显式表态（人工/机械/材料）——路径是人工维护的强意图；
//  2. 外委关键词 → 外委（独立行类，三费归集时并入材料桶）；
//  3. 显式人工/机械关键词（含「工日」「台班」等强信号）；
//  4. 现场技术服务关键词 → 人工；
//  5. 兜底：材料（资源型条目里材料是绝大多数，且「未知即材料」比「未知即
//     人工」误判代价低——人工费被多算会直接抬高直接费）。
//
// 注意：本函数只看分类路径与标题，**不看金额、不看来源**——归类必须是
// 可复现的、与数据导入顺序无关的纯判定。
func ClassifyKind(categoryPath, title string) string {
	hay := strings.TrimSpace(categoryPath) + " " + strings.TrimSpace(title)

	// 1. 顶层路径显式表态，优先级最高。
	top := topSegment(categoryPath)
	switch top {
	case pathLabor:
		return KindLabor
	case pathMachinery:
		return KindMachine
	case pathMaterials:
		// 材料路径下仍可能是机械（「材料/辅助材料/机械配件」）或外委
		// （「材料/辅助材料/外委加工」）——继续往下看，未命中才按材料算。
		if containsAny(hay, machineKeywords) {
			return KindMachine
		}
		if containsAny(hay, outsourcedKeywords) {
			return KindOutsourced
		}
		return KindMaterial
	}

	// 2. 无顶层表态时按关键词。外委判定在材料路径之前也要生效——实测库里
	//    「其他/处置」「运输/场外运输」等路径下的委外项必须归外委。
	if containsAny(hay, outsourcedKeywords) {
		return KindOutsourced
	}
	if containsAny(hay, laborKeywords) {
		return KindLabor
	}
	if containsAny(hay, machineKeywords) {
		return KindMachine
	}
	if containsAny(hay, serviceKeywords) {
		return KindLabor
	}
	return KindMaterial
}

// topSegment 取分类路径首段（"材料/安装材料/管材管件" → "材料"）。
func topSegment(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	if i := strings.Index(p, "/"); i >= 0 {
		p = p[:i]
	}
	return strings.TrimSpace(p)
}

func containsAny(hay string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(hay, n) {
			return true
		}
	}
	return false
}

// SeedCandidate 一条待入库的资源候选（从成本条目派生，仍可人工复审）。
type SeedCandidate struct {
	// SourceName 来源成本条目的 name（回写 price_derived / 溯源用）。
	SourceName   string  `json:"sourceName"`
	Title        string  `json:"title"`
	Spec         string  `json:"spec"`
	Unit         string  `json:"unit"`
	Kind         string  `json:"kind"`
	Price        float64 `json:"price"`
	CategoryPath string  `json:"categoryPath"`
	Source       string  `json:"source"`
	Region       string  `json:"region"`
	PriceDate    string  `json:"priceDate"`
	PriceType    string  `json:"priceType"`
	// Note 保留原条目的 body 摘要（如「宝兴县旺平矿业地块成本测算表最终版」
	// 这类口径说明），它是资源的价源依据，不能丢。
	Note string `json:"note"`
}

// AsResource 把候选转成资源（基准价与现行价都取条目价：存量条目是**当期
// 已知价**，同时作为基准与现行没有语义损失；后续调价由信息价更新推进现行价）。
func (c SeedCandidate) AsResource() Resource {
	kind := c.Kind
	if kind == "" {
		kind = ClassifyKind(c.CategoryPath, c.Title)
	}
	return Resource{
		Kind:         kind,
		Title:        strings.TrimSpace(c.Title),
		Spec:         strings.TrimSpace(c.Spec),
		Unit:         strings.TrimSpace(c.Unit),
		BasePrice:    c.Price,
		CurrentPrice: c.Price,
		CategoryPath: c.CategoryPath,
		Source:       c.Source,
		Region:       c.Region,
		PriceDate:    c.PriceDate,
		PriceType:    c.PriceType,
		Note:         c.Note,
		Status:       "现行",
	}
}

// ClassifyCount 归类统计（供入库前的预览与回归断言）。
type ClassifyCount struct {
	Labor      int `json:"labor"`
	Material   int `json:"material"`
	Machine    int `json:"machine"`
	Outsourced int `json:"outsourced"`
}

// Total 四类合计。
func (c ClassifyCount) Total() int { return c.Labor + c.Material + c.Machine + c.Outsourced }

// CountKinds 统计一批候选的归类分布。
func CountKinds(list []SeedCandidate) ClassifyCount {
	var c ClassifyCount
	for _, it := range list {
		switch it.Kind {
		case KindLabor:
			c.Labor++
		case KindMachine:
			c.Machine++
		case KindOutsourced:
			c.Outsourced++
		default:
			c.Material++
		}
	}
	return c
}

// ── 资源化规则 ──────────────────────────────────────────────────────

// SeedOptions 资源化规则（哪些条目应进资源库）。
//
// 现状（实测）：库里约 1564/1590 条是纯资源价，26 条分类为「综合单价/…」的
// 是真·综合单价条目——后者不该被降级成资源（它有组成语义，应留在综合单价层，
// 组成从最近项目录入）。因此默认排除「综合单价」路径。
type SeedOptions struct {
	// IncludeComposite 是否把「综合单价/…」路径的条目也纳入资源库。
	// 默认 false（它们是综合单价，不是资源）。
	IncludeComposite bool
	// MinPrice 价格下限（>0 才入库；0 或负价条目无核算意义，跳过）。
	MinPrice float64
}

// DefaultSeedOptions 默认资源化规则。
func DefaultSeedOptions() SeedOptions {
	return SeedOptions{IncludeComposite: false, MinPrice: 0}
}

// SeedInput 一条成本条目的资源化输入（显式字段，避免 workcost 反向依赖 cost 包）。
type SeedInput struct {
	Name         string
	Title        string
	Spec         string
	Unit         string
	Price        float64
	CategoryPath string
	Source       string
	Region       string
	PriceDate    string
	PriceType    string
	Body         string
}

// IsCompositeEntry 判定是否为综合单价条目（分类路径含「综合单价」段）。
// 综合单价条目有组成语义，不应降级为单一资源。
func IsCompositeEntry(categoryPath string) bool {
	for _, seg := range strings.Split(categoryPath, "/") {
		if strings.TrimSpace(seg) == "综合单价" {
			return true
		}
	}
	return false
}

// ToSeedCandidate 把一条成本条目转成资源候选；返回 ok=false 表示按规则跳过。
// 跳过原因经 skipReasons 统计，调用方可在预览里显形「为什么少了几条」。
func ToSeedCandidate(in SeedInput, opt SeedOptions) (SeedCandidate, bool, string) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSpace(in.Name)
	}
	if title == "" {
		return SeedCandidate{}, false, "无名称"
	}
	if in.Price <= opt.MinPrice {
		return SeedCandidate{}, false, "价格非正"
	}
	if !opt.IncludeComposite && IsCompositeEntry(in.CategoryPath) {
		return SeedCandidate{}, false, "综合单价条目（保留在综合单价层）"
	}
	// spec 字段在实测库里被用作来源台账（如「宝兴县旺平矿业地块成本测算表最终版」），
	// 而不是技术规格。据此把「含项目名/测算/报价/最终版/成本表」等特征的 spec
	// 识别为口径说明 → 并入 Note，避免把台账串当成规格污染资源身份。
	spec := strings.TrimSpace(in.Spec)
	note := strings.TrimSpace(in.Body)
	if spec != "" && looksLikeLedger(spec) {
		if note != "" {
			note = spec + "；" + note
		} else {
			note = spec
		}
		spec = ""
	}

	return SeedCandidate{
		SourceName:   in.Name,
		Title:        title,
		Spec:         spec,
		Unit:         strings.TrimSpace(in.Unit),
		Kind:         ClassifyKind(in.CategoryPath, title),
		Price:        in.Price,
		CategoryPath: in.CategoryPath,
		Source:       in.Source,
		Region:       in.Region,
		PriceDate:    in.PriceDate,
		PriceType:    in.PriceType,
		Note:         note,
	}, true, ""
}

// looksLikeLedger 判定 spec 实为口径台账串而非技术规格。
var ledgerMarkers = []string{
	"成本测算", "测算表", "报价单", "最终版", "成本表", "信息价", "作废",
	"元/t", "元/m", "元/项", "元/座", "元-", "地块", "项目",
}

func looksLikeLedger(spec string) bool {
	return containsAny(spec, ledgerMarkers)
}

// BuildSeedCandidates 批量转换（纯函数）：返回候选与跳过原因统计。
func BuildSeedCandidates(inputs []SeedInput, opt SeedOptions) ([]SeedCandidate, map[string]int) {
	var out []SeedCandidate
	skipped := map[string]int{}
	for _, in := range inputs {
		c, ok, reason := ToSeedCandidate(in, opt)
		if !ok {
			skipped[reason]++
			continue
		}
		out = append(out, c)
	}
	return out, skipped
}

// SeedPreview 资源化干跑预览（入库前给用户看，不写库）。
type SeedPreview struct {
	Candidates []SeedCandidate `json:"candidates"`
	Counts     ClassifyCount   `json:"counts"`
	Skipped    map[string]int  `json:"skipped"`
	Total      int             `json:"total"`
}

// PreviewSeed 从条目输入构建干跑预览。
func PreviewSeed(inputs []SeedInput, opt SeedOptions) SeedPreview {
	cands, skipped := BuildSeedCandidates(inputs, opt)
	return SeedPreview{
		Candidates: cands,
		Counts:     CountKinds(cands),
		Skipped:    skipped,
		Total:      len(inputs),
	}
}

// SeedResult 资源化入库结果。
type SeedResult struct {
	Created int           `json:"created"`
	Updated int           `json:"updated"`
	Skipped int           `json:"skipped"`
	Counts  ClassifyCount `json:"counts"`
	Errors  []string      `json:"errors"`
}

// ApplySeed 把候选写入工料机资源库（同身份 UPDATE、新身份 INSERT，由
// SaveResource 的身份唯一索引保证幂等）。重复调用不会制造重复资源。
//
// 逐条容错：单条失败记入 Errors 而不中断整批——1564 条批量入库若因一条脏数据
// 整体回滚，用户无法知道卡在哪条。
func (s *Store) ApplySeed(cands []SeedCandidate) SeedResult {
	var res SeedResult
	if err := s.requireDB(); err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}
	for _, c := range cands {
		r := c.AsResource()
		// 入库前探测是否已存在同身份资源，以区分 created / updated 计数。
		var existed int
		_ = s.db.QueryRow(
			`SELECT COUNT(*) FROM gf_resources WHERE kind=? AND title=? AND spec=? AND unit=?`,
			r.Kind, r.Title, r.Spec, r.Unit).Scan(&existed)

		saved, err := s.SaveResource(r)
		if err != nil {
			res.Errors = append(res.Errors, c.Title+": "+err.Error())
			res.Skipped++
			continue
		}
		_ = saved
		if existed > 0 {
			res.Updated++
		} else {
			res.Created++
		}
		// 计数按**落库后**的归一类别（SaveResource 会 normalizeKind），
		// 否则「委外」这类写法会在统计里落进材料桶而报表显示外委。
		switch normalizeKind(r.Kind) {
		case KindLabor:
			res.Counts.Labor++
		case KindMachine:
			res.Counts.Machine++
		case KindOutsourced:
			res.Counts.Outsourced++
		default:
			res.Counts.Material++
		}
	}
	return res
}
