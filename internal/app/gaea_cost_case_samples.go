package app

import (
	"log/slog"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// 项目案例清单单价样本（造价数据库 master plan 三期 3.3 对标升级，2026-10-14）：
// 把已导入项目案例的清单综合单价并入 AI 组价的分位数聚合——资料条目（手册/
// 信息价）之外的第二路真实市场样本，来源列以「案例」区分（BandSource.Source，
// 证据表/compose 证据链原样渲染，前端零改动）。
//
// 归一口径=名称+单位（计划文档 3.3）：名称去空白小写后双向包含（短侧 ≥2 字
// 防单字误伤），单位宽松相等（与 ComputePriceBand 的单位过滤同款，空=不过滤）。
//
// 单价解析：手填价（PriceOverride>0，外委包干/项目级调价）优先；单定额清单项
// 按定额现算（ComposeQuotas 批一次，CompositePrice=人材机综合单价口径）；1:N
// 组合项无单一单价（只取主定额会低估组合价），诚实跳过。
//
// 增强面语义：库不可达/读失败降级（slog.Warn）为仅资料条目样本，绝不挡组价
// 主流程——案例只是样本增量，与检索主链（GA6-09 语料读不全如实报错）不同层。
func (a *App) casePriceSamples(desc, unit string) []cost.Summary {
	store := a.hubWorkcostStore()
	projects, err := store.BillProjects()
	if err != nil {
		slog.Warn("案例样本：项目清单读取失败（降级为仅资料条目样本）", "error", err)
		return nil
	}
	if len(projects) == 0 {
		return nil
	}

	ndesc := normalizeCostName(desc)
	wantUnit := strings.TrimSpace(unit)

	type cand struct {
		name      string  // 清单项名称（样本 Name）
		unit      string  // 清单项单位
		projTitle string  // 项目名（样本 Title，证据表溯源到工程）
		region    string  // 项目地点（样本 Region）
		price     float64 // 手填单价（>0 即定）
		quotaCode string  // 待现算定额码（空=手填价已定或跳过）
	}
	var cands []cand
	for _, p := range projects {
		items, err := store.BillItems(p.ID)
		if err != nil {
			continue
		}
		for _, it := range items {
			if !caseNameMatch(ndesc, normalizeCostName(it.Title)) {
				continue
			}
			iu := strings.TrimSpace(it.Unit)
			if wantUnit != "" && iu != wantUnit {
				continue
			}
			c := cand{name: it.Title, unit: iu, projTitle: p.Name, region: p.Location, price: it.PriceOverride}
			if c.price <= 0 {
				code := strings.TrimSpace(it.QuotaCode)
				if len(it.QuotaCodes) == 1 {
					code = strings.TrimSpace(it.QuotaCodes[0])
				} else if len(it.QuotaCodes) > 1 {
					continue // 1:N 组合项无单一单价，不做样本
				}
				if code == "" {
					continue // 无手填价、无定额=无单价来源
				}
				c.quotaCode = code
			}
			cands = append(cands, c)
		}
	}
	if len(cands) == 0 {
		return nil
	}

	// 待现算码去重后批一次（ComposeQuotas 内部单遍资源索引）
	var codes []string
	seen := map[string]bool{}
	for _, c := range cands {
		if c.quotaCode != "" && !seen[c.quotaCode] {
			seen[c.quotaCode] = true
			codes = append(codes, c.quotaCode)
		}
	}
	var composes map[string]*workcost.Compose
	if len(codes) > 0 {
		if comps, err := store.ComposeQuotas(codes); err == nil {
			composes = comps
		} else {
			slog.Warn("案例样本：定额核算失败（仅保留手填价样本）", "error", err)
		}
	}

	out := make([]cost.Summary, 0, len(cands))
	for _, c := range cands {
		price := c.price
		if price <= 0 {
			cp, ok := composes[c.quotaCode]
			if !ok || cp == nil || cp.CompositePrice <= 0 {
				continue // 现算不出（定额缺失/零价）——不造 0 价样本
			}
			price = cp.CompositePrice
		}
		out = append(out, cost.Summary{
			Name:      c.name,
			Title:     "案例·" + c.projTitle,
			Unit:      c.unit,
			Price:     price,
			Source:    "案例",
			Region:    c.region,
			PriceType: "综合单价",
		})
	}
	return out
}

// normalizeCostName 清单项名称归一：去全部空白（含全角空格）+ 小写。
func normalizeCostName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// caseNameMatch 双向包含（入参须先经 normalizeCostName）：短侧 ≥2 字防单字
// 误伤（查询「墙」不该匹配全部墙类清单）。
func caseNameMatch(a, b string) bool {
	as, bs := []rune(a), []rune(b)
	short, long := as, bs
	if len(short) > len(long) {
		short, long = bs, as
	}
	if len(short) < 2 {
		return false
	}
	return strings.Contains(string(long), string(short))
}
