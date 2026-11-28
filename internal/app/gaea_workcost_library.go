package app

// gaea_workcost_library.go — 造价数据库板块的库维护绑定（2026-11-28 索引与编辑优化批）：
// 字典读写（规范值下拉读字典、就地新增、检索别名）、定额解除全部挂接、定额地区回填。
//
// 边界变更（2026-11-28 用户令「优化完善造价数据库的索引和编辑功能」，推翻 11-27 的
// 「批量维护只归造价模式」里的一刀切）：板块里可以勾选批量改维度、可以新增字典值——
// 真实库里 330 条定额未归分部，靠模型逐条归类太慢。批量导入、解析、Excel 批量调价
// 仍归造价模式。批量改维度不走这里的专门绑定：板块按行调既有的保存入口
// （Workcost*Save，规矩校验在存储层把关），逐行回报成败。

import (
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// GaeaWorkcostDictList 读造价库字典。kind：specialty（专业）/ region（地区）/
// category（工料机分类）/ alias（检索别名），空 = 全部。
func (a *App) GaeaWorkcostDictList(kind string) []workcost.DictItem {
	items, err := a.hubWorkcostStore().DictList(kind)
	if err != nil {
		return []workcost.DictItem{}
	}
	return nonNilSlice(items)
}

// GaeaWorkcostDictSet 写一条字典项：remove=false 新增（已存在 = 幂等），remove=true 删除
// （仍被数据引用的拒删）。检索别名：name = 别名，parent = 对应的标准说法。
func (a *App) GaeaWorkcostDictSet(item workcost.DictItem, remove bool) error {
	if remove {
		return a.hubWorkcostStore().DictDelete(item.Kind, item.Name)
	}
	return a.hubWorkcostStore().DictAdd(item)
}

// GaeaWorkcostQuotaDetachEverywhere 解除一条定额在全部清单项上的挂接（删它之前的清理），
// 返回受影响的清单项数。
func (a *App) GaeaWorkcostQuotaDetachEverywhere(code string) (int, error) {
	return a.hubWorkcostStore().DetachQuotaEverywhere(code)
}

// GaeaWorkcostQuotaRegionBackfill 按上游清单项目回填定额地区：只被一个省的项目引用的
// 定额写上那个省，其余不动并说明原因。dryRun=true 只出方案不写库。
func (a *App) GaeaWorkcostQuotaRegionBackfill(dryRun bool) ([]workcost.QuotaRegionFix, error) {
	plan, err := a.hubWorkcostStore().BackfillQuotaRegions(dryRun)
	if err != nil {
		return []workcost.QuotaRegionFix{}, err
	}
	return nonNilSlice(plan), nil
}
