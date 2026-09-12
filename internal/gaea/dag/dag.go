// Package dag 办公多文件流水线（阶段六 6.3，设计 docs/gaea-office-dag-63-design-2026-09.md）：
// goal+节点依赖图作为一等公民。真相=<cwd>/.gaea/work/dag/<id>.json（workspace 本地，
// 与 journal/rollback 同域，删档即弃不进用户库）。本包只做纯存储+纯函数
// （校验/分波/派生态）；执行在 App 层（gaea_dag.go）——节点=TaskTool.RunNew 的
// 全新子代理会话，产物=证据卡窗口增量，验收=人拍板回流记忆。
package dag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// 节点状态机：pending→running→done|failed→（steer 续跑→done|failed）→accepted。
// skipped=上游失败/终止级联未起跑；重跑把 accepted 退回 done（重跑即产物可能变，
// 验收失效诚实降级）。
const (
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusDone     = "done"
	StatusFailed   = "failed"
	StatusSkipped  = "skipped"
	StatusAccepted = "accepted"
	StatusHold     = "hold" // 高风险节点待审批（v4.243 审批分级；approve 后回 pending 放行）
)

// 风险分级（v4.243 审批分级，roadmap §12.4「危险操作分级审批」）：dag_plan 对
// 覆盖/删除既有文件、批量移动、全局性改动的节点标 high——执行器起跑前置闸
// （hold 待审批，人批准才跑）；normal 缺省不设闸。
const (
	RiskNormal = "normal"
	RiskHigh   = "high"
)

// 派生态（Derived）：不落盘，列表/详情实时折算——单一事实源在节点状态。
const (
	DerivedDraft    = "draft"    // 尚未起跑过
	DerivedRunning  = "running"  // 有节点在跑
	DerivedFailed   = "failed"   // 有失败（且无在跑）
	DerivedReady    = "ready"    // 无失败、全 done 待验收
	DerivedAccepted = "accepted" // 全节点已验收
)

// Node 一节=一次委托式文件加工（跨格式：读报表/透视/嵌图表/出 Word/导 PDF…）。
type Node struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Prompt     string   `json:"prompt"`
	DependsOn  []string `json:"dependsOn,omitempty"`
	Status     string   `json:"status"`
	Ref        string   `json:"ref,omitempty"`     // 最近一次子代理运行 ref（sa_ 前缀）
	Outputs    []string `json:"outputs,omitempty"` // 工作区相对路径（证据卡窗口增量）
	Error      string   `json:"error,omitempty"`
	RunCount   int      `json:"runCount"`
	SteerCount int      `json:"steerCount,omitempty"`
	AcceptedAt string   `json:"acceptedAt,omitempty"`
	// Risk 风险分级（normal 缺省/high 高风险=起跑前置审批闸）；
	// Approved=人已批准本次执行（approve 置位，改图变更节点归零=新指令重新批）。
	Risk     string `json:"risk,omitempty"`
	Approved bool   `json:"approved,omitempty"`
}

// Run 一条流水线。
type Run struct {
	ID        string `json:"id"`
	Goal      string `json:"goal"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Nodes     []Node `json:"nodes"`
}

// RunView = Run + 派生态（绑定面视图，json 与前端契约逐字一致）。
type RunView struct {
	Run
	Derived string `json:"derived"`
}

// dagSeq 进程内原子序号：NewID 追加它，保证同进程内 id 全局唯一（同秒重复
// 规划/并发测试互不撞名，SaveNew 的后缀逻辑退化为纯兜底）。
var dagSeq atomic.Uint64

// NewID 运行 id：时间戳+进程内序号 slug。
func NewID() string {
	return fmt.Sprintf("dag_%s_%d", time.Now().Format("20060102_150405"), dagSeq.Add(1))
}

// Validate 规划校验（dag_plan 落盘前必过）：goal/节点非空、id 唯一、依赖在场、无环。
// fail-closed：拒绝即报错，不截断不静默。
func Validate(goal string, nodes []Node) error {
	if strings.TrimSpace(goal) == "" {
		return fmt.Errorf("goal 不能为空")
	}
	if len(nodes) == 0 {
		return fmt.Errorf("至少规划一个节点")
	}
	ids := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if strings.TrimSpace(n.ID) == "" {
			return fmt.Errorf("节点 id 不能为空")
		}
		if ids[n.ID] {
			return fmt.Errorf("节点 id 重复: %s", n.ID)
		}
		if strings.TrimSpace(n.Title) == "" {
			return fmt.Errorf("节点 %s 的 title 不能为空", n.ID)
		}
		if strings.TrimSpace(n.Prompt) == "" {
			return fmt.Errorf("节点 %s 的 prompt 不能为空", n.ID)
		}
		if n.Risk != "" && n.Risk != RiskNormal && n.Risk != RiskHigh {
			return fmt.Errorf("节点 %s 的 risk 非法（只认 normal/high）", n.ID)
		}
		ids[n.ID] = true
	}
	for _, n := range nodes {
		for _, d := range n.DependsOn {
			if !ids[d] {
				return fmt.Errorf("节点 %s 依赖了不存在的 %s", n.ID, d)
			}
		}
	}
	if _, err := Order(nodes, nil); err != nil {
		return err
	}
	return nil
}

// Order 把待跑节点按依赖分波（波内顺序执行=产物归因窗口单调不重叠的首刀口径）。
// selected 为 nil/空 = 全部节点；selected 内部成环报错，selected 引用的外部依赖
// 视作已满足（调用方保证其 done/accepted）。
func Order(nodes []Node, selected map[string]bool) ([][]Node, error) {
	byID := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
	}
	if len(selected) == 0 {
		selected = make(map[string]bool, len(nodes))
		for _, n := range nodes {
			selected[n.ID] = true
		}
	}
	indeg := make(map[string]int, len(selected))
	dependents := make(map[string][]string)
	for id := range selected {
		n := byID[id]
		deg := 0
		for _, d := range n.DependsOn {
			if selected[d] {
				deg++
				dependents[d] = append(dependents[d], id)
			}
		}
		indeg[id] = deg
	}
	var waves [][]Node
	placed := 0
	for placed < len(selected) {
		var ids []string
		for id, deg := range indeg {
			if deg == 0 {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("依赖成环")
		}
		sort.Strings(ids)
		wave := make([]Node, 0, len(ids))
		for _, id := range ids {
			wave = append(wave, byID[id])
			indeg[id] = -1 // 本波占位，防重复入选
		}
		for _, n := range wave {
			for _, dep := range dependents[n.ID] {
				indeg[dep]--
			}
		}
		waves = append(waves, wave)
		placed += len(wave)
	}
	return waves, nil
}

// Derived 从节点状态折算 run 派生态（不落盘，单一事实源在节点）。
// 判序：有在跑=运行中；有失败=有失败（含 fail-closed/中断清扫，RunCount 可为
// 0）；从未起跑=草稿；无失败全终态=待验收/已验收。
func Derived(r Run) string {
	total := len(r.Nodes)
	accepted, runningN, failed, ran := 0, 0, 0, 0
	for _, n := range r.Nodes {
		switch n.Status {
		case StatusAccepted:
			accepted++
		case StatusRunning:
			runningN++
		case StatusFailed:
			failed++
		}
		if n.RunCount > 0 {
			ran++
		}
	}
	if runningN > 0 {
		return DerivedRunning
	}
	if failed > 0 {
		return DerivedFailed
	}
	if ran == 0 {
		return DerivedDraft
	}
	if accepted == total {
		return DerivedAccepted
	}
	return DerivedReady
}

// View 折算绑定面视图。
func View(r Run) RunView { return RunView{Run: r, Derived: Derived(r)} }

// MarkInterrupted 应用重启/进程退出后的懒清扫：running 节点已无人执行，诚实置
// failed（错误注明中断原因），由用户重跑续接——不假装还在跑。
func MarkInterrupted(r *Run, reason string) bool {
	changed := false
	for i := range r.Nodes {
		if r.Nodes[i].Status == StatusRunning {
			r.Nodes[i].Status = StatusFailed
			r.Nodes[i].Error = reason
			changed = true
		}
	}
	return changed
}

// EditReport 是一次增量改图的调和结果（人读摘要用）。
type EditReport struct {
	Kept       int // 原样保留（形状未变，状态/产物/运行痕迹延续）
	Updated    int // 形状有变或新增，回 pending（含撤销验收）
	Removed    int // 移除的节点数
	RevokedAcc int // 其中撤销了已验收节点的数量
}

// ApplyEdit 增量改图（v4.222）：新图（goal+全量节点）对既有 run 做原地调和。
// 语义（诚实可解释）：
//   - 运行中拒绝（Derived==running fail-closed，先终止再改）；
//   - 新图必须自身合法（Validate：非空/id 唯一/依赖在场/无环）——被移除节点
//     若仍被保留节点依赖，Validate 悬空依赖自然拒绝，不静默断链；
//   - 节点按 id 对账：id 在且 title/prompt/dependsOn 全等 → 原样保留（状态/
//     产物/运行痕迹延续）；id 在但有变、或新 id → 回 pending 全新节点；
//   - 已验收节点被改/被删 = 撤销验收（产物是旧指令的产出，新指令下不再成立），
//     RevokedAcc 计数交调用方透出，不静默。
func ApplyEdit(r Run, goal string, nodes []Node) (Run, EditReport, error) {
	if d := Derived(r); d == DerivedRunning {
		return Run{}, EditReport{}, fmt.Errorf("流水线运行中，先终止再改图")
	}
	if err := Validate(goal, nodes); err != nil {
		return Run{}, EditReport{}, err
	}
	old := make(map[string]Node, len(r.Nodes))
	for _, n := range r.Nodes {
		old[n.ID] = n
	}
	var rep EditReport
	reconciled := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		prev, existed := old[n.ID]
		sameShape := existed &&
			prev.Title == n.Title && prev.Prompt == n.Prompt && prev.Risk == n.Risk && sameDeps(prev.DependsOn, n.DependsOn)
		if sameShape {
			reconciled = append(reconciled, prev)
			rep.Kept++
			continue
		}
		if prev.Status == StatusAccepted {
			rep.RevokedAcc++
		}
		if existed {
			rep.Updated++
		} else {
			rep.Updated++ // 新增同计入 Updated（回 pending 的都算「改」）
		}
		reconciled = append(reconciled, Node{
			ID:        n.ID,
			Title:     n.Title,
			Prompt:    n.Prompt,
			DependsOn: n.DependsOn,
			Risk:      n.Risk,
			Status:    StatusPending,
		})
	}
	rep.Removed = len(r.Nodes) - (rep.Kept + rep.Updated)
	r.Goal = strings.TrimSpace(goal)
	r.Nodes = reconciled
	return r, rep, nil
}

// sameDeps 依赖列表等值比较（顺序敏感——dag_plan 落盘与模型回传同序，顺序
// 变化视为形状变化回 pending，宁可多改不可漏改）。
func sameDeps(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Store run 档文件存储（无状态；并发读改写由调用方串行化）。
type Store struct{ dir string }

func NewStore(dir string) *Store { return &Store{dir: dir} }

// safeID run id 必须是文件名安全 slug——Get/List 用 id 拼路径，拒绝穿越。
func safeID(id string) error {
	if id == "" || len(id) > 128 {
		return fmt.Errorf("非法 run id")
	}
	for _, r := range id {
		ok := r == '_' || r == '-' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !ok {
			return fmt.Errorf("非法 run id: %q", id)
		}
	}
	return nil
}

func (s *Store) path(id string) string { return filepath.Join(s.dir, id+".json") }

// Save 原子落盘（临时文件+改名），顺带刷新 UpdatedAt。
func (s *Store) Save(r Run) error {
	if err := safeID(r.ID); err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("创建流水线目录: %w", err)
	}
	r.UpdatedAt = time.Now().Format(time.RFC3339)
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path(r.ID) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(r.ID))
}

// SaveNew 唯一化落盘：id 冲突（同一秒重复规划）时追加 -2/-3 后缀，绝不覆盖
// 已有 run。用于 dag_plan 落盘路径；重跑/状态更新走 Save。
func (s *Store) SaveNew(r *Run) error {
	base := r.ID
	for i := 2; ; i++ {
		if _, err := os.Stat(s.path(r.ID)); os.IsNotExist(err) {
			break
		}
		r.ID = fmt.Sprintf("%s-%d", base, i)
	}
	return s.Save(*r)
}

// Get 读单条；不存在报错（fail-closed，不静默空值）。
func (s *Store) Get(id string) (Run, error) {
	if err := safeID(id); err != nil {
		return Run{}, err
	}
	b, err := os.ReadFile(s.path(id))
	if err != nil {
		return Run{}, fmt.Errorf("读取流水线 %s: %w", id, err)
	}
	var r Run
	if err := json.Unmarshal(b, &r); err != nil {
		return Run{}, fmt.Errorf("解析流水线 %s: %w", id, err)
	}
	return r, nil
}

// List 全部 run（创建时间倒序）；损坏文件跳过（只读兼容）。
func (s *Store) List() ([]Run, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		// 目录不存在=空集语义：nil slice 经 JSON 序列化成 null，会把前端
		// 非空断言的消费方（DagPanel tpls.length）炸掉（v4.234 真机走查实锤）。
		return []Run{}, nil
	}
	if err != nil {
		return nil, err
	}
	var runs []Run
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var r Run
		if json.Unmarshal(b, &r) == nil && r.ID != "" {
			runs = append(runs, r)
		}
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt > runs[j].CreatedAt })
	return runs, nil
}
