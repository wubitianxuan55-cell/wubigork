package app

// rewrite_scene.go — 场景章局部重写的选段→场景映射（v4.442 解禁）。
//
// blob 拼接语义与 syncBlobFromScenes 逐字对齐（"\n\n" join、List() 序、读失败
// 跳过）——编辑区展示的正文就是这个投影，选段偏移才有意义。提案时以场景真值
// 重算投影（不信磁盘 blob：旁路改场景文件可能失同步，以场景为准才配谈归属）。

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

const blobJoinSep = "\n\n" // 与 syncBlobFromScenes 同款分隔

type sceneSplicePlan struct {
	chapterNum int
	blob       string         // 场景真值重算的全章投影（= 编辑区视图口径）
	items      []*types.Scene // List() 序
	begins     []int          // 各场景内容在 blob 中的 rune 起
	owning     int            // 最近一次定位的归属场景下标；-1=未定位
}

// buildSceneSplicePlan 读场景真值并重算 blob 投影。无内容场景跳过（与 Stitch/
// syncBlobFromScenes 同口径）；全部为空时返回错误（没有可映射的对象）。
func buildSceneSplicePlan(pm *project.Manager, chapterNum int) (*sceneSplicePlan, error) {
	sm := pm.SceneManager(chapterNum)
	metas, err := sm.List()
	if err != nil {
		return nil, fmt.Errorf("读取场景列表失败: %w", err)
	}
	plan := &sceneSplicePlan{chapterNum: chapterNum, owning: -1}
	var parts []string
	for i := range metas {
		sc, rerr := sm.Read(metas[i].ID)
		if rerr != nil {
			continue
		}
		plan.items = append(plan.items, sc)
		plan.begins = append(plan.begins, len([]rune(strings.Join(parts, blobJoinSep))))
		parts = append(parts, sc.Content)
	}
	if len(plan.items) == 0 {
		return nil, fmt.Errorf("场景内容全部为空，无法定位选段归属")
	}
	plan.blob = strings.Join(parts, blobJoinSep)
	return plan, nil
}

// locate 返回 [start,end) 完整落入的场景下标；跨场景/落分隔区返回 -1。
func (p *sceneSplicePlan) locate(start, end int) int {
	for i, sc := range p.items {
		begin := p.begins[i]
		finish := begin + len([]rune(sc.Content))
		if start >= begin && end <= finish {
			return i
		}
	}
	return -1
}

// checkSpan 选段必须完整落在单个场景内，否则如实拒绝并点名两端场景。
func (p *sceneSplicePlan) checkSpan(start, end int) error {
	idx := p.locate(start, end)
	if idx >= 0 {
		p.owning = idx
		return nil
	}
	first, last := -1, -1
	for i, sc := range p.items {
		begin := p.begins[i]
		finish := begin + len([]rune(sc.Content))
		if start < finish && (first == -1 || i < first) {
			first = i
		}
		if end > begin && i > last {
			last = i
		}
	}
	if first == last || first == -1 || last == -1 {
		return fmt.Errorf("选段未落在任何场景内（可能选中了场景间分隔区），请重新选段")
	}
	return fmt.Errorf("选段跨越场景边界（「%s」→「%s」），请按场景分别重写",
		p.items[first].Meta.Title, p.items[last].Meta.Title)
}

// contextAround 场景内 ±budget rune 前后文（跨场景的前后文对本场景重写是噪声）。
func (p *sceneSplicePlan) contextAround(start, end, budget int) (string, string, error) {
	idx := p.locate(start, end)
	if idx < 0 {
		return "", "", fmt.Errorf("选段未落在任何场景内")
	}
	runes := []rune(p.items[idx].Content)
	s, e := start-p.begins[idx], end-p.begins[idx]
	before := runes[max(0, s-budget):s]
	after := runes[e:min(len(runes), e+budget)]
	return string(before), string(after), nil
}

// sceneOffsets 选段换算到场景内坐标（版本审计字段）。
func (p *sceneSplicePlan) sceneOffsets(start, end int) (int, int) {
	idx := p.locate(start, end)
	return start - p.begins[idx], end - p.begins[idx]
}

// splice 用新选段生成归属场景的新全文（场景级真值）。
func (p *sceneSplicePlan) splice(newSelected string, start, end int) (string, string) {
	idx := p.locate(start, end)
	original := p.items[idx].Content
	runes := []rune(original)
	s, e := start-p.begins[idx], end-p.begins[idx]
	updated := string(runes[:s]) + newSelected + string(runes[e:])
	return original, updated
}

// sceneID 归属场景 ID。
func (p *sceneSplicePlan) sceneID() string { return p.items[p.owning].Meta.ID }
