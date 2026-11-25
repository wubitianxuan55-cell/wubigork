package workcost

// importstage.go — 模型读表导入的暂存、勾稽与提交（2026-11-25 用户拍板：造价资料
// 格式不固定，不只是标准五表；导入由模型读资料、整理后整批提交。五表机械解析只留
// 作标准模版的快速通道，见 projectapply.go）。
//
// 一次导入 = 一个暂存会话（ImportSession）：
//
//	Open   登记来源与项目口径（专业/地区/计价期过规矩）与资料里的控制数；
//	Add    模型分批提交整理好的工料机 / 定额 / 清单行——每行可带来源位置（哪张表第
//	       几行）。程序当场按库的规矩校验：不合规的行退回并说明原因，合规的进暂存区；
//	Check  勾稽（只读）：引用完整 / 回读核对（程序回到文件里读那一行，提交的每个数
//	       必须在那行里出现——数字由模型搬运，这一步防抄错）/ 控制数 / 单价对照；
//	Commit 演练只出报告；正式写入（按编码/身份幂等，中断可重提）。
//
// 暂存只在内存里，库里一行不动；正式提交才写库。规矩（rules.go）在 Add 与 Commit
// 两道都把关——暂存区里的行也可能因为后来字典变了而不再合规。

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
)

// ImportMaxPerBatch import_add 单批每个数组最多多少行（模型一次整理得动的量）。
const ImportMaxPerBatch = 200

// ── 入参 ─────────────────────────────────────────────────────────────

// ImportSrc 一行数据在资料里的位置——回读核对的锚点。
type ImportSrc struct {
	Sheet string `json:"sheet"` // 工作表名（xlsx 必给；csv 留空）
	Row   int    `json:"row"`   // 行号（从 1 起，与 Excel 一致）
}

func (l *ImportSrc) label() string {
	if l == nil {
		return ""
	}
	if l.Sheet == "" {
		return fmt.Sprintf("第 %d 行", l.Row)
	}
	return fmt.Sprintf("%s 第 %d 行", l.Sheet, l.Row)
}

// ImportResource 模型整理好的一条工料机。
type ImportResource struct {
	Code         string     `json:"code"` // 可空（提交时自动分配）；被定额含量行引用的必须给
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	Spec         string     `json:"spec"`
	Unit         string     `json:"unit"`
	Price        *float64   `json:"price"`         // 单价：同时作基准价与现行价
	BasePrice    *float64   `json:"base_price"`    // 单独给时优先于 price
	CurrentPrice *float64   `json:"current_price"` // 单独给时优先于 price
	CategoryPath string     `json:"category_path"` // 字典值；空=待整理
	Region       string     `json:"region"`        // 不给=项目所在地区
	Period       string     `json:"period"`        // 现行价的期；不给=项目计价期
	PriceType    string     `json:"price_type"`
	Source       string     `json:"source"`
	Note         string     `json:"note"`
	Src          *ImportSrc `json:"src"`
	Derived      []string   `json:"derived"` // 由模型换算（不是照抄）的字段名，回读时不核
}

// ImportQuotaItem 模型整理好的一条定额含量行。
type ImportQuotaItem struct {
	ResourceCode string     `json:"resource_code"`
	Quantity     float64    `json:"quantity"`
	LossRate     float64    `json:"loss_rate"`
	Note         string     `json:"note"`
	Src          *ImportSrc `json:"src"` // 明细行的来源位置；不给=这一行不回读
}

// ImportQuota 模型整理好的一条消耗定额。
type ImportQuota struct {
	Code         string            `json:"code"`
	Title        string            `json:"title"`
	Unit         string            `json:"unit"`
	Specialty    string            `json:"specialty"` // 不给=项目专业
	Chapter      string            `json:"chapter"`
	CategoryPath string            `json:"category_path"`
	Source       string            `json:"source"`
	Note         string            `json:"note"`
	Items        []ImportQuotaItem `json:"items"`
	Src          *ImportSrc        `json:"src"`
	Derived      []string          `json:"derived"`
}

// ImportBill 模型整理好的一条清单行。
type ImportBill struct {
	Code          string     `json:"code"` // 空=自动 M 序号
	Title         string     `json:"title"`
	Unit          string     `json:"unit"`
	Division      string     `json:"division"`
	Specialty     string     `json:"specialty"` // 不给=跟随项目专业
	Quantity      float64    `json:"quantity"`
	QuantityExpr  string     `json:"quantity_expr"`
	QuotaCode     string     `json:"quota_code"`
	QuotaCodes    []string   `json:"quota_codes"`
	Feature       string     `json:"feature"`
	PriceOverride float64    `json:"price_override"`
	ListPrice     float64    `json:"list_price"` // 原表综合单价（单价对照用）
	Src           *ImportSrc `json:"src"`
	Derived       []string   `json:"derived"`
}

// links 该行挂接的定额码（主定额在前，其余保序；空码与重复码剔除）。
func (in ImportBill) links() []BillQuotaLinkInput {
	var out []BillQuotaLinkInput
	seen := map[string]bool{}
	add := func(c string) {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			return
		}
		seen[c] = true
		out = append(out, BillQuotaLinkInput{QuotaCode: c})
	}
	add(in.QuotaCode)
	for _, c := range in.QuotaCodes {
		add(c)
	}
	return out
}

// ImportProject 导入的项目口径与封面。
type ImportProject struct {
	Name                     string   `json:"name"`
	FileName                 string   `json:"file_name"`
	Source                   string   `json:"source"`
	Location                 string   `json:"location"`
	Region                   string   `json:"region"`
	Specialty                string   `json:"specialty"`
	PricePeriod              string   `json:"price_period"`
	Duration                 string   `json:"duration"`
	Pricing                  string   `json:"pricing"`
	ManagementRate           *float64 `json:"management_rate"`
	RegulatoryRate           *float64 `json:"regulatory_rate"`
	ProfitRate               *float64 `json:"profit_rate"`
	TaxRate                  *float64 `json:"tax_rate"`
	ProfitIncludesRegulatory *bool    `json:"profit_includes_regulatory"`
	ControlPrice             *float64 `json:"control_price"`
}

func (p ImportProject) anyRate() bool {
	return p.ManagementRate != nil || p.RegulatoryRate != nil || p.ProfitRate != nil || p.TaxRate != nil
}

func (p ImportProject) rates() *RateSet {
	if !p.anyRate() {
		return nil // 一个费率都没给：同名项目不动已存费率，新项目落显式全零
	}
	rs := &RateSet{}
	if p.ManagementRate != nil {
		rs.ManagementRate = *p.ManagementRate
	}
	if p.RegulatoryRate != nil {
		rs.RegulatoryRate = *p.RegulatoryRate
	}
	if p.ProfitRate != nil {
		rs.ProfitRate = *p.ProfitRate
	}
	if p.TaxRate != nil {
		rs.TaxRate = *p.TaxRate
	}
	return rs
}

// ImportControls 资料里的控制数——模型从资料上读到的「总量」，程序拿来对暂存数据。
// 都是可选的；给了才比。
type ImportControls struct {
	Resources *int     `json:"resources"`          // 工料机价格表的行数
	Quotas    *int     `json:"quotas"`             // 定额子目（综合单价分析）条数
	Bills     *int     `json:"bills"`              // 清单行数
	BillTotal *float64 `json:"bill_total"`         // 清单合计 = Σ(工程量 × 原表综合单价)
	PriceSum  *float64 `json:"resource_price_sum"` // 工料机单价合计
}

// ── 会话 ─────────────────────────────────────────────────────────────

// ImportSession 一次导入的暂存会话。只在内存里；Open 后由工具层按 import_id 保管。
type ImportSession struct {
	ID         string
	SourcePath string // 资料的绝对路径；只为回读核对用，不解析它的结构
	Project    ImportProject
	Controls   ImportControls
	// 默认口径（已过规矩并归一）：行里没给时用它。
	Specialty string
	Region    string
	Period    string

	resources []ImportResource
	quotas    []ImportQuota
	bills     []ImportBill
	resIdx    map[string]int
	quotaIdx  map[string]int
	billIdx   map[string]int
	source    *importSource
}

var importSeq atomic.Int64

// OpenImport 开一次导入：登记来源与项目口径、控制数。口径过规矩（专业、地区在字典里，
// 计价期 YYYY-MM），不过规矩当场拒。项目可以不给（只导工料机/定额）——清单行提交时
// 再要求。
func (s *Store) OpenImport(sourcePath string, project ImportProject, controls ImportControls) (*ImportSession, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	period, err := s.ValidateProjectScope(project.Specialty, project.Region, project.PricePeriod)
	if err != nil {
		return nil, fmt.Errorf("导入口径不合规矩：%w", err)
	}
	project.Name = strings.TrimSpace(project.Name)
	project.Region = strings.TrimSpace(project.Region)
	project.Specialty = strings.TrimSpace(project.Specialty)
	return &ImportSession{
		ID: fmt.Sprintf("imp-%d", importSeq.Add(1)), SourcePath: strings.TrimSpace(sourcePath),
		Project: project, Controls: controls,
		Specialty: project.Specialty, Region: project.Region, Period: period,
		resIdx: map[string]int{}, quotaIdx: map[string]int{}, billIdx: map[string]int{},
	}, nil
}

// Close 释放会话占着的资源文件句柄。
func (imp *ImportSession) Close() {
	if imp != nil && imp.source != nil {
		imp.source.Close()
		imp.source = nil
	}
}

// Staged 暂存区三类行的条数。
func (imp *ImportSession) Staged() (resources, quotas, bills int) {
	return len(imp.resources), len(imp.quotas), len(imp.bills)
}

// defaultSource 行里没给来源时的缺省值（溯源到资料文件）。
func (imp *ImportSession) defaultSource() string {
	if imp.SourcePath == "" {
		return "导入"
	}
	return "导入：" + filepath.Base(imp.SourcePath)
}

func (imp *ImportSession) hasQuotaCode(s *Store, code string) bool {
	if _, ok := imp.quotaIdx[code]; ok {
		return true
	}
	_, err := s.GetQuota(code)
	return err == nil
}

func (imp *ImportSession) hasResourceCode(s *Store, code string) bool {
	if _, ok := imp.resIdx[code]; ok {
		return true
	}
	_, err := s.GetResourceByCode(code)
	return err == nil
}

func isDerived(derived []string, field string) bool {
	for _, d := range derived {
		if strings.TrimSpace(d) == field {
			return true
		}
	}
	return false
}

// ── 入暂存 ───────────────────────────────────────────────────────────

// ImportReject 一条被退回的行及原因。
type ImportReject struct {
	Index   int      `json:"index"` // 本批里的序号（从 1 起）
	Code    string   `json:"code"`
	Title   string   `json:"title"`
	Reasons []string `json:"reasons"`
}

// ImportAddResult 一批提交的结果。
type ImportAddResult struct {
	Kind     string         `json:"kind"` // resource / quota / bill
	Accepted int            `json:"accepted"`
	Replaced int            `json:"replaced"` // 覆盖了暂存区里同码的行
	Rejected []ImportReject `json:"rejected"`
	// 暂存区合计（三类）。
	StagedResources int `json:"stagedResources"`
	StagedQuotas    int `json:"stagedQuotas"`
	StagedBills     int `json:"stagedBills"`
}

func (imp *ImportSession) addResult(kind string) ImportAddResult {
	return ImportAddResult{Kind: kind, Rejected: []ImportReject{}}
}

func (imp *ImportSession) finish(r ImportAddResult) ImportAddResult {
	r.StagedResources, r.StagedQuotas, r.StagedBills = len(imp.resources), len(imp.quotas), len(imp.bills)
	return r
}

// resourceOf 把一行工料机整理成库里的形态并过规矩校验（地区、期数默认取项目口径）。
func (imp *ImportSession) resourceOf(s *Store, in ImportResource) (*Resource, error) {
	r := Resource{
		Code:         strings.TrimSpace(in.Code),
		Kind:         strings.TrimSpace(in.Kind),
		Title:        strings.TrimSpace(in.Title),
		Spec:         strings.TrimSpace(in.Spec),
		Unit:         strings.TrimSpace(in.Unit),
		Region:       firstNonEmpty(strings.TrimSpace(in.Region), imp.Region),
		CategoryPath: strings.TrimSpace(in.CategoryPath),
		PriceType:    strings.TrimSpace(in.PriceType),
		Source:       firstNonEmpty(strings.TrimSpace(in.Source), imp.defaultSource()),
		Note:         strings.TrimSpace(in.Note),
	}
	switch {
	case in.CurrentPrice != nil:
		r.CurrentPrice = *in.CurrentPrice
	case in.Price != nil:
		r.CurrentPrice = *in.Price
	}
	switch {
	case in.BasePrice != nil:
		r.BasePrice = *in.BasePrice
	case in.Price != nil:
		r.BasePrice = *in.Price
	}
	if r.BasePrice == 0 && r.CurrentPrice > 0 {
		r.BasePrice = r.CurrentPrice
	}
	if r.BasePrice < 0 || r.CurrentPrice < 0 {
		return nil, fmt.Errorf("价格不能为负（基准价 %v / 现行价 %v）", r.BasePrice, r.CurrentPrice)
	}
	if r.EffectivePrice() > 0 {
		r.PriceDate = firstNonEmpty(strings.TrimSpace(in.Period), imp.Period)
	}
	if err := s.ValidateResource(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// AddResources 工料机分批入暂存。每行按库的规矩校验（类别四选一、地区/分类在字典里、
// 期数 YYYY-MM、编码像编码、码不被别的身份占用）；合规的进暂存区，同码后者覆盖。
func (imp *ImportSession) AddResources(s *Store, rows []ImportResource) (ImportAddResult, error) {
	res := imp.addResult("resource")
	for i, in := range rows {
		in.Code, in.Title = strings.TrimSpace(in.Code), strings.TrimSpace(in.Title)
		var reasons []string
		r, err := imp.resourceOf(s, in)
		switch {
		case err != nil:
			reasons = append(reasons, err.Error())
		default:
			if in.Code != "" {
				if ok, why := CodeWellFormed(in.Code); !ok {
					reasons = append(reasons, why)
				}
				if exist, err := s.GetResourceByCode(in.Code); err == nil && exist != nil && !sameResourceIdentity(*exist, *r) {
					reasons = append(reasons, fmt.Sprintf("编码 %s 已被「%s」占用，不是同一条工料机", in.Code, exist.Title))
				}
			}
			// 同身份已有行、入参却带了另一个码：库里的码是引用锚点，不换。
			if prev, err := s.findResourceByIdentity(*r); err == nil && prev != nil && in.Code != "" && prev.Code != in.Code {
				reasons = append(reasons, fmt.Sprintf("同身份的工料机已在库（编码 %s）——把资源编码改成 %s 再提交", prev.Code, prev.Code))
			}
		}
		if len(reasons) > 0 {
			res.Rejected = append(res.Rejected, ImportReject{Index: i + 1, Code: in.Code, Title: in.Title, Reasons: reasons})
			continue
		}
		if in.Code != "" {
			if idx, ok := imp.resIdx[in.Code]; ok {
				imp.resources[idx] = in
				res.Replaced++
				continue
			}
			imp.resIdx[in.Code] = len(imp.resources)
		}
		imp.resources = append(imp.resources, in)
		res.Accepted++
	}
	return imp.finish(res), nil
}

// quotaOf 把一条定额整理成库里的形态，逐项校验，收齐全部问题（不只报第一个）。
func (imp *ImportSession) quotaOf(s *Store, in ImportQuota) (*Quota, []string) {
	q := Quota{
		Code:         strings.TrimSpace(in.Code),
		Title:        strings.TrimSpace(in.Title),
		Unit:         strings.TrimSpace(in.Unit),
		Specialty:    firstNonEmpty(strings.TrimSpace(in.Specialty), imp.Specialty),
		Chapter:      strings.TrimSpace(in.Chapter),
		CategoryPath: strings.TrimSpace(in.CategoryPath),
		Source:       firstNonEmpty(strings.TrimSpace(in.Source), imp.defaultSource()),
		Note:         strings.TrimSpace(in.Note),
	}
	var reasons []string
	if err := s.ValidateQuota(&q); err != nil {
		reasons = append(reasons, err.Error())
	}
	for k, it := range in.Items {
		rc := strings.TrimSpace(it.ResourceCode)
		switch {
		case rc == "":
			reasons = append(reasons, fmt.Sprintf("第 %d 行含量行缺 resource_code", k+1))
		case it.Quantity < 0:
			reasons = append(reasons, fmt.Sprintf("第 %d 行含量为负数", k+1))
		case it.LossRate < 0 || it.LossRate >= 1:
			reasons = append(reasons, fmt.Sprintf("第 %d 行损耗率 %v 不在 [0,1) 区间（0.03=3%%）", k+1, it.LossRate))
		case !imp.hasResourceCode(s, rc):
			reasons = append(reasons, fmt.Sprintf("第 %d 行含量行引用的工料机 %s 不在库也不在暂存区——先把工料机提交进暂存区", k+1, rc))
		}
		q.Items = append(q.Items, QuotaItem{ResourceCode: rc, Quantity: it.Quantity, LossRate: it.LossRate, Note: strings.TrimSpace(it.Note)})
	}
	return &q, reasons
}

// AddQuotas 定额分批入暂存。库里的同码定额若已是别的专业则退回（定额专业固定）；
// 同码同专业=提交时更新。含量行引用的工料机必须已在库或在暂存区。
func (imp *ImportSession) AddQuotas(s *Store, rows []ImportQuota) (ImportAddResult, error) {
	res := imp.addResult("quota")
	for i, in := range rows {
		in.Code, in.Title = strings.TrimSpace(in.Code), strings.TrimSpace(in.Title)
		var reasons []string
		if ok, why := CodeWellFormed(in.Code); !ok {
			reasons = append(reasons, why)
		}
		q, problems := imp.quotaOf(s, in)
		reasons = append(reasons, problems...)
		if len(problems) == 0 && in.Code != "" {
			if exist, err := s.GetQuota(in.Code); err == nil && strings.TrimSpace(exist.Specialty) != "" &&
				strings.TrimSpace(exist.Specialty) != strings.TrimSpace(q.Specialty) {
				reasons = append(reasons, fmt.Sprintf("定额 %s 已是「%s」专业，专业固定不改——本专业（%s）的请另起编码；借鉴生成走 borrow_quota",
					in.Code, exist.Specialty, q.Specialty))
			}
		}
		if len(reasons) > 0 {
			res.Rejected = append(res.Rejected, ImportReject{Index: i + 1, Code: in.Code, Title: in.Title, Reasons: reasons})
			continue
		}
		if idx, ok := imp.quotaIdx[in.Code]; ok {
			imp.quotas[idx] = in
			res.Replaced++
		} else {
			imp.quotaIdx[in.Code] = len(imp.quotas)
			imp.quotas = append(imp.quotas, in)
			res.Accepted++
		}
	}
	return imp.finish(res), nil
}

// billOf 把一条清单行整理成库里的形态并校验，收齐全部问题。
func (imp *ImportSession) billOf(s *Store, in ImportBill) (*BillItem, []BillQuotaLinkInput, []string) {
	b := BillItem{
		Code:          strings.TrimSpace(in.Code),
		Title:         strings.TrimSpace(in.Title),
		Unit:          strings.TrimSpace(in.Unit),
		Division:      strings.TrimSpace(in.Division),
		Specialty:     strings.TrimSpace(in.Specialty),
		Quantity:      in.Quantity,
		QuantityExpr:  strings.TrimSpace(in.QuantityExpr),
		Feature:       strings.TrimSpace(in.Feature),
		PriceOverride: in.PriceOverride,
		ListPrice:     in.ListPrice,
	}
	var reasons []string
	if b.Title == "" || b.Unit == "" {
		reasons = append(reasons, "缺 title/unit")
	}
	if in.Quantity <= 0 {
		reasons = append(reasons, fmt.Sprintf("工程量 %g 必须>0", in.Quantity))
	}
	if in.PriceOverride < 0 || in.ListPrice < 0 {
		reasons = append(reasons, "手填单价与原表综合单价不能为负")
	}
	if b.Code != "" {
		if ok, why := CodeWellFormed(b.Code); !ok {
			reasons = append(reasons, why)
		}
	}
	if err := s.checkDict(DictSpecialty, b.Specialty, "清单项专业"); err != nil {
		reasons = append(reasons, err.Error())
	}
	links := in.links()
	for _, l := range links {
		if !imp.hasQuotaCode(s, l.QuotaCode) {
			reasons = append(reasons, fmt.Sprintf("挂接的定额 %s 不在库也不在暂存区——先把定额提交进暂存区", l.QuotaCode))
		}
	}
	if len(links) > 0 {
		b.QuotaCode = links[0].QuotaCode
	}
	return &b, links, reasons
}

// AddBills 清单行分批入暂存。挂接的定额必须已在库或在暂存区；同码后者覆盖。
func (imp *ImportSession) AddBills(s *Store, rows []ImportBill) (ImportAddResult, error) {
	res := imp.addResult("bill")
	for i, in := range rows {
		in.Code = strings.TrimSpace(in.Code)
		b, _, reasons := imp.billOf(s, in)
		if len(reasons) > 0 {
			res.Rejected = append(res.Rejected, ImportReject{Index: i + 1, Code: in.Code, Title: b.Title, Reasons: reasons})
			continue
		}
		if in.Code != "" {
			if idx, ok := imp.billIdx[in.Code]; ok {
				imp.bills[idx] = in
				res.Replaced++
				continue
			}
			imp.billIdx[in.Code] = len(imp.bills)
		}
		imp.bills = append(imp.bills, in)
		res.Accepted++
	}
	return imp.finish(res), nil
}

// ── 勾稽 ─────────────────────────────────────────────────────────────

// ImportControlCheck 一项控制数的比对。
type ImportControlCheck struct {
	Label    string `json:"label"`
	Declared string `json:"declared"`
	Staged   string `json:"staged"`
	OK       bool   `json:"ok"`
}

// ImportReadbackIssue 一处回读不符。
type ImportReadbackIssue struct {
	Row   string `json:"row"`   // 「工料机 C20」/「定额 WP01 含量行 2」/「清单 WP03」
	Where string `json:"where"` // 来源位置
	Field string `json:"field"`
	Gave  string `json:"gave"`
	Found string `json:"found"` // 该行里实际有的数（帮助定位抄错）
}

// ImportPriceCheck 一条单价对照（原表综合单价 vs 现算）。
type ImportPriceCheck struct {
	Code         string  `json:"code"`
	Title        string  `json:"title"`
	ListPrice    float64 `json:"listPrice"`
	Computed     float64 `json:"computed"`
	DeviationPct float64 `json:"deviationPct"`
}

// ImportReport 勾稽报告。
type ImportReport struct {
	StagedResources int `json:"stagedResources"`
	StagedQuotas    int `json:"stagedQuotas"`
	StagedBills     int `json:"stagedBills"`

	Controls []ImportControlCheck `json:"controls"`
	// Dangling 引用悬空（应为空：入暂存时已拦；这里复查，防暂存区后来被改）。
	Dangling []string `json:"dangling"`

	// 回读核对。
	// NoSource 这次导入没有可回读的资料（没登记路径 / 格式不支持）——所有行都只能是未核对。
	NoSource       bool                  `json:"noSource"`
	ReadbackRows   int                   `json:"readbackRows"`   // 带来源位置、参与回读的行数
	ReadbackFields int                   `json:"readbackFields"` // 核对的字段数
	UnverifiedRows int                   `json:"unverifiedRows"` // 没核对上的行数（没带来源位置，或没有可回读的资料）
	DerivedFields  int                   `json:"derivedFields"`  // 模型换算、不核的字段数
	ReadbackIssues []ImportReadbackIssue `json:"readbackIssues"`

	// 单价对照（清单有原表综合单价且挂了定额的）。
	PriceChecked int                `json:"priceChecked"`
	PriceIssues  []ImportPriceCheck `json:"priceIssues"`

	// Problems 需要处理的总数 = 控制数不符 + 悬空 + 回读不符 + 单价偏差。
	Problems int `json:"problems"`
}

// 偏差超过这个比例（且绝对差超过半元）才算单价对不上——原表与本库资源价本来
// 就可能有零头差异。
const (
	importPriceDeviationPct = 1.0
	importPriceDeviationAbs = 0.5
)

// Check 勾稽（只读，不写库）。
func (imp *ImportSession) Check(s *Store) (*ImportReport, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rep := &ImportReport{
		Controls: []ImportControlCheck{}, Dangling: []string{}, ReadbackIssues: []ImportReadbackIssue{}, PriceIssues: []ImportPriceCheck{},
	}
	rep.StagedResources, rep.StagedQuotas, rep.StagedBills = imp.Staged()

	// ① 控制数：资料里的总量对暂存数据。
	intControl := func(label string, declared *int, staged int) {
		if declared == nil {
			return
		}
		c := ImportControlCheck{Label: label, Declared: strconv.Itoa(*declared), Staged: strconv.Itoa(staged)}
		c.OK = *declared == staged
		rep.Controls = append(rep.Controls, c)
	}
	intControl("工料机行数", imp.Controls.Resources, rep.StagedResources)
	intControl("定额条数", imp.Controls.Quotas, rep.StagedQuotas)
	intControl("清单行数", imp.Controls.Bills, rep.StagedBills)
	priceSum, billTotal := 0.0, 0.0
	for _, r := range imp.resources {
		if rr, err := imp.resourceOf(s, r); err == nil {
			priceSum += rr.CurrentPrice
		}
	}
	for _, b := range imp.bills {
		if b.ListPrice > 0 {
			billTotal += b.Quantity * b.ListPrice
		}
	}
	floatControl := func(label string, declared *float64, staged float64) {
		if declared == nil {
			return
		}
		c := ImportControlCheck{Label: label,
			Declared: strconv.FormatFloat(*declared, 'f', 2, 64), Staged: strconv.FormatFloat(staged, 'f', 2, 64)}
		// 合计类控制数：相对误差 0.5% 以内算对上（资料里可能有四舍五入与个别项不计）。
		c.OK = math.Abs(*declared-staged) <= math.Abs(*declared)*0.005+0.5
		rep.Controls = append(rep.Controls, c)
	}
	floatControl("工料机单价合计", imp.Controls.PriceSum, priceSum)
	floatControl("清单合计（Σ 工程量 × 原表综合单价）", imp.Controls.BillTotal, billTotal)

	// ② 引用完整（复查）。
	for _, q := range imp.quotas {
		for k, it := range q.Items {
			if rc := strings.TrimSpace(it.ResourceCode); rc != "" && !imp.hasResourceCode(s, rc) {
				rep.Dangling = append(rep.Dangling, fmt.Sprintf("定额 %s 第 %d 行含量行引用的工料机 %s 不存在", q.Code, k+1, rc))
			}
		}
	}
	for _, b := range imp.bills {
		for _, l := range b.links() {
			if !imp.hasQuotaCode(s, l.QuotaCode) {
				rep.Dangling = append(rep.Dangling, fmt.Sprintf("清单 %s 挂接的定额 %s 不存在", b.Code, l.QuotaCode))
			}
		}
	}

	// ③ 回读核对。
	unverified := 0
	// 没有可回读的资料（没登记路径 / 格式不支持）：所有行只能是未核对，不算问题——
	// 把「核不了」报成「不符」会掩盖真正的抄错。
	rep.NoSource = !imp.sourceReadable()
	verify := func(src *ImportSrc, row string, texts map[string]string, nums map[string]float64, derived []string) {
		if src == nil || rep.NoSource {
			unverified++
			return
		}
		fields, issues := imp.verifyRow(src, row, texts, nums, derived)
		rep.ReadbackRows++
		rep.ReadbackFields += fields
		rep.DerivedFields += importDerivedCount(derived)
		rep.ReadbackIssues = append(rep.ReadbackIssues, issues...)
	}
	for _, r := range imp.resources {
		texts := map[string]string{"编码": r.Code, "名称": r.Title, "规格": r.Spec, "单位": r.Unit}
		nums := map[string]float64{}
		if r.Price != nil {
			nums["单价"] = *r.Price
		}
		if r.BasePrice != nil && r.BasePrice != r.Price {
			nums["基准价"] = *r.BasePrice
		}
		if r.CurrentPrice != nil && r.CurrentPrice != r.Price {
			nums["现行价"] = *r.CurrentPrice
		}
		verify(r.Src, "工料机 "+r.Code, texts, nums, r.Derived)
	}
	for _, q := range imp.quotas {
		verify(q.Src, "定额 "+q.Code, map[string]string{"编码": q.Code, "名称": q.Title, "单位": q.Unit}, nil, q.Derived)
		for k, it := range q.Items {
			nums := map[string]float64{"含量": it.Quantity}
			if it.LossRate > 0 {
				nums["损耗率"] = it.LossRate
			}
			verify(it.Src, fmt.Sprintf("定额 %s 第 %d 行含量行", q.Code, k+1), map[string]string{"资源编码": it.ResourceCode}, nums, nil)
		}
	}
	for _, b := range imp.bills {
		texts := map[string]string{}
		if b.Code != "" {
			texts["编码"] = b.Code
		}
		texts["名称"], texts["单位"] = strings.TrimSpace(b.Title), strings.TrimSpace(b.Unit)
		if strings.TrimSpace(b.Division) != "" {
			texts["分部"] = b.Division
		}
		if strings.TrimSpace(b.Feature) != "" {
			texts["项目特征"] = b.Feature
		}
		nums := map[string]float64{"工程量": b.Quantity}
		if b.ListPrice > 0 {
			nums["原表综合单价"] = b.ListPrice
		}
		if b.PriceOverride > 0 {
			nums["手填单价"] = b.PriceOverride
		}
		verify(b.Src, "清单 "+b.Code, texts, nums, b.Derived)
	}
	rep.UnverifiedRows = unverified

	// ④ 单价对照：清单有原表综合单价、又挂了定额（暂存或库）的，现算对原表。
	for _, b := range imp.bills {
		links := b.links()
		if b.ListPrice <= 0 || len(links) == 0 {
			continue
		}
		computed, ok := 0.0, true
		for _, l := range links {
			p, resolved := imp.quotaPrice(s, l.QuotaCode)
			if !resolved {
				ok = false
				break
			}
			computed += p
		}
		if !ok {
			continue
		}
		rep.PriceChecked++
		dev := (computed - b.ListPrice) / b.ListPrice * 100
		if math.Abs(dev) > importPriceDeviationPct && math.Abs(computed-b.ListPrice) > importPriceDeviationAbs {
			rep.PriceIssues = append(rep.PriceIssues, ImportPriceCheck{Code: b.Code, Title: strings.TrimSpace(b.Title),
				ListPrice: b.ListPrice, Computed: computed, DeviationPct: dev})
		}
	}

	for _, c := range rep.Controls {
		if !c.OK {
			rep.Problems++
		}
	}
	rep.Problems += len(rep.Dangling) + len(rep.ReadbackIssues) + len(rep.PriceIssues)
	return rep, nil
}

// quotaPrice 一条定额的综合单价（人材机）：暂存区的按暂存含量行 × 资源价现算，库里的
// 走 ComposeQuota。定额不存在时 ok=false。
func (imp *ImportSession) quotaPrice(s *Store, code string) (float64, bool) {
	if idx, ok := imp.quotaIdx[code]; ok {
		total := 0.0
		for _, it := range imp.quotas[idx].Items {
			if p, ok := imp.resourcePrice(s, strings.TrimSpace(it.ResourceCode)); ok {
				total += it.Quantity * (1 + it.LossRate) * p
			}
		}
		return total, true
	}
	if c, _, err := s.ComposeQuota(code, nil); err == nil && c != nil {
		return c.CompositePrice, true
	}
	return 0, false
}

// resourcePrice 一条工料机的取用价：暂存区优先，其次库；都没有 ok=false（按 0 计）。
func (imp *ImportSession) resourcePrice(s *Store, code string) (float64, bool) {
	if idx, ok := imp.resIdx[code]; ok {
		if r, err := imp.resourceOf(s, imp.resources[idx]); err == nil {
			return r.EffectivePrice(), true
		}
		return 0, false
	}
	if r, err := s.GetResourceByCode(code); err == nil && r != nil {
		return r.EffectivePrice(), true
	}
	return 0, false
}

// ── 回读核对 ─────────────────────────────────────────────────────────

// importCell 资料里的一个单元格：数值型记数值，其余记原文。
type importCell struct {
	Num   float64
	IsNum bool
	Text  string
}

// importSource 懒加载的资料读取器（只读指定行，不解析结构）。
type importSource struct {
	kind   string // xlsx / csv
	file   *excelize.File
	sheets map[string][][]string
	rows   [][]string
}

func openImportSource(path string) (*importSource, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx", ".xlsm":
		f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
		if err != nil {
			return nil, fmt.Errorf("打开资料失败: %w", err)
		}
		return &importSource{kind: "xlsx", file: f, sheets: map[string][][]string{}}, nil
	case ".csv":
		fp, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("打开资料失败: %w", err)
		}
		defer func() { _ = fp.Close() }()
		r := csv.NewReader(fp)
		r.LazyQuotes = true
		r.FieldsPerRecord = -1
		var rows [][]string
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("读资料失败: %w", err)
			}
			rows = append(rows, rec)
		}
		if len(rows) > 0 && len(rows[0]) > 0 {
			rows[0][0] = strings.TrimPrefix(rows[0][0], "\uFEFF") // 去掉 UTF-8 BOM
		}
		return &importSource{kind: "csv", rows: rows}, nil
	default:
		return nil, fmt.Errorf("资料 %s 不是 xlsx / csv，无法回读核对（行里可以不带 src，按「未核对」计）", ext)
	}
}

func (is *importSource) Close() {
	if is != nil && is.file != nil {
		_ = is.file.Close()
	}
}

// read 读一行（行号从 1 起）。找不到那张表或那一行时 ok=false。
func (is *importSource) read(loc ImportSrc) ([]importCell, string, bool, error) {
	var raw []string
	switch is.kind {
	case "xlsx":
		if loc.Sheet == "" {
			return nil, "", false, fmt.Errorf("xlsx 的来源位置要给工作表名 src.sheet")
		}
		if index, _ := is.file.GetSheetIndex(loc.Sheet); index < 0 {
			// 没有这张表：与行号越界同样按「找不到来源行」处理（表名写错是常见错）。
			return nil, "", false, nil
		}
		rows, ok := is.sheets[loc.Sheet]
		if !ok {
			got, err := is.file.GetRows(loc.Sheet, excelize.Options{RawCellValue: true})
			if err != nil {
				return nil, "", false, fmt.Errorf("读工作表「%s」失败: %v", loc.Sheet, err)
			}
			is.sheets[loc.Sheet] = got
			rows = got
		}
		if loc.Row < 1 || loc.Row > len(rows) {
			return nil, "", false, nil
		}
		raw = rows[loc.Row-1]
	case "csv":
		if loc.Row < 1 || loc.Row > len(is.rows) {
			return nil, "", false, nil
		}
		raw = is.rows[loc.Row-1]
	}
	cells := make([]importCell, 0, len(raw))
	var text strings.Builder
	for _, v := range raw {
		c := importCell{Text: strings.TrimSpace(v)}
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			c.Num, c.IsNum = n, true
		}
		cells = append(cells, c)
		if c.Text != "" {
			text.WriteString(c.Text)
			text.WriteString(" ")
		}
	}
	return cells, text.String(), true, nil
}

// verifyRow 回读核对一行：提交的文本字段必须出现在该行里，数值字段必须是该行某个
// 单元格的值（百分数按 0.03 ↔ 3 互认）。derived 里的字段跳过。返回核了的字段数与
// 不符清单。
func (imp *ImportSession) verifyRow(src *ImportSrc, row string, texts map[string]string, nums map[string]float64, derived []string) (int, []ImportReadbackIssue) {
	cells, text, ok, err := imp.readSource(*src)
	if err != nil || !ok {
		why := "找不到来源行（表名或行号不对）"
		if err != nil {
			why = err.Error()
		}
		return 0, []ImportReadbackIssue{{Row: row, Where: src.label(), Field: "来源位置", Gave: "", Found: why}}
	}
	var issues []ImportReadbackIssue
	count := 0
	for field, gave := range texts {
		if strings.TrimSpace(gave) == "" {
			continue
		}
		if isDerived(derived, field) {
			continue
		}
		count++
		if !importTextIn(gave, text, cells) {
			issues = append(issues, ImportReadbackIssue{Row: row, Where: src.label(), Field: field, Gave: gave, Found: "该行里没有这段文字"})
		}
	}
	for field, gave := range nums {
		if isDerived(derived, field) {
			continue
		}
		count++
		if !importNumIn(gave, cells) {
			issues = append(issues, ImportReadbackIssue{Row: row, Where: src.label(), Field: field, Gave: strconv.FormatFloat(gave, 'f', -1, 64), Found: "该行里的数：" + importNums(cells)})
		}
	}
	return count, issues
}

// importTextIn 提交的文字是否出现在该行：忽略空白与大小写；纯数字（编码）容忍首零丢失。
func importTextIn(gave, text string, cells []importCell) bool {
	g := importNormText(gave)
	if g == "" {
		return true
	}
	if strings.Contains(importNormText(text), g) {
		return true
	}
	if importAllDigits(g) {
		stripped := strings.TrimLeft(g, "0")
		for _, c := range cells {
			cv := c.Text
			if c.IsNum {
				cv = strconv.FormatFloat(c.Num, 'f', -1, 64)
			}
			for _, tok := range strings.FieldsFunc(cv, func(r rune) bool { return !importIsDigit(r) }) {
				if tok != "" && strings.TrimLeft(tok, "0") == stripped {
					return true
				}
			}
		}
	}
	return false
}

// importNumIn 提交的数是否是该行某个单元格的值。
func importNumIn(gave float64, cells []importCell) bool {
	for _, c := range cells {
		if !c.IsNum {
			if p, ok := importParsePercent(c.Text); ok && importCloseNum(gave, p) {
				return true
			}
			continue
		}
		if importCloseNum(gave, c.Num) || importCloseNum(gave, c.Num*100) || (c.Num != 0 && importCloseNum(gave, c.Num/100)) {
			return true
		}
	}
	return false
}

// importDerivedCount derived 里实际用上的字段个数（去空）。
func importDerivedCount(derived []string) int {
	n := 0
	for _, d := range derived {
		if strings.TrimSpace(d) != "" {
			n++
		}
	}
	return n
}

// importNormText 文本比对的归一：去空白、忽略大小写、上标数字与全角标点归成 ASCII。
func importNormText(s string) string {
	r := strings.NewReplacer("²", "2", "³", "3", "（", "(", "）", ")", "：", ":", "，", ",", "．", ".", "　", " ")
	return strings.ToLower(strings.Join(strings.Fields(r.Replace(s)), ""))
}

func importIsDigit(r rune) bool { return r >= '0' && r <= '9' }

func importAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !importIsDigit(r) {
			return false
		}
	}
	return true
}

// importParsePercent 「3%」→ 0.03。
func importParsePercent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "%") {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, "%")), 64)
	if err != nil {
		return 0, false
	}
	return n / 100, true
}

// importCloseNum 数值相等（半分钱以内；大数按百万分之一相对误差）。
func importCloseNum(a, b float64) bool {
	return math.Abs(a-b) <= math.Max(0.005, math.Abs(b)*1e-6)
}

// importNums 该行里的数（回执里帮模型定位抄错用，最多列 6 个）。
func importNums(cells []importCell) string {
	var out []string
	for _, c := range cells {
		if c.IsNum {
			out = append(out, strconv.FormatFloat(c.Num, 'f', -1, 64))
		}
		if len(out) >= 6 {
			break
		}
	}
	if len(out) == 0 {
		return "（该行没有数）"
	}
	return strings.Join(out, "、")
}

// sourceReadable 这次导入能不能回读（登记了路径且格式支持）。读取器懒加载；打不开
// 或格式不支持时记成「没有可回读的资料」，之后不再重试。
func (imp *ImportSession) sourceReadable() bool {
	if imp.SourcePath == "" {
		return false
	}
	if imp.source == nil {
		is, err := openImportSource(imp.SourcePath)
		if err != nil {
			imp.source = &importSource{kind: "none"}
		} else {
			imp.source = is
		}
	}
	return imp.source.kind != "none"
}

// readSource 读来源行（调用前先过 sourceReadable）。
func (imp *ImportSession) readSource(loc ImportSrc) ([]importCell, string, bool, error) {
	return imp.source.read(loc)
}

// ── 提交 ─────────────────────────────────────────────────────────────

// ImportWriteResult 提交导入的写入结果。
type ImportWriteResult struct {
	ProjectID    int64    `json:"projectId"`
	ProjectName  string   `json:"projectName"`
	ResourcesNew int      `json:"resourcesNew"`
	ResourcesUpd int      `json:"resourcesUpd"`
	QuotasNew    int      `json:"quotasNew"`
	QuotasUpd    int      `json:"quotasUpd"`
	Bills        int      `json:"bills"`
	PriceFacts   int      `json:"priceFacts"`
	Errors       []string `json:"errors"` // 写入中途失败的行（其余已写；修掉后重新提交即可）
}

// Commit 把暂存区写进库。写前把每一行再过一遍规矩与引用完整——有任何一行不合规
// 就整批不写（回执点名，模型改好重新 import_add 覆盖后再提交）。写入全是按编码/
// 身份的幂等更新：先工料机、后定额、最后清单；万一中途出错，修掉后重新提交即可，
// 已写的行按更新处理不会重复。
func (imp *ImportSession) Commit(s *Store) (*ImportWriteResult, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	res := &ImportWriteResult{Errors: []string{}}
	_, _, bills := imp.Staged()
	if strings.TrimSpace(imp.Project.Name) == "" && bills > 0 {
		return nil, fmt.Errorf("暂存区里有 %d 条清单行，但这次导入没有项目（import_open 的 project.name）——清单行必须归属项目", bills)
	}

	// 写前闸：逐行复校，收齐全部问题。
	var problems []string
	for _, r := range imp.resources {
		if _, err := imp.resourceOf(s, r); err != nil {
			problems = append(problems, fmt.Sprintf("工料机 %s「%s」：%v", r.Code, r.Title, err))
		}
	}
	for _, q := range imp.quotas {
		if _, reasons := imp.quotaOf(s, q); len(reasons) > 0 {
			problems = append(problems, fmt.Sprintf("定额 %s「%s」：%s", q.Code, q.Title, strings.Join(reasons, "；")))
		}
	}
	for _, b := range imp.bills {
		if _, _, reasons := imp.billOf(s, b); len(reasons) > 0 {
			problems = append(problems, fmt.Sprintf("清单 %s「%s」：%s", b.Code, b.Title, strings.Join(reasons, "；")))
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("暂存区有 %d 行已不合规，未写入任何数据（改好重新 import_add 覆盖，再 import_check）：%s",
			len(problems), strings.Join(problems, "；"))
	}

	// ① 项目封面与口径（只登记，不加总）。
	if name := strings.TrimSpace(imp.Project.Name); name != "" {
		flag := imp.Project.ProfitIncludesRegulatory != nil && *imp.Project.ProfitIncludesRegulatory
		var control float64
		if imp.Project.ControlPrice != nil {
			control = *imp.Project.ControlPrice
		}
		id, err := s.UpsertBillProject(name, strings.TrimSpace(imp.Project.FileName),
			firstNonEmpty(strings.TrimSpace(imp.Project.Source), imp.defaultSource()),
			strings.TrimSpace(imp.Project.Location), strings.TrimSpace(imp.Project.Duration),
			strings.TrimSpace(imp.Project.Pricing), imp.Project.rates(), flag, control)
		if err != nil {
			return nil, fmt.Errorf("项目「%s」落库失败: %w", name, err)
		}
		if err := s.UpdateBillProjectProfile(id, imp.Project.Specialty, imp.Project.Region, imp.Period); err != nil {
			return nil, fmt.Errorf("项目「%s」的专业/地区/计价期落库失败: %w", name, err)
		}
		res.ProjectID, res.ProjectName = id, name
	}

	// ② 工料机：按身份 UPSERT（同身份已有行=更新，编码沿用库里的）。
	for _, in := range imp.resources {
		r, _ := imp.resourceOf(s, in)
		prev, err := s.findResourceByIdentity(*r)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("工料机 %s「%s」身份回查失败: %v", in.Code, in.Title, err))
			continue
		}
		saved, err := s.SaveResource(*r)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("工料机 %s「%s」写入失败: %v", in.Code, in.Title, err))
			continue
		}
		if prev != nil {
			res.ResourcesUpd++
		} else {
			res.ResourcesNew++
		}
		if price := saved.EffectivePrice(); price > 0 && saved.PriceDate != "" {
			if err := s.RecordPriceFact(saved.ID, price, saved.PriceDate, saved.Region, saved.PriceType,
				saved.Source, "导入时按提交价记"); err == nil {
				res.PriceFacts++
			}
		}
	}

	// ③ 定额（含量行引用的工料机此时都已在库）。
	idx, err := s.ResourceIndex()
	if err != nil {
		return nil, err
	}
	for _, in := range imp.quotas {
		q, _ := imp.quotaOf(s, in)
		_, existErr := s.GetQuota(q.Code)
		if _, err := s.saveQuota(s.db, *q, idx); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("定额 %s「%s」写入失败: %v", in.Code, in.Title, err))
			continue
		}
		if existErr == nil {
			res.QuotasUpd++
		} else {
			res.QuotasNew++
		}
	}

	// ④ 清单行与定额挂接。
	for _, in := range imp.bills {
		b, links, _ := imp.billOf(s, in)
		b.ProjectID = res.ProjectID
		saved, err := s.SaveBillItem(*b)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("清单 %s「%s」写入失败: %v", in.Code, in.Title, err))
			continue
		}
		if len(links) > 0 {
			if _, err := s.SaveBillItemLinks(saved.ID, links); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("清单 %s 的定额挂接写入失败: %v", in.Code, err))
				continue
			}
		}
		res.Bills++
	}
	imp.Close()
	return res, nil
}
