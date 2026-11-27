// 工料法成本数据库（工料机资源库 + 消耗定额库）——绑定门面 CostB。
//
// 用户定调：工料机（人材材机）是基本数据，综合单价分析表靠「工料机 × 消耗定额」
// 核算得出。本文件承载第①②层的 app 入口；综合单价是核算**输出**，不进本文件。
package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// workcostStoreOverride 测试注入的隔离工料法存储。
var (
	workcostStoreOverride    *workcost.Store
	workcostStoreOverrideMu  sync.RWMutex
	workcostStoreOverrideSet bool
)

// SetWorkcostStoreForTest 注入隔离存储（测试用）。
func SetWorkcostStoreForTest(s *workcost.Store) {
	workcostStoreOverrideMu.Lock()
	defer workcostStoreOverrideMu.Unlock()
	workcostStoreOverride = s
	workcostStoreOverrideSet = true
}

// ResetWorkcostStoreForTest 清除测试注入。
func ResetWorkcostStoreForTest() {
	workcostStoreOverrideMu.Lock()
	defer workcostStoreOverrideMu.Unlock()
	workcostStoreOverrideSet = false
}

// hubWorkcostStore 构造工料法存储（Hephaestus.db）。
func (a *App) hubWorkcostStore() *workcost.Store {
	workcostStoreOverrideMu.RLock()
	if workcostStoreOverrideSet {
		s := workcostStoreOverride
		workcostStoreOverrideMu.RUnlock()
		return s
	}
	workcostStoreOverrideMu.RUnlock()
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return workcost.Open(nil)
	}
	return workcost.Open(db.GetDatabase(userDir))
}

// ── 工料机资源库 ────────────────────────────────────────────────────

// GaeaWorkcostResourceSave 新建/更新工料机资源（同身份 UPDATE，新身份 INSERT），
// 返回落库后的资源（含自动分配的编码）。
func (a *App) GaeaWorkcostResourceSave(r workcost.Resource) (*workcost.Resource, error) {
	saved, err := a.hubWorkcostStore().SaveResource(r)
	if err != nil {
		return nil, err
	}
	// 新建资源时若无基准价，用现行价补齐——否则 EffectivePrice 会取到 0。
	if saved.BasePrice <= 0 && saved.CurrentPrice > 0 {
		saved.BasePrice = saved.CurrentPrice
		if _, err := a.hubWorkcostStore().SaveResource(*saved); err != nil {
			return saved, err
		}
	}
	return saved, nil
}

// GaeaWorkcostResourceGet 按 id 读取资源；不存在返回 nil。
func (a *App) GaeaWorkcostResourceGet(id int64) *workcost.Resource {
	r, err := a.hubWorkcostStore().GetResource(id)
	if err != nil {
		return nil
	}
	return r
}

// GaeaWorkcostResourceList 资源列表（kind/keyword 可空 = 不过滤）。
func (a *App) GaeaWorkcostResourceList(kind string, keyword string) []workcost.Resource {
	if a.costSearchEnabled(workcost.IndexKindResource, keyword) {
		if hits, err := a.costSearcher().Resources(context.Background(), workcost.SearchQuery{
			Text: keyword, Kind: kind, Limit: costSearchListLimit,
		}); err == nil {
			out := make([]workcost.Resource, len(hits))
			for i, h := range hits {
				out[i] = h.Item
			}
			return nonNilSlice(out)
		}
	}
	list, err := a.hubWorkcostStore().ListResources(kind, keyword)
	if err != nil {
		return []workcost.Resource{}
	}
	return nonNilSlice(list)
}

// GaeaWorkcostResourceDelete 删除资源。
// force=false 时拒绝删除仍被消耗定额引用的资源（引用完整性）。
func (a *App) GaeaWorkcostResourceDelete(id int64, force bool) error {
	return a.hubWorkcostStore().DeleteResource(id, force)
}

// GaeaWorkcostResourceSetPrice 登记一次资源调价（写调价历史 + 推进现行价）。
//
// 这是「信息价一涨、所有引用该资源的综合单价同步重算」的唯一入口——现行价
// 变了，引用它的定额子目在下次读取时自然按新价重算，无需逐条改价。
func (a *App) GaeaWorkcostResourceSetPrice(id int64, price float64, period string, region string, priceType string, source string, note string) (*workcost.Resource, error) {
	return a.hubWorkcostStore().UpdateResourcePrice(id, price, period, region, priceType, source, note)
}

// GaeaWorkcostResourcePrices 某资源的调价历史（新→旧）。
func (a *App) GaeaWorkcostResourcePrices(id int64) []workcost.ResourcePrice {
	list, err := a.hubWorkcostStore().ResourcePrices(id)
	if err != nil {
		return []workcost.ResourcePrice{}
	}
	return nonNilSlice(list)
}

// GaeaWorkcostResourceUsage 某工料机被哪些定额含量行引用（三级下钻的回链：
// 从定额点到工料机之后，看这条工料机被谁用）。构造即非 nil。
func (a *App) GaeaWorkcostResourceUsage(code string) []workcost.UsageRef {
	refs, err := a.hubWorkcostStore().ResourceUsage(code)
	if err != nil {
		return []workcost.UsageRef{}
	}
	return nonNilSlice(refs)
}

// ── 消耗定额库 ──────────────────────────────────────────────────────

// GaeaWorkcostQuotaSave 新建/更新消耗定额（含工料机含量行，整组替换）。
func (a *App) GaeaWorkcostQuotaSave(q workcost.Quota) (*workcost.Quota, error) {
	return a.hubWorkcostStore().SaveQuota(q, nil)
}

// GaeaWorkcostQuotaGet 按编码读取定额（含含量行）；不存在返回 nil。
func (a *App) GaeaWorkcostQuotaGet(code string) *workcost.Quota {
	q, err := a.hubWorkcostStore().GetQuota(code)
	if err != nil || q == nil {
		return nil
	}
	// 空定额（无含量行）Items 归一非 nil：nil 切片序列化 JSON null 崩前端。
	q.Items = nonNilSlice(q.Items)
	return q
}

// GaeaWorkcostQuotaList 定额列表（specialty/keyword 可空 = 不过滤）。索引就绪且带
// keyword 时走混合检索（相关度序，含同义说法召回）；否则按原列表口径全量返回。
func (a *App) GaeaWorkcostQuotaList(specialty string, keyword string) []workcost.Quota {
	if a.costSearchEnabled(workcost.IndexKindQuota, keyword) {
		if hits, err := a.costSearcher().Quotas(context.Background(), workcost.SearchQuery{
			Text: keyword, Specialty: specialty, Limit: costSearchListLimit,
		}); err == nil {
			out := make([]workcost.Quota, len(hits))
			for i, h := range hits {
				out[i] = h.Item
			}
			return nonNilSlice(out)
		}
	}
	list, err := a.hubWorkcostStore().ListQuotas(specialty, keyword)
	if err != nil {
		return []workcost.Quota{}
	}
	return nonNilSlice(list)
}

// GaeaWorkcostQuotaDelete 删除定额及其含量行。仍被清单项引用时拒绝——造价
// 数据库板块是只读浏览面，不留悬空引用；确要连带解除引用走造价模式的
// cost_library delete_quota（force=true，逐条确认）。
func (a *App) GaeaWorkcostQuotaDelete(code string) error {
	s := a.hubWorkcostStore()
	if n, err := s.QuotaReferenced(code); err == nil && n > 0 {
		return fmt.Errorf("该定额仍被 %d 条清单项引用，不能删除——先在造价模式解除挂接", n)
	}
	return s.DeleteQuota(code)
}

// GaeaWorkcostQuotaCompose 按定额核算综合单价（**只含人材机**，不含管理费/
// 利润/税金）。overrides 支持项目级覆盖含量/单价（key=资源编码；含量或单价
// 传负值表示不覆盖该项）。
func (a *App) GaeaWorkcostQuotaCompose(code string, overrides map[string]workcost.ComposeOverride) (*workcost.Compose, error) {
	c, _, err := a.hubWorkcostStore().ComposeQuota(code, overrides)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GaeaWorkcostQuotaComposeMany 批量核算综合单价（清单库参考价列专用）：一次
// 资源索引复用，替代逐行调用（63 行清单=63 次全资源扫描，真机首屏 longtask
// ~700ms）。key=定额编码；不存在的编码不在结果里（前端按缺失回退显示）。
func (a *App) GaeaWorkcostQuotaComposeMany(codes []string) (map[string]*workcost.Compose, error) {
	return a.hubWorkcostStore().ComposeQuotas(codes)
}

// ── 分部分项清单（工料法第④层：录入数据，不做加总）─────────────────

// GaeaWorkcostBillProjects 清单项目列表（含封面、费率与清单项数）。
// 造价数据库是数据库：清单/费率/模版都是录入数据，加总计算不在库里发生。
func (a *App) GaeaWorkcostBillProjects() []workcost.BillProject {
	list, err := a.hubWorkcostStore().BillProjects()
	if err != nil {
		return []workcost.BillProject{}
	}
	return nonNilSlice(list)
}

// GaeaWorkcostBillItems 某项目的分部分项清单（编码/名称/单位/工程量/引用定额）。
func (a *App) GaeaWorkcostBillItems(projectID int64) []workcost.BillItem {
	list, err := a.hubWorkcostStore().BillItems(projectID)
	if err != nil {
		return []workcost.BillItem{}
	}
	return nonNilSlice(list)
}

// GaeaWorkcostBillItemSave 新增/更新一条清单项（同项目同编码走更新；code 空
// 自动生成 M 序号）。工程量与引用定额都是录入数据——合价导出时由 Excel 算。
func (a *App) GaeaWorkcostBillItemSave(item workcost.BillItem) (*workcost.BillItem, error) {
	saved, err := a.hubWorkcostStore().SaveBillItem(item)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// GaeaWorkcostBillItemDelete 删除一条清单项。
func (a *App) GaeaWorkcostBillItemDelete(id int64) error {
	return a.hubWorkcostStore().DeleteBillItem(id)
}

// GaeaWorkcostBillProjectDelete 整体删除导入的项目（封面+清单+独占定额；
// 共享定额与资源保留）。返回删除的独占定额数。
func (a *App) GaeaWorkcostBillProjectDelete(id int64) (int, error) {
	return a.hubWorkcostStore().DeleteBillProject(id)
}

// ── 清单↔定额 1:N 组合（SchemaV30 gf_bill_quota_links）──────────────

// GaeaWorkcostBillQuotaLinks 清单项挂接的定额引用（1:N；构造即非 nil）。
func (a *App) GaeaWorkcostBillQuotaLinks(billItemID int64) ([]workcost.BillQuotaLink, error) {
	links, err := a.hubWorkcostStore().BillQuotaLinks(billItemID)
	return nonNilSlice(links), err
}

// GaeaWorkcostBillQuotaAttach 挂接一条定额到清单项（同清单同定额幂等更新；
// 定额必须存在）。quantity≤0 表示跟随清单工程量。
func (a *App) GaeaWorkcostBillQuotaAttach(billItemID int64, quotaCode string, quantity float64) (workcost.BillQuotaLink, error) {
	return a.hubWorkcostStore().AttachBillQuotaLink(billItemID, quotaCode, quantity)
}

// GaeaWorkcostBillQuotaDetach 解挂一条定额引用。
func (a *App) GaeaWorkcostBillQuotaDetach(billItemID int64, linkID int64) error {
	return a.hubWorkcostStore().DetachBillQuotaLink(billItemID, linkID)
}

// GaeaWorkcostBillProjectRatesSave 保存项目费率（企管/规费/利润/税率 + 利润
// 基数含规费开关 + 控制价）——录入数据，随取随改，不参与任何库内计算。
func (a *App) GaeaWorkcostBillProjectRatesSave(id int64, rates workcost.RateSet, profitIncludesRegulatory bool, controlPrice float64) error {
	return a.hubWorkcostStore().UpdateBillProjectRates(id, rates, profitIncludesRegulatory, controlPrice)
}

// GaeaWorkcostBillProjectProfileSave 保存项目口径（专业/地区/计价期；空=保持
// 原值），规范值校验走 store（专业、地区须在字典里，计价期归一 YYYY-MM）。
// 造价库板块「项目封面」编辑档用——录入错了口径别再绕回造价模式归类。
func (a *App) GaeaWorkcostBillProjectProfileSave(id int64, specialty, region, pricePeriod string) error {
	return a.hubWorkcostStore().UpdateBillProjectProfile(id, specialty, region, pricePeriod)
}

// ── 项目层取费 ──────────────────────────────────────────────────────

// GaeaWorkcostProjectFees 项目合计层取费（直接费 → 企管/利润/规费 → 税前合计
// → 增值税 → 含税总造价，含招标控制价对照）。
//
// 口径：综合单价只含人材机，取费只在合计层跑一次（实测产物「费用汇总」下半段）。
// profitIncludesRegulatory 决定利润基数是否含规费（百锦路=false，市政道路=true）。
func (a *App) GaeaWorkcostProjectFees(directFee float64, rates workcost.RateSet, profitIncludesRegulatory bool, measures float64, contingency float64, controlPrice float64) workcost.FeeResult {
	return workcost.ComposeProjectFees(directFee, rates, workcost.FeePolicy{
		ProfitBaseIncludesRegulatory: profitIncludesRegulatory,
	}, measures, contingency, controlPrice)
}

// ── 项目工作簿解析与录入（绑定面）───────────────────────────────────
//
// 2026-11-27：测算簿的硬契约只有综合单价表（三级树表），其余表是参考——规矩在
// workcost.ParseProjectWorkbook 与 cost-estimate 技能。旧的「解析 → 重新导出」
// 渲染器（产出「行类」版式）已删：它导出的簿导不回，样表出口改走样表复制。

// GaeaWorkcostProjectParse 解析一份测算工作簿，**只解析不落库**。
func (a *App) GaeaWorkcostProjectParse(path string) (*workcost.ProjectBundle, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("需要工作簿路径")
	}
	return workcost.ParseProjectWorkbook(path)
}

// GaeaWorkcostProjectApply 把测算工作簿解析后直接落库：工料机价格 → 资源库，
// 综合单价的定额块 → 消耗定额库，清单行 → 项目清单。幂等（按身份 / 编码 UPSERT）。
//
// 综合单价是核算输出，由 GaeaWorkcostQuotaCompose 现算，不落库（否则派生值会与
// 资源价漂移）。日常录入走办公「造价」运行模式的 cost_library（带演练与用户
// 确认）；本绑定不在前端门面生成范围内，只作绑定面的端到端验证。
func (a *App) GaeaWorkcostProjectApply(path string) (workcost.ApplyProjectResult, error) {
	b, err := workcost.ParseProjectWorkbook(path)
	if err != nil {
		return workcost.ApplyProjectResult{}, err
	}
	return a.hubWorkcostStore().ApplyProjectBundle(b), nil
}
