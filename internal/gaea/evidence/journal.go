package evidence

// ── v4.1 证据链（Apply→Verify→Journal）──────────────────────────────────
// 设计：docs/gaea-v41-evidence-chain-design.md。
// ChangeRecord 是一次「已应用的变更」证据卡：Before/After 存原文摘要
// （非展示截断文本，§15 补丁），供 Verifier 复核与回滚比对。
// 落点：<cwd>/.gaea/work/journal/<sessionKey>.jsonl（JSONL 追加，只读兼容）；
// 回合投影：<cwd>/.gaea/work/exports/journal/<sessionKey>/turn-<n>.md。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 摘要上限：Before/After 原文摘要统一截断（防 JSONL 膨胀；边界按字节）。
const SummaryLimit = 8 * 1024

// ChangeRecord 证据卡（设计 §3）。
type ChangeRecord struct {
	ID            string `json:"id"`
	SessionID     string `json:"sessionId"`
	Space         string `json:"space"` // 恒 "work"（play 不落证据链，红线）
	Turn          int    `json:"turn"`
	Tool          string `json:"tool"`   // edit_file / write_file / move_file / xlsx_apply …
	Target        string `json:"target"` // 工作区相对路径 / 单元格引用
	BeforeSummary string `json:"beforeSummary"`
	AfterSummary  string `json:"afterSummary"`
	Model         string `json:"model,omitempty"`
	At            int64  `json:"at"` // unix ms
	Status        string `json:"status"`
}

// Status 常量：Apply 后默认 pending_verify；Verifier（v4.1b）推进后续状态。
const (
	StatusPendingVerify = "pending_verify"
)

// ChangeLedger 每回合在内存收集证据卡（与 Ledger 同生命周期：回合开始 Reset）。
type ChangeLedger struct {
	mu      sync.Mutex
	changes []ChangeRecord
}

// NewChangeLedger 构造空台账。
func NewChangeLedger() *ChangeLedger { return &ChangeLedger{} }

// Reset 清空本回合证据卡。
func (l *ChangeLedger) Reset() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.changes = nil
}

// Add 追加一条证据卡。
func (l *ChangeLedger) Add(rec ChangeRecord) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.changes = append(l.changes, rec)
}

// Records 返回本回合证据卡快照（顺序保持）。
func (l *ChangeLedger) Records() []ChangeRecord {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ChangeRecord(nil), l.changes...)
}

type changeCtxKey struct{}

// WithChanges 把台账盖章进工具调用 ctx（execute_one 与 WithLedger 同位注入）。
func WithChanges(ctx context.Context, l *ChangeLedger) context.Context {
	return context.WithValue(ctx, changeCtxKey{}, l)
}

// ChangesFrom 读取 ctx 中的台账（无则 nil）。
func ChangesFrom(ctx context.Context) *ChangeLedger {
	v, _ := ctx.Value(changeCtxKey{}).(*ChangeLedger)
	return v
}

// RecordChange 写盘工具在变更成功后调用；ctx 无台账（非代理直调/dev/旧后端）
// 时静默跳过。Space/Turn/At/ID 由 agent 回合收尾统一补齐（flushJournal）。
func RecordChange(ctx context.Context, rec ChangeRecord) {
	l := ChangesFrom(ctx)
	if l == nil {
		return
	}
	if len(rec.BeforeSummary) > SummaryLimit {
		rec.BeforeSummary = rec.BeforeSummary[:SummaryLimit]
	}
	if len(rec.AfterSummary) > SummaryLimit {
		rec.AfterSummary = rec.AfterSummary[:SummaryLimit]
	}
	if rec.Status == "" {
		rec.Status = StatusPendingVerify
	}
	l.Add(rec)
}

// JournalStore 按会话 JSONL 追加存储 + 回合 markdown 投影。
type JournalStore struct {
	dir string
	mu  sync.Mutex
}

// OpenJournal 打开（不存在则创建）journal 目录。
func OpenJournal(dir string) (*JournalStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("journal dir empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &JournalStore{dir: dir}, nil
}

// sessionKeyOf 清洗会话标识为安全文件名（路径分隔符/通配符等替换为 _）。
func sessionKeyOf(sessionID string) string {
	r := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_", " ", "_",
	)
	return r.Replace(sessionID)
}

// Append 追加一条证据卡到该会话的 JSONL。
func (s *JournalStore) Append(rec ChangeRecord) error {
	if s == nil {
		return nil
	}
	if rec.Space == "" {
		rec.Space = "work"
	}
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("%s-%d-%d", sessionKeyOf(rec.SessionID), rec.Turn, time.Now().UnixMilli())
	}
	if rec.At == 0 {
		rec.At = time.Now().UnixMilli()
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(filepath.Join(s.dir, sessionKeyOf(rec.SessionID)+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// List 返回某会话全部证据卡（旧→新）。
func (s *JournalStore) List(sessionID string) ([]ChangeRecord, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(filepath.Join(s.dir, sessionKeyOf(sessionID)+".jsonl"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ChangeRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var rec ChangeRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue // 单行损坏跳过（只读兼容：旧行格式演进）
		}
		out = append(out, rec)
	}
	return out, nil
}

// WriteTurnMarkdown 把一回合证据卡投影为可读 markdown（审计导出）。
// 返回写入的文件路径（绝对）。exportsJournalDir = <exports>/journal。
func (s *JournalStore) WriteTurnMarkdown(exportsJournalDir, sessionID string, turn int, recs []ChangeRecord) (string, error) {
	if len(recs) == 0 {
		return "", nil
	}
	dir := filepath.Join(exportsJournalDir, sessionKeyOf(sessionID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# 回合证据 Journal\n\n")
	b.WriteString(fmt.Sprintf("- 会话：`%s`\n- 回合：%d\n- 空间：work\n- 证据卡数：%d\n", sessionID, turn, len(recs)))
	b.WriteString("\n---\n")
	for i, rec := range recs {
		b.WriteString(fmt.Sprintf("\n## %d. %s → %s\n\n", i+1, rec.Tool, rec.Target))
		b.WriteString(fmt.Sprintf("- 状态：`%s`\n- 时间：%s\n", rec.Status, time.UnixMilli(rec.At).Format("2006-01-02 15:04:05")))
		if rec.Model != "" {
			b.WriteString(fmt.Sprintf("- 模型：`%s`\n", rec.Model))
		}
		b.WriteString("\n### 变更前（原文摘要）\n\n```\n")
		b.WriteString(rec.BeforeSummary)
		b.WriteString("\n```\n\n### 变更后（原文摘要）\n\n```\n")
		b.WriteString(rec.AfterSummary)
		b.WriteString("\n```\n")
	}
	path := filepath.Join(dir, fmt.Sprintf("turn-%d.md", turn))
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
