// schedule/baseline.go — 计划基线快照与漂移对比（v4.116.0 刀7）。
//
// 与前端 frontend/src/schedule/baseline.ts 互为镜像（同批场景同批期望值）：
// 把当前排程结果固化为基线，之后每次调整都可量化偏差——总工期漂移、任务
// 推移、关键链进出。基线只存叶任务；行值是工作日序号，对比口径=相对开工日
// 的偏移，不受开工日调整影响。
package schedule

import (
	"fmt"
	"sort"
	"strings"
)

// SnapshotBaseline 保存基线：以当前计划 CPM 结果固快照。守卫 fail-closed——
// 循环依赖或无叶任务时报错，绝不存出无意义的空基线。savedAt 由调用方标注
// （ops 通道=工具层盖时间戳；测试传固定值）。Dur 存等效工作日跨度 EF−ES
// （v4.150 双工期刀1 换源：wd 任务=effDur 逐位不变；cd 任务=养护窗口内
// 工作日数），漂移仍在工作日空间对比。
func SnapshotBaseline(p *Project, savedAt, name string) (*Baseline, error) {
	cpm := ComputeCpmCal(p.Tasks, p.Links, p.Calendar, p.StartDate)
	if !cpm.OK {
		return nil, fmt.Errorf("%s", cpm.Error)
	}
	rows := make(map[string]BaselineRow, len(p.Tasks))
	for _, t := range p.Tasks {
		if t.Level == 0 {
			continue
		}
		row, ok := cpm.Rows[t.ID]
		if !ok {
			continue
		}
		rows[t.ID] = BaselineRow{Name: t.Name, ES: row.ES, EF: row.EF, Dur: row.EF - row.ES, Critical: row.Critical}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("计划中没有叶任务，无可固化的基线")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "基线"
	}
	return &Baseline{Name: name, SavedAt: savedAt, Duration: cpm.Duration, Rows: rows}, nil
}

// DriftNow 基线漂移行的当前排程值。
type DriftNow struct {
	ES  int `json:"es"`
	EF  int `json:"ef"`
	Dur int `json:"dur"`
}

// DriftRow 基线漂移行：只含有偏差的任务（shifted=推移 / added=新增 / removed=已移除）。
type DriftRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	// Base 基线行（新增任务为 nil）。
	Base *BaselineRow `json:"base"`
	// Now 当前排程（已移除任务为 nil）。
	Now      *DriftNow `json:"now"`
	ESDrift  int       `json:"esDrift"`
	EFDrift  int       `json:"efDrift"`
	DurDrift int       `json:"durDrift"`
	// CriticalNow/CriticalBase 当前/基线时是否关键。
	CriticalNow  bool `json:"criticalNow"`
	CriticalBase bool `json:"criticalBase"`
}

// Drift 基线漂移对比汇总。
type Drift struct {
	BaselineName     string `json:"baselineName"`
	BaselineSavedAt  string `json:"baselineSavedAt"`
	BaselineDuration int    `json:"baselineDuration"`
	CurrentDuration  int    `json:"currentDuration"`
	// DurationDrift 总工期漂移（正=拖后，负=提前）。
	DurationDrift int `json:"durationDrift"`
	SameCount     int `json:"sameCount"`
	ShiftedCount  int `json:"shiftedCount"`
	AddedCount    int `json:"addedCount"`
	RemovedCount  int `json:"removedCount"`
	// CriticalGained/CriticalLost 新进入/退出关键线路的任务名。
	CriticalGained []string   `json:"criticalGained"`
	CriticalLost   []string   `json:"criticalLost"`
	Rows           []DriftRow `json:"rows"`
}

// ComputeBaselineDrift 基线漂移对比。无基线或计划未通过 CPM 返回 nil；
// Rows 只含有偏差行（一致行不进列表，数量在 SameCount），顺序=当前任务表
// 序，移除行按 id 排序（跨语言镜像一致）。
func ComputeBaselineDrift(p *Project, cpm CpmResult) *Drift {
	base := p.Baseline
	if base == nil || !cpm.OK {
		return nil
	}
	d := &Drift{
		BaselineName:     base.Name,
		BaselineSavedAt:  base.SavedAt,
		BaselineDuration: base.Duration,
		CurrentDuration:  cpm.Duration,
		DurationDrift:    cpm.Duration - base.Duration,
	}
	seen := make(map[string]bool, len(p.Tasks))
	for _, t := range p.Tasks {
		if t.Level == 0 {
			continue
		}
		row, ok := cpm.Rows[t.ID]
		if !ok {
			continue
		}
		seen[t.ID] = true
		// Dur=等效工作日跨度 EF−ES（v4.150 双工期刀1 换源，wd 逐位不变）
		now := DriftNow{ES: row.ES, EF: row.EF, Dur: row.EF - row.ES}
		b, has := base.Rows[t.ID]
		if !has {
			d.AddedCount++
			d.Rows = append(d.Rows, DriftRow{
				ID: t.ID, Name: t.Name, Kind: "added", Now: &now,
				ESDrift: row.ES, EFDrift: row.EF, DurDrift: now.Dur,
				CriticalNow: row.Critical,
			})
			if row.Critical {
				d.CriticalGained = append(d.CriticalGained, t.Name)
			}
			continue
		}
		esD, efD, durD := row.ES-b.ES, row.EF-b.EF, now.Dur-b.Dur
		shifted := esD != 0 || efD != 0 || durD != 0 || row.Critical != b.Critical
		if !shifted {
			d.SameCount++
			continue
		}
		d.ShiftedCount++
		bc := b
		d.Rows = append(d.Rows, DriftRow{
			ID: t.ID, Name: t.Name, Kind: "shifted", Base: &bc, Now: &now,
			ESDrift: esD, EFDrift: efD, DurDrift: durD,
			CriticalNow: row.Critical, CriticalBase: b.Critical,
		})
		if row.Critical && !b.Critical {
			d.CriticalGained = append(d.CriticalGained, t.Name)
		}
		if !row.Critical && b.Critical {
			d.CriticalLost = append(d.CriticalLost, t.Name)
		}
	}
	removed := make([]string, 0)
	for id := range base.Rows {
		if !seen[id] {
			removed = append(removed, id)
		}
	}
	sort.Strings(removed)
	for _, id := range removed {
		b := base.Rows[id]
		d.RemovedCount++
		d.Rows = append(d.Rows, DriftRow{
			ID: id, Name: b.Name, Kind: "removed", Base: &b,
			ESDrift: -b.ES, EFDrift: -b.EF, DurDrift: -b.Dur,
			CriticalBase: b.Critical,
		})
	}
	return d
}
