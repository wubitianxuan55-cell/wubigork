package app

// 上下文编译清单（长篇刀6）：dry-run 复刻生成时的区段组装路径，返回将注入的
// 区段清单（名称/rune 长度/说明）——「同样信息量下 prompt 更短」可量化、调参
// 可见。零生成零写盘。

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/maturecraft"
	"github.com/gaea/gaea/internal/types"
)

// NovelContextInventory 本章生成将注入的上下文区段清单（dry-run）。
// setting 槽由编辑区传入（服务端不可得），清单里以固定行说明其预算口径。
func (a *writingState) NovelContextInventory(chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	type item struct {
		name, body, note string
	}
	items := []item{
		{"setting（小说设定）", "", fmt.Sprintf("由编辑区传入；超过 %d rune 截断并附提示", ctxSettingBudget)},
	}
	add := func(name, body, note string) {
		items = append(items, item{name, body, note})
	}

	// 章节计划 + 大纲要点（刀1；hint 来源）
	planSec, outlineSec := "", ""
	if chapterNum > 0 {
		if plans, err := readChapterPlansForGate(pm); err == nil && plans != nil {
			if p, ok := plans.Plans[fmt.Sprintf("%d", chapterNum)]; ok {
				planSec = buildChapterPlanSection(&p)
			}
		}
		if of, err := pm.ReadOutlines(); err == nil && of != nil {
			outlineSec = buildOutlinePointsSection(findOutlineNodeByNum(of.Nodes, chapterNum))
		}
	}
	add("章节计划", planSec, "刀1 意图注入")
	add("大纲要点", outlineSec, "刀1 意图注入")

	if body := buildForeshadowSection(pm, chapterNum); body != "" {
		add("未回收伏笔", body, "创作约束")
	}
	if body := joinStyleSections(styleSectionBody(pm), a.digestBody(pm)); body != "" {
		add("文风与表达", body, "刀6 合并：显式偏好+成稿学习单区段")
	}
	hint := planSec + "\n" + outlineSec
	if body := buildWorldviewSection(pm, hint); body != "" {
		add("世界观要点", body, "刀6 相关性排序（命中本章关键词的维度优先）")
	}
	if body := a.storySpineSection(pm, chapterNum); body != "" {
		add("故事层骨架切片", body, "刀3：节拍/线程/悬念/弧线")
	}
	if of, err := pm.ReadOutlines(); err == nil && of != nil {
		resolve := func(n types.OutlineNode) string {
			if s := strings.TrimSpace(n.Summary); s != "" {
				return s
			}
			if cs, err := pm.ReadChapterSummary(n.OrderIndex); err == nil && cs != nil {
				if s := strings.TrimSpace(cs.Summary); s != "" {
					return s
				}
			}
			return ""
		}
		if body := buildPrevSummaryWindow(of.Nodes, chapterNum, resolve); body != "" {
			add("前文摘要窗口", body, "最近 10 章")
		}
	}
	if body := a.buildCharacterSummary(pm); body != "" {
		add("角色摘要", body, "出场角色画像")
	}
	// v4.441 成人向档位可见性：档位是用户设置、注入却发生在模板槽里——dry-run
	// 清单必须恒列一行如实回显（含「未启用」），作者才能确认生效面。
	level := pm.Meta.Mature
	note := fmt.Sprintf("档位=%s；正文向纪律随整章/重写/收敛/场景注入", maturecraft.LevelName(level))
	if level == "" {
		note = "档位=未启用；非成人向零注入（创作页「本书定位」或建档时可选）"
	}
	add("成人向工艺区段", maturecraft.CraftSection(level), note)

	out := make([]map[string]interface{}, 0, len(items))
	total := 0
	for _, it := range items {
		n := runeLen(it.body)
		total += n
		out = append(out, map[string]interface{}{
			"name": it.name, "runes": n, "note": it.note,
			"preview": truncateBudget(it.body, 80),
		})
	}
	out = append(out, map[string]interface{}{
		"name": "合计（不含 setting 槽）", "runes": total, "note": "各区段另受分项与总预算约束",
	})
	return out, nil
}
