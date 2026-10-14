package app

// ── 伏笔到期预警（阶段八 8.6）────────────────────────────────
//
// 判定全部复用既有单源（types.UrgencyLevel：0 不紧急 / 1 需关注 / 2 急需回收 /
// 3 已超期；已回收/已废弃/无计划章恒 0——不猜值）。本绑定只做**作者可见性**：
// 超期与临期清单推到作者天天看的地方（章节计划卡/阅读页），不再要求主动去伏笔
// 面板点 Lint 才知道。注入面（foreshadow_context L1/L2 硬约束）与 Lint 报告
// 均不受影响——预警是作者面，include_in_context=false（退出注入面）的条目照常
// 预警（登记表数据完整性口径，见 foreshadow_sync 同款注释）。

import (
	"errors"
	"os"

	"github.com/gaea/gaea/internal/types"
)

// ForeshadowDueItem 单条预警（wire camelCase，见 wire_shape_guard）。
type ForeshadowDueItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	PlantedIn string `json:"plantedIn,omitempty"`
	TargetIn  string `json:"targetIn,omitempty"` // 计划回收章文件名
	Urgency   int    `json:"urgency"`            // 1=需关注 2=急需回收 3=已超期
	// OverdueBy 超期章数（urgency=3 时 >0；target - current 的绝对值）。
	OverdueBy int  `json:"overdueBy,omitempty"`
	LongTerm  bool `json:"longTerm,omitempty"`
}

// ForeshadowDueReport 到期预警报告：Urgent=急需回收+已超期（红），Soon=需关注
// （黄，含「本章必须回收」——urgency=2 的 target==current 子集）。
type ForeshadowDueReport struct {
	ChapterNum int                 `json:"chapterNum"`
	Urgent     []ForeshadowDueItem `json:"urgent"`
	Soon       []ForeshadowDueItem `json:"soon"`
}

// ForeshadowDue 伏笔到期预警（计划页/阅读页挂载点）。chapterNum<=0 回退全书
// 已写章数（GetForeshadowStats 同款口径）；登记表缺失是正常态（空报告）。
func (a *writingState) ForeshadowDue(chapterNum int) (ForeshadowDueReport, error) {
	pm := a.getPM()
	if pm == nil {
		return ForeshadowDueReport{}, errors.New("请先打开项目")
	}
	report := ForeshadowDueReport{
		ChapterNum: chapterNum,
		Urgent:     []ForeshadowDueItem{},
		Soon:       []ForeshadowDueItem{},
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil // 空登记=无预警，正常态
		}
		return ForeshadowDueReport{}, err
	}
	if report.ChapterNum <= 0 {
		report.ChapterNum = countWrittenChapters(pm)
	}
	for _, it := range ff.Items {
		u := types.UrgencyLevel(it, report.ChapterNum)
		if u <= 0 {
			continue
		}
		item := ForeshadowDueItem{
			ID:        it.ID,
			Title:     it.Title,
			Status:    string(types.NormalizeForeshadowStatus(it.Status)),
			PlantedIn: it.PlantedIn,
			TargetIn:  it.TargetResolveIn,
			Urgency:   u,
			LongTerm:  it.IsLongTerm,
		}
		if target := types.ChapterNumOf(it.TargetResolveIn); target > 0 && target < report.ChapterNum {
			item.OverdueBy = report.ChapterNum - target
		}
		if u >= 2 {
			report.Urgent = append(report.Urgent, item)
		} else {
			report.Soon = append(report.Soon, item)
		}
	}
	return report, nil
}
