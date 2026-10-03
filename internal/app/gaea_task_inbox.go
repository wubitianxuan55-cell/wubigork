package app

// 阶段七 7.3-1 任务收件箱（docs/gaea-stage7-plan-2026-09.md §3 / 规格 §2 线 B）：
// 四入口（意图中枢/微信/语音/Ctrl+K）的指令除即时执行外可「存为任务」→
// 状态机追踪（pending/doing/done/abandoned）。任务 = 用户意图的持久卡
// （title + 来源 + space + 状态机），不是后台执行队列——收件箱是清单不是
// 调度器；任务只在被打开/被操作时读写（零常驻：无事件、无轮询，前端每次
// 变更后重拉，清单量级 ≤500 读即所得）。
// 纯校验/归一/排序在 internal/taskinbox（线 A，零 IO 表驱动）；本文件只做
// 状态文件持久化（<DataRoot>/task_inbox.json，route_suggestions 同款容错读
// + temp+rename 原子写）与四个 App 绑定方法。意图执行层接线
// （execSaveTask）在 intent_router.go，走本文件同一 Save 内核。
// 纪律：space 维度必带（work/play 隔离，跨空间仅显式=本刀不做）；来源字段
// （Source/Action/Target/Session/CreatedAt）编辑不可变——审计链。

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/taskinbox"
)

// TaskInboxView 收件箱任务视图（Wails 绑定，前端 TaskInboxPanel 消费）。
// 字段与 internal/taskinbox.Task 完全同 json 标签（camelCase，与状态文件
// 契约一致；herdsman 防环先例——视图层不直接暴露内部包类型别名之外的形状）。
type TaskInboxView struct {
	ID        string           `json:"id"`                // "ti-"+12hex
	Title     string           `json:"title"`             // NormalizeTitle 产物
	Space     string           `json:"space"`             // "work"|"play"
	Status    taskinbox.Status `json:"status"`            // pending|doing|done|abandoned
	Source    string           `json:"source"`            // ctrlk|palette|voice|weixin|inbox
	Action    string           `json:"action,omitempty"`  // 意图动作（save_task/…；手动空）
	Target    string           `json:"target,omitempty"`  // 意图目标（任务标题文本等）
	Session   string           `json:"session,omitempty"` // 源会话（审计链）
	Note      string           `json:"note,omitempty"`    // 备注（唯一可编辑副字段）
	CreatedAt int64            `json:"createdAt"`         // 毫秒
	UpdatedAt int64            `json:"updatedAt"`         // 毫秒
}

// taskInboxToView 内部任务 → 绑定视图（字段一一对应）。
func taskInboxToView(tk taskinbox.Task) TaskInboxView {
	return TaskInboxView{
		ID: tk.ID, Title: tk.Title, Space: tk.Space, Status: tk.Status,
		Source: tk.Source, Action: tk.Action, Target: tk.Target,
		Session: tk.Session, Note: tk.Note,
		CreatedAt: tk.CreatedAt, UpdatedAt: tk.UpdatedAt,
	}
}

// ── 状态文件持久化（<DataRoot>/task_inbox.json）──────────────────────────
// 结构 {"version":1,"tasks":[…]}，Task json 标签 camelCase；load 容错
// （缺失/损坏回空表——收件箱丢得起，不该为一个坏文件挡住整个面板）、
// save 原子（temp+rename，route_suggestions 同款手法）。

type taskInboxFile struct {
	Version int              `json:"version"`
	Tasks   []taskinbox.Task `json:"tasks"`
}

// taskInboxPath 状态文件路径（<DataRoot>/task_inbox.json）。
func taskInboxPath(dataRoot string) string {
	return filepath.Join(dataRoot, "task_inbox.json")
}

// taskInboxMu 串行化收件箱状态文件的每一次「读→改→写」（P0-3）。
//
// 为什么必须包住整个 RMW 而不只是整写：状态文件是「内存切片全量 temp+rename
// 覆盖」模型，若两个写者各自 load 出同一份旧快照、各改各的、再先后 save，
// 后写者会用自己那份过期切片把先写者的新增/编辑/删除**静默盖掉**（丢卡），
// 因此 saveTaskInbox 的原子性只解决「半截文件」，解决不了「丢更新」——
// 丢更新只能靠把 load…save 整段放进同一临界区。写者来自多个入口
// （UI 面板 / Ctrl+K / 语音 execSaveTask / 微信通道），彼此是不同 goroutine，
// 只有本包级互斥能约束它们。
//
// 覆盖点（每个函数的整个 RMW 都在锁内）：
//   - (*App).taskInboxSave        —— 新建分支与编辑分支（含 MaxTasks 上限判定）
//   - (*App).GaeaTaskInboxSetStatus —— 读→查表→改状态→整写
//   - (*App).GaeaTaskInboxDelete  —— 读→删元素→整写
//
// 只读路径 (*App).GaeaTaskInboxList 取 RLock：它不写，但需要一份「不被本进程
// 写者切开的」一致视图（temp+rename 已保证它读到的文件本身总是完整的旧版或
// 新版；RLock 只是把「读盘 + 过滤排序」与本进程写者的临界区对齐）。RLock 不
// 引入额外分配，也不改变任何返回结构与可见性语义。
//
// 不覆盖（未解风险）：本锁是进程内的。双开 app（两个进程）写同一份
// task_inbox.json 仍是文件级竞态——temp+rename 保证不产生半截文件，但跨进程的
// 「丢更新」需要文件锁（如 O_EXCL 哨兵 / LockFileEx）或单实例约束，属另一刀。
var taskInboxMu sync.RWMutex

// loadTaskInbox 读收件箱任务；文件缺失/损坏一律回空表（恒非 nil，调用方
// 零判空）。
//
// 锁纪律：本函数是**不加锁的裸读原语**。任何「读→改→写」调用方必须先持有
// taskInboxMu 的写锁（否则改出来的切片会把并发写者的改动盖掉）；只想读快照
// 的调用方（GaeaTaskInboxList）自行取读锁。此处不再嵌套加锁，避免自死锁。
func loadTaskInbox(dataRoot string) []taskinbox.Task {
	b, err := os.ReadFile(taskInboxPath(dataRoot))
	if err != nil {
		return []taskinbox.Task{}
	}
	var f taskInboxFile
	if err := json.Unmarshal(b, &f); err != nil || f.Tasks == nil {
		return []taskinbox.Task{}
	}
	return f.Tasks
}

// saveTaskInbox 整写收件箱（temp+rename 原子落盘）。
//
// 锁纪律：本函数是**不加锁的裸写原语**——原子写只保证读者不会看到半截文件，
// 不保证并发「丢更新」。任何「读→改→写」调用方必须先持有 taskInboxMu 写锁，
// 并把 load…save 整段留在锁内（见 taskInboxMu 注释）。此处不再嵌套加锁。
func saveTaskInbox(dataRoot string, tasks []taskinbox.Task) error {
	if tasks == nil {
		tasks = []taskinbox.Task{}
	}
	b, err := json.MarshalIndent(taskInboxFile{Version: 1, Tasks: tasks}, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(taskInboxPath(dataRoot), b, 0o644)
}

// newTaskInboxID 生成任务 ID："ti-"+12 小写十六进制（crypto/rand 6 字节，
// 实体非重算候选——与 routesuggest.ParseSuggestionID 同款防御口径）。
// newTaskInboxID 身份口径见 tasks 包对照表（A2①）：单文件建议队列，短 id
// （ti-+12hex）碰撞域足够；rand 失败如实报错（与 tasks 的 nanos 兜底不同——
// 队列条目可重试生成，无兜底必要）。格式由 TestTaskInboxIDFormatPin 把守。
func newTaskInboxID() (string, error) {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "ti-" + hex.EncodeToString(buf[:]), nil
}

// taskInboxSaveInput Save 请求形状（reqJSON 反序列化目标；id 空=新建，
// 非空=编辑）。
type taskInboxSaveInput struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Space   string `json:"space"`
	Source  string `json:"source"`
	Action  string `json:"action"`
	Target  string `json:"target"`
	Session string `json:"session"`
	Note    string `json:"note"`
}

// taskInboxSave Save 内核（GaeaRouteIntent 的 execSaveTask 与绑定
// GaeaTaskInboxSave 共用——两路落同一状态文件，来源可区分）：
//   - id 空=新建：全部经纯包校验（NormalizeTitle/ValidSpace/ValidSource），
//     status=pending、CreatedAt/UpdatedAt=now、ID=ti-+12hex；超 MaxTasks
//     拒绝新建（防膨胀；编辑不受限）。source 缺省 "inbox"（手动新建兜底，
//     面板入口本就显式传——这里只兜不挡）。
//   - id 非空=编辑：只更新 title/note；Source/Action/Target/Session/
//     CreatedAt 不可变（来源审计链——任务从哪来不能被后续编辑洗白）。
func (a *App) taskInboxSave(in taskInboxSaveInput) (TaskInboxView, error) {
	// 临界区：load → 改切片（新建 append / 编辑就地改）→ save 整段独占，
	// 新建分支的 MaxTasks 判定也必须在锁内（否则并发新建可越过 500 上限）。
	taskInboxMu.Lock()
	defer taskInboxMu.Unlock()

	dataRoot := config.DataRoot()
	tasks := loadTaskInbox(dataRoot)
	now := time.Now().UnixMilli()

	if in.ID == "" {
		title, err := taskinbox.NormalizeTitle(in.Title)
		if err != nil {
			return TaskInboxView{}, &appError{err.Error()}
		}
		if !taskinbox.ValidSpace(in.Space) {
			return TaskInboxView{}, &appError{"非法空间 " + in.Space + "（仅 work|play）"}
		}
		source := strings.TrimSpace(in.Source)
		if source == "" {
			source = "inbox" // 手动新建兜底（判据①第五来源）
		}
		if !taskinbox.ValidSource(source) {
			return TaskInboxView{}, &appError{"非法来源 " + source + "（仅 ctrlk|palette|voice|weixin|inbox）"}
		}
		if len(tasks) >= taskinbox.MaxTasks {
			return TaskInboxView{}, &appError{fmt.Sprintf("任务收件箱已满（上限 %d 条），请先清理再新建", taskinbox.MaxTasks)}
		}
		id, err := newTaskInboxID()
		if err != nil {
			return TaskInboxView{}, fmt.Errorf("生成任务 ID 失败: %w", err)
		}
		pending, ok := taskinbox.ParseStatus("pending")
		if !ok {
			return TaskInboxView{}, &appError{"初始状态 pending 解析失败（内部不变量被破坏）"}
		}
		tk := taskinbox.Task{
			ID: id, Title: title, Space: in.Space, Status: pending,
			Source: source, Action: in.Action, Target: in.Target,
			Session: in.Session, Note: in.Note, CreatedAt: now, UpdatedAt: now,
		}
		tasks = append(tasks, tk)
		if err := saveTaskInbox(dataRoot, tasks); err != nil {
			return TaskInboxView{}, fmt.Errorf("写入任务收件箱失败: %w", err)
		}
		slog.Info("任务收件箱：新建任务", "id", id, "space", in.Space, "source", source)
		return taskInboxToView(tk), nil
	}

	// 编辑：id 形状先验（残渣防御，ParseSuggestionID 先例），找不到如实报错。
	if !taskinbox.ParseTaskID(in.ID) {
		return TaskInboxView{}, &appError{"任务 ID 不合法: " + in.ID}
	}
	for i := range tasks {
		if tasks[i].ID != in.ID {
			continue
		}
		title, err := taskinbox.NormalizeTitle(in.Title)
		if err != nil {
			return TaskInboxView{}, &appError{err.Error()}
		}
		tasks[i].Title = title // 只改 title/note；来源字段不可变（审计链）
		tasks[i].Note = in.Note
		tasks[i].UpdatedAt = now
		if err := saveTaskInbox(dataRoot, tasks); err != nil {
			return TaskInboxView{}, fmt.Errorf("写入任务收件箱失败: %w", err)
		}
		slog.Info("任务收件箱：更新任务", "id", in.ID)
		return taskInboxToView(tasks[i]), nil
	}
	return TaskInboxView{}, &appError{"任务不存在: " + in.ID}
}

// GaeaTaskInboxList 收件箱清单：读状态文件 → FilterBySpace（space 空=全部）
// → Sort（状态组序 pending<doing<done<abandoned，组内 UpdatedAt 降序）→
// 视图。文件缺失/损坏=空表（恒非 nil，前端零判空）；纯同步按需，零常驻。
func (a *App) GaeaTaskInboxList(space string) []TaskInboxView {
	// 只读路径：RLock 只求「读盘 + 过滤排序」与本进程写者临界区对齐的一致
	// 视图（零额外分配；返回值语义不变——仍是恒非 nil 的同步快照）。
	taskInboxMu.RLock()
	tasks := taskinbox.Sort(taskinbox.FilterBySpace(loadTaskInbox(config.DataRoot()), space))
	taskInboxMu.RUnlock()
	out := make([]TaskInboxView, 0, len(tasks))
	for _, tk := range tasks {
		out = append(out, taskInboxToView(tk))
	}
	return out
}

// GaeaTaskInboxSave 新建/编辑收件箱任务（reqJSON={id?,title,space,source?,
// action?,target?,session?,note?}）。语义见 taskInboxSave（单一内核）。
func (a *App) GaeaTaskInboxSave(reqJSON string) (TaskInboxView, error) {
	var in taskInboxSaveInput
	if err := json.Unmarshal([]byte(reqJSON), &in); err != nil {
		return TaskInboxView{}, &appError{"任务请求格式不正确: " + err.Error()}
	}
	return a.taskInboxSave(in)
}

// GaeaTaskInboxSetStatus 状态迁移：ParseStatus + CanTransition（同值短路
// 成功——无变化不落盘）；合法迁移 UpdatedAt=now。终态（done/abandoned）
// 不接受任何迁移（纯包矩阵的集成面）。
func (a *App) GaeaTaskInboxSetStatus(id, status string) (TaskInboxView, error) {
	if !taskinbox.ParseTaskID(id) {
		return TaskInboxView{}, &appError{"任务 ID 不合法: " + id}
	}
	to, ok := taskinbox.ParseStatus(status)
	if !ok {
		return TaskInboxView{}, &appError{"未知任务状态: " + status + "（pending|doing|done|abandoned）"}
	}
	// 临界区：读→查任务→CanTransition→改状态→整写；同值短路分支也留在锁内
	// （它不落盘，但必须基于本临界区的同一个快照判断，否则会读到陈旧状态）。
	taskInboxMu.Lock()
	defer taskInboxMu.Unlock()
	dataRoot := config.DataRoot()
	tasks := loadTaskInbox(dataRoot)
	for i := range tasks {
		if tasks[i].ID != id {
			continue
		}
		if tasks[i].Status == to {
			return taskInboxToView(tasks[i]), nil // 同值短路成功（无变化不落盘）
		}
		if !taskinbox.CanTransition(tasks[i].Status, to) {
			return TaskInboxView{}, &appError{fmt.Sprintf("任务状态不能从 %s 变为 %s", tasks[i].Status, to)}
		}
		tasks[i].Status = to
		tasks[i].UpdatedAt = time.Now().UnixMilli()
		if err := saveTaskInbox(dataRoot, tasks); err != nil {
			return TaskInboxView{}, fmt.Errorf("写入任务收件箱失败: %w", err)
		}
		slog.Info("任务收件箱：状态迁移", "id", id, "status", to)
		return taskInboxToView(tasks[i]), nil
	}
	return TaskInboxView{}, &appError{"任务不存在: " + id}
}

// GaeaTaskInboxDelete 删除收件箱任务（终态清理与误删回收同一路——收件箱
// 无回收站，删即删）。
func (a *App) GaeaTaskInboxDelete(id string) error {
	if !taskinbox.ParseTaskID(id) {
		return &appError{"任务 ID 不合法: " + id}
	}
	// 临界区：读→定位→切片删除→整写（避免并发编辑/新建被这份旧快照盖掉）。
	taskInboxMu.Lock()
	defer taskInboxMu.Unlock()
	dataRoot := config.DataRoot()
	tasks := loadTaskInbox(dataRoot)
	for i := range tasks {
		if tasks[i].ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			if err := saveTaskInbox(dataRoot, tasks); err != nil {
				return fmt.Errorf("写入任务收件箱失败: %w", err)
			}
			slog.Info("任务收件箱：删除任务", "id", id)
			return nil
		}
	}
	return &appError{"任务不存在: " + id}
}
