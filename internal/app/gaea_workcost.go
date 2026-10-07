// 工料法成本数据库（工料机资源库 + 消耗定额库）——绑定门面 CostB。
//
// 用户定调：工料机（人材材机）是基本数据，综合单价分析表靠「工料机 × 消耗定额」
// 核算得出。本文件承载第①②层的 app 入口；综合单价是核算**输出**，不进本文件。
package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/spaces"
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

// hubWorkcostStore 构造工料法存储（Hephaestus.db，与成本库同库）。
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
	list, err := a.hubWorkcostStore().ListResources(kind, keyword)
	if err != nil {
		return nil
	}
	return list
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
		return nil
	}
	return list
}

// ── 消耗定额库 ──────────────────────────────────────────────────────

// GaeaWorkcostQuotaSave 新建/更新消耗定额（含工料机含量行，整组替换）。
func (a *App) GaeaWorkcostQuotaSave(q workcost.Quota) (*workcost.Quota, error) {
	return a.hubWorkcostStore().SaveQuota(q, nil)
}

// GaeaWorkcostQuotaGet 按编码读取定额（含含量行）；不存在返回 nil。
func (a *App) GaeaWorkcostQuotaGet(code string) *workcost.Quota {
	q, err := a.hubWorkcostStore().GetQuota(code)
	if err != nil {
		return nil
	}
	return q
}

// GaeaWorkcostQuotaList 定额列表（specialty/keyword 可空 = 不过滤）。
func (a *App) GaeaWorkcostQuotaList(specialty string, keyword string) []workcost.Quota {
	list, err := a.hubWorkcostStore().ListQuotas(specialty, keyword)
	if err != nil {
		return nil
	}
	return list
}

// GaeaWorkcostQuotaDelete 删除定额及其含量行。
func (a *App) GaeaWorkcostQuotaDelete(code string) error {
	return a.hubWorkcostStore().DeleteQuota(code)
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
		return nil
	}
	return list
}

// GaeaWorkcostBillItems 某项目的分部分项清单（编码/名称/单位/工程量/引用定额）。
func (a *App) GaeaWorkcostBillItems(projectID int64) []workcost.BillItem {
	list, err := a.hubWorkcostStore().BillItems(projectID)
	if err != nil {
		return nil
	}
	return list
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
	return a.hubWorkcostStore().BillQuotaLinks(billItemID)
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

// ── 存量资源化（干跑预览 + 应用）────────────────────────────────────

// GaeaWorkcostSeedPreview 存量成本条目 → 工料机资源库的**干跑预览**（不写库）。
// 返回候选清单、归类分布与跳过原因，供用户在入库前审阅。
func (a *App) GaeaWorkcostSeedPreview(includeComposite bool) workcost.SeedPreview {
	inputs := a.workcostSeedInputs()
	return workcost.PreviewSeed(inputs, workcost.SeedOptions{IncludeComposite: includeComposite})
}

// GaeaWorkcostSeedApply 应用资源化：把存量条目归入工料机资源库。
// 幂等——同身份资源走 UPDATE，重复调用不制造重复资源。
func (a *App) GaeaWorkcostSeedApply(includeComposite bool) (workcost.SeedResult, error) {
	inputs := a.workcostSeedInputs()
	cands, _ := workcost.BuildSeedCandidates(inputs, workcost.SeedOptions{IncludeComposite: includeComposite})
	if len(cands) == 0 {
		return workcost.SeedResult{}, fmt.Errorf("没有可资源化的条目（库内成本条目为空或全部不满足规则）")
	}
	res := a.hubWorkcostStore().ApplySeed(cands)
	// 回写成本条目：标记这些条目已成为资源（price_derived 保持 0=手填价语义，
	// 但在 body 追加溯源标记会让同一批数据在两次运行时重复追加——故只改内存
	// 计数，不做文本改写）。留一条 info 日志便于排查。
	slog.Info("工料法资源化完成", "created", res.Created, "updated", res.Updated,
		"skipped", res.Skipped, "errors", len(res.Errors))
	return res, nil
}

// workcostSeedInputs 读取全部成本条目为资源化输入。
// 读失败返回空切片（调用方得到「无可资源化条目」，不会把读坏装成成功）。
func (a *App) workcostSeedInputs() []workcost.SeedInput {
	store := a.hubCostStore()
	if !store.Available() {
		return nil
	}
	rows := store.DB()
	if rows == nil {
		return nil
	}
	q, err := rows.Query(`
SELECT name, title, spec, unit, price, category_path, source, region, price_date, price_type, body
FROM cost_entries`)
	if err != nil {
		slog.Warn("工料法资源化：读取成本条目失败", "error", err)
		return nil
	}
	defer func() { _ = q.Close() }()
	var out []workcost.SeedInput
	for q.Next() {
		var in workcost.SeedInput
		if err := q.Scan(&in.Name, &in.Title, &in.Spec, &in.Unit, &in.Price,
			&in.CategoryPath, &in.Source, &in.Region, &in.PriceDate, &in.PriceType, &in.Body); err != nil {
			continue
		}
		out = append(out, in)
	}
	if err := q.Err(); err != nil {
		slog.Warn("工料法资源化：成本条目迭代中断，返回部分数据", "error", err)
	}
	return out
}

// ── 项目工作簿（五表模版）解析与录入 ─────────────────────────────────

// GaeaWorkcostProjectParse 解析一份五表项目工作簿（封面/费用汇总/综合单价/
// 工料机价格/工程量计算），**只解析不落库**，供前端预览后确认。
func (a *App) GaeaWorkcostProjectParse(path string) (*workcost.ProjectBundle, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("需要工作簿路径")
	}
	return workcost.ParseProjectWorkbook(path)
}

// GaeaWorkcostProjectApply 把解析出的项目落库：工料机价格表 → 资源库，
// 综合单价表（清单项 + 工序消耗量）→ 消耗定额库。幂等（按身份/编码 UPSERT）。
//
// 只落工料法前两层——综合单价是核算输出，由 GaeaWorkcostQuotaCompose 现算，
// 不落库（否则派生值会与资源价漂移）。
func (a *App) GaeaWorkcostProjectApply(path string) (workcost.ApplyProjectResult, error) {
	b, err := workcost.ParseProjectWorkbook(path)
	if err != nil {
		return workcost.ApplyProjectResult{}, err
	}
	return a.hubWorkcostStore().ApplyProjectBundle(b), nil
}

// GaeaWorkcostProjectExport 把项目工作簿解析结果重新导出为**五表模版**工作簿，
// 写出到指定路径（活公式：改资源价/费率后 Excel 全表重算）。
//
// 入参为源工作簿路径：导出内容取自解析结果（工料机价格 → 资源、综合单价 →
// 消耗量、费用汇总 → 取费参数、工程量计算 → 工程量），因此「导出」= 用当前
// 工料法口径重新生成一份同构模版，可用于「导出 → 改价 → 导回」闭环。
func (a *App) GaeaWorkcostProjectExport(srcPath string, outPath string, project string, location string, duration string) (string, error) {
	if strings.TrimSpace(srcPath) == "" || strings.TrimSpace(outPath) == "" {
		return "", fmt.Errorf("需要源工作簿与输出路径")
	}
	b, err := workcost.ParseProjectWorkbook(srcPath)
	if err != nil {
		return "", err
	}
	data, err := workcost.ExportProjectWorkbook(b, workcost.ExportOptions{
		Project: project, Location: location, Duration: duration,
	})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return "", fmt.Errorf("写出工作簿失败: %w", err)
	}
	return outPath, nil
}

// GaeaWorkcostProjectExportToWorkspace 把五表工作簿导出到工作区 `.gaea/exports`
// （与既有导出链同分区约定：work → .gaea/exports，play → .gaea/play/exports），
// 返回写出路径。文件名带时间戳避免覆盖。
//
// 与 GaeaWorkcostProjectExport（接显式路径）的分工：本方法给 UI 用——前端没有
// 文件系统能力，由后端按工作区约定落盘；前者给「另存为」类显式路径场景。
func (a *App) GaeaWorkcostProjectExportToWorkspace(srcPath string, fileName string) (string, error) {
	if strings.TrimSpace(srcPath) == "" {
		return "", fmt.Errorf("需要源工作簿路径")
	}
	b, err := workcost.ParseProjectWorkbook(srcPath)
	if err != nil {
		return "", err
	}
	data, err := workcost.ExportProjectWorkbook(b, workcost.ExportOptions{})
	if err != nil {
		return "", err
	}
	exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	base := strings.TrimSpace(fileName)
	if base == "" {
		base = "成本测算表"
	}
	base = strings.TrimSuffix(base, ".xlsx")
	stamp := time.Now().Format("20060102-150405")
	out := filepath.Join(exportsDir, fmt.Sprintf("%s-%s.xlsx", base, stamp))
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", fmt.Errorf("写出工作簿失败: %w", err)
	}
	return filepath.ToSlash(out), nil
}

// ── 核算缓存化（cost_entries.price 降级为核算结果）──────────────────
// GaeaWorkcostRecompose 把成本库中**带工料机组成**的条目的 price 重算为
// Σ(含量×单价×(1+损耗))，并置 price_derived=1（声明该价是核算缓存价）。
//
// 无组成的条目不参与（其 price 是查到的当期价，重算会把它抹成 0）。实测用户库
// 组成行 0 条，故当前为 no-op——它是后续回填组成后的重算入口。
func (a *App) GaeaWorkcostRecompose() (workcost.RecomposeResult, error) {
	costStore := a.hubCostStore()
	if !costStore.Available() {
		return workcost.RecomposeResult{}, fmt.Errorf("成本库不可用")
	}
	return workcost.RecomposeEntries(workcost.RecomposeHooks{
		Names: func() ([]string, error) {
			list, err := costStore.List()
			if err != nil {
				return nil, err
			}
			names := make([]string, 0, len(list))
			for _, s := range list {
				names = append(names, s.Name)
			}
			return names, nil
		},
		Load: func(name string) (*workcost.RecomposeEntry, error) {
			e, err := costStore.Get(name)
			if err != nil || e == nil {
				return nil, err
			}
			out := &workcost.RecomposeEntry{
				Name: e.Name, Title: e.Title, Unit: e.Unit, CategoryPath: e.CategoryPath,
			}
			for _, c := range e.Components {
				out.Components = append(out.Components, workcost.RecomposeComponent{
					Kind: c.Kind, Title: c.Title, Unit: c.Unit,
					Quantity: c.Quantity, Price: c.Price,
				})
			}
			return out, nil
		},
		Write: func(w workcost.RecomposeWrite) error {
			e, err := costStore.Get(w.Name)
			if err != nil || e == nil {
				return fmt.Errorf("条目 %q 不存在", w.Name)
			}
			// 只改价与三费，其余字段（规格/来源/期数/组成行）原样保留——
			// SaveRecomposed 内部走 saveTx 会整组重写组成行，故先把组成带回去。
			e.Price = w.Price
			e.LaborFee = w.LaborFee
			e.MaterialFee = w.MaterialFee
			e.MachineFee = w.MachineFee
			return costStore.SaveRecomposed(*e)
		},
	})
}
