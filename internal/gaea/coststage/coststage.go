// Package coststage 五算对比(估/概/预/结/决):五算阶段值的存储、标准对比表
// 与相邻阶段偏差特征提取。
//
// 五算 = 投资估算/设计概算/施工图预算/竣工结算/竣工决算,是建设项目造价在
// 不同阶段的呈现;本项目用 cost_stage_values 表按 (project_id, stage) 唯一保存
// 各阶段金额,project_id 对应 cost_projects.id(本项目内不做外键约束)。对比表与
// 偏差提取均为纯函数、本地计算,LLM 仅做偏差诊断文案(本包提供规则文案兜底)。
// 存储于 Hephaestus.db;表在包内幂等自建(CREATE TABLE IF NOT EXISTS),正式迁移
// 由父代理收编进 db/schema.go,本包不改动该文件。
package coststage

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// 五算阶段常量(按行业口径:投资估算/设计概算/施工图预算/竣工结算/竣工决算)。
const (
	StageEstimate   = "估算" // 投资估算
	StageDesign     = "概算" // 设计概算
	StageBudget     = "预算" // 施工图预算
	StageSettlement = "结算" // 竣工结算
	StageFinal      = "决算" // 竣工决算
)

// StageOrder 五算标准顺序(对比表/列表统一按此顺序输出)。
var StageOrder = []string{StageEstimate, StageDesign, StageBudget, StageSettlement, StageFinal}

// 偏差阈值(导出供测试与前端调参)。
const (
	// DeviationNormalPct 正常档上限:|差幅%| 小于该值视为正常波动。
	DeviationNormalPct = 5.0
	// DeviationAlertPct 关注档上限:|差幅%| 不超过该值视为需关注,超过视为异常。
	DeviationAlertPct = 15.0
	// levelEps 阈值比较容差:吸收浮点舍入误差,保证边界值(恰好 5%/15%)
	// 归入「关注」档而非因 1e-15 级误差落入相邻档位。
	levelEps = 1e-9
)

// StageValue 五算阶段值(project_id 对应 cost_projects.id)。
type StageValue struct {
	ID        int64
	ProjectID string
	Stage     string
	Amount    float64
	Date      string // YYYY-MM-DD
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CompareRow 对比行:阶段金额 + 环比(相对上一阶段) + 累计差(相对估算) + 差幅%。
type CompareRow struct {
	Stage         string
	Amount        float64
	HasValue      bool    // 该阶段是否有值(缺阶段为 false,Amount=0)
	PrevStage     string  // 上一有值阶段(用于环比);无则空
	HasPrev       bool
	ChainDiff     float64 // 环比差额(本阶段 - 上一有值阶段)
	ChainDiffPct  float64 // 环比差幅%((本-上)/上*100;上<=0 时为 0)
	BaseDiff      float64 // 累计差(相对估算/首个有值阶段)
	BaseDiffPct   float64
}

// Deviation 相邻有值阶段的偏差特征(供前端展示与复盘笔记 AI 诊断输入)。
type Deviation struct {
	FromStage  string
	ToStage    string
	FromAmount float64
	ToAmount   float64
	Diff       float64
	DiffPct    float64
	Direction  string // 上升/下降(Diff>=0 为上升)
	Level      string // 正常(|pct|<5) / 关注(5<=|pct|<=15) / 异常(|pct|>15)
	Suggestion string // 规则文案(中文)
}

// 建表 SQL(对齐 docs/gaea-v42-cost-ai-design.md §3;包内幂等自建,正式迁移由
// 父代理收编进 db/schema.go)。UNIQUE(project_id, stage) 同时为 ListStages 的
// project_id 查询提供索引。
const createTableSQL = `
CREATE TABLE IF NOT EXISTS cost_stage_values (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL,
  stage      TEXT NOT NULL,              -- 估算/概算/预算/结算/决算
  amount     REAL NOT NULL,              -- 阶段金额（元）
  date       TEXT NOT NULL DEFAULT '',   -- 阶段日期 YYYY-MM-DD
  note       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(project_id, stage)
);
`

// Store 五算阶段值存储(Hephaestus.db)。
type Store struct {
	db *sql.DB
}

// Open 打开五算阶段值存储并幂等自建表;gdb 为 nil 或建表失败时返回不可用 store。
func Open(gdb *sql.DB) *Store {
	s := &Store{db: gdb}
	if gdb == nil {
		return s
	}
	if _, err := gdb.Exec(createTableSQL); err != nil {
		log.Printf("[coststage] 建表失败,五算对比不可用: %v", err)
		s.db = nil
	}
	return s
}

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

func (s *Store) requireDB() error {
	if s.db == nil {
		return fmt.Errorf("cost stage store unavailable")
	}
	return nil
}

// validStage 报告 stage 是否为五算阶段之一。
func validStage(stage string) bool {
	for _, st := range StageOrder {
		if st == stage {
			return true
		}
	}
	return false
}

// SaveStage 新建/更新五算阶段值:按 (project_id, stage) UPSERT(同项目同阶段重复
// 保存即更新金额/日期/备注,不新增行;created_at 保持首次录入值)。stage 必须
// ∈ StageOrder,否则报错。
func (s *Store) SaveStage(v StageValue) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	if strings.TrimSpace(v.ProjectID) == "" {
		return fmt.Errorf("五算阶段值需要项目 id")
	}
	if !validStage(v.Stage) {
		return fmt.Errorf("无效的五算阶段 %q,须为:估算/概算/预算/结算/决算", v.Stage)
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(`
INSERT INTO cost_stage_values(project_id, stage, amount, date, note, created_at, updated_at)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(project_id, stage) DO UPDATE SET
  amount=excluded.amount, date=excluded.date, note=excluded.note, updated_at=excluded.updated_at`,
		v.ProjectID, v.Stage, v.Amount, v.Date, v.Note,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

// ListStages 返回项目的五算阶段值(按 StageOrder 顺序,未录入的跳过;无数据返回空)。
func (s *Store) ListStages(projectID string) []StageValue {
	if err := s.requireDB(); err != nil {
		return nil
	}
	rows, err := s.db.Query(`
SELECT id, project_id, stage, amount, date, note, created_at, updated_at
FROM cost_stage_values WHERE project_id=?`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	byStage := make(map[string]StageValue)
	for rows.Next() {
		var v StageValue
		var created, updated string
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Stage, &v.Amount, &v.Date, &v.Note,
			&created, &updated); err != nil {
			continue
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		byStage[v.Stage] = v
	}
	out := make([]StageValue, 0, len(byStage))
	for _, st := range StageOrder {
		if v, ok := byStage[st]; ok {
			out = append(out, v)
		}
	}
	return out
}

// DeleteStage 删除项目的某个五算阶段值(不存在的阶段为无操作)。
func (s *Store) DeleteStage(projectID, stage string) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	if !validStage(stage) {
		return fmt.Errorf("无效的五算阶段 %q,须为:估算/概算/预算/结算/决算", stage)
	}
	_, err := s.db.Exec("DELETE FROM cost_stage_values WHERE project_id=? AND stage=?", projectID, stage)
	return err
}

// ── 对比与偏差(纯函数,本地计算)────────────────────────────────────────

// ComputeComparison 纯函数:输入阶段值,按 StageOrder 补全缺失阶段(Amount=0,
// HasValue=false),输出对比表(顺序=StageOrder,固定 5 行)。少于 2 个有值阶段
// 返回 nil。环比以「上一有值阶段」为基准;累计差以「首个有值阶段」为基准;
// 金额 <=0 的阶段不算有值。
func ComputeComparison(values []StageValue) []CompareRow {
	byStage := make(map[string]float64, len(values))
	for _, v := range values {
		if v.Amount > 0 {
			byStage[v.Stage] = v.Amount
		}
	}
	// 按 StageOrder 收集有值阶段(不足 2 个不构成对比)。
	valued := make([]string, 0, len(StageOrder))
	for _, st := range StageOrder {
		if _, ok := byStage[st]; ok {
			valued = append(valued, st)
		}
	}
	if len(valued) < 2 {
		return nil
	}
	base := byStage[valued[0]]
	rows := make([]CompareRow, len(StageOrder))
	var prev string // 上一有值阶段
	for i, st := range StageOrder {
		row := CompareRow{Stage: st}
		if amt, ok := byStage[st]; ok {
			row.Amount = amt
			row.HasValue = true
			if prev != "" {
				pAmt := byStage[prev]
				row.PrevStage = prev
				row.HasPrev = true
				row.ChainDiff = amt - pAmt
				row.ChainDiffPct = pctChange(amt, pAmt)
			}
			prev = st
			row.BaseDiff = amt - base
			row.BaseDiffPct = pctChange(amt, base)
		}
		rows[i] = row
	}
	return rows
}

// ExtractDeviations 纯函数:从对比行提取相邻有值阶段的偏差(每对相邻有值阶段
// 产出一条);少于 2 个有值阶段返回 nil。
func ExtractDeviations(rows []CompareRow) []Deviation {
	var out []Deviation
	var prev *CompareRow
	for i := range rows {
		r := &rows[i]
		if !r.HasValue {
			continue
		}
		if prev != nil {
			diff := r.Amount - prev.Amount
			pct := pctChange(r.Amount, prev.Amount)
			out = append(out, Deviation{
				FromStage:  prev.Stage,
				ToStage:    r.Stage,
				FromAmount: prev.Amount,
				ToAmount:   r.Amount,
				Diff:       diff,
				DiffPct:    pct,
				Direction:  directionOf(diff),
				Level:      classifyLevel(pct),
				Suggestion: deviationSuggestion(prev.Stage, r.Stage, pct),
			})
		}
		prev = r
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// pctChange 差幅百分比:(cur-base)/base*100;基准金额 <=0 时返回 0(除零保护)。
func pctChange(cur, base float64) float64 {
	if base <= 0 {
		return 0
	}
	return (cur - base) / base * 100
}

// directionOf 偏差方向:Diff>=0 为上升,否则下降。
func directionOf(diff float64) string {
	if diff >= 0 {
		return "上升"
	}
	return "下降"
}

// classifyLevel 偏差档位:|pct|<5 正常;5<=|pct|<=15 关注;|pct|>15 异常。
func classifyLevel(pct float64) string {
	ap := math.Abs(pct)
	switch {
	case ap < DeviationNormalPct-levelEps:
		return "正常"
	case ap <= DeviationAlertPct+levelEps:
		return "关注"
	default:
		return "异常"
	}
}

// deviationSuggestion 按偏差档位生成规则文案(中文),如
// 「预算较概算 +8.1%,建议核查工程量或单价差异」。
func deviationSuggestion(from, to string, pct float64) string {
	suffix := "处于正常波动范围"
	switch classifyLevel(pct) {
	case "关注":
		suffix = "建议核查工程量或单价差异"
	case "异常":
		suffix = "异常偏离,建议核查变更签证与调价依据"
	}
	return fmt.Sprintf("%s较%s %+.1f%%,%s", to, from, pct, suffix)
}
