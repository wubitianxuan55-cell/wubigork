package agent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// ── Subagent transcript persistence (V10.29) ─────────────────────────
// Ported from Reasonix V1.15 (MIT). Enables task sub-agents to be
// continued across parent turns via continue_from, so long-running
// sub-tasks (exploration, research) can persist their context instead of
// starting from scratch each time.
//
// Each sub-agent produces two files under <sessionDir>/subagents/:
//   sa_YYYYMMDD_HHMMSS_nnnnnnnnnn_<hex>.jsonl     — transcript
//   sa_YYYYMMDD_HHMMSS_nnnnnnnnnn_<hex>.meta.json  — metadata sidecar
//
// The same store also persists "本地模型工具"（model-tool）runs so a tool that
// calls a local model (vision / summarize_file) is inspectable in the exact
// same session UI as a spawned sub-agent (kind=model_tool, ref prefix mt_):
//   mt_YYYYMMDD_HHMMSS_nnnnnnnnnn_<hex>.jsonl
//   mt_YYYYMMDD_HHMMSS_nnnnnnnnnn_<hex>.meta.json

// SubagentStatus enumerates the lifecycle states of a persisted sub-agent run.
type SubagentStatus string

const (
	SubagentRunning   SubagentStatus = "running"
	SubagentCompleted SubagentStatus = "completed"
	SubagentFailed    SubagentStatus = "failed"
)

// Run kind constants tag the two record families the store persists. Legacy
// sub-agent meta without a kind field reads as SubagentKindSubagent, so the
// read side never has to guess.
const (
	SubagentKindSubagent  = "subagent"
	SubagentKindModelTool = "model_tool"
)

// SubagentMeta is the sidecar metadata persisted next to the transcript JSONL.
type SubagentMeta struct {
	Ref       string         `json:"ref"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Status    SubagentStatus `json:"status"`
	// Kind 区分两类运行：subagent（task/run_skill 派生的真子代理）与
	// model_tool（vision/summarize_file 等本地模型工具的单轮调用）。旧 meta
	// 无此字段，读端按 subagent 降级。
	Kind string `json:"kind,omitempty"`
	// Title 是任务摘要（子代理 = 父调用 prompt；model_tool = 工具调用的人话
	// 摘要）。旧文件无该字段时由读端从 transcript 首条 user 消息推导。
	Title string `json:"title,omitempty"`
	// Tool 仅 model_tool 填写：触发记录的工具名（vision/summarize_file…）。
	Tool      string   `json:"tool,omitempty"`
	ToolScope []string `json:"toolScope,omitempty"`
	Model     string   `json:"model,omitempty"`
	// Space 是子代理的空间自描述（S3 双空间，设计 §1「子代理 transcript meta
	// 前瞻」）：work/play。旧 meta 无此字段 = 零值，读端按 work 降级
	// （spaces.SpaceOr）——与事件日志 space 字段同一兼容语义。
	Space string `json:"space,omitempty"`
}

// SubagentRun holds a prepared sub-agent transcript ready to execute.
// The caller MUST call Release() after the run finishes so the store
// can unlock the ref and allow concurrent reuse.
type SubagentRun struct {
	Ref     string
	Session *session.Session
	// Title 是任务摘要；落 meta.Title，UI 直接消费（model_tool 不可从
	// transcript 可靠推导标题，子代理旧数据则由读端回退首条 user 消息）。
	Title string
	// Tool 仅 model_tool 记录填写（记录发起工具名）。
	Tool string
	// Space 是本次子代理运行的空间（S3 前瞻）：prepare 时落定，meta
	// sidecar 写入时带上，续跑（continue_from）时用于一致性校验。
	Space   string
	store   *SubagentStore
	release func()
}

// Release unlocks the run's ref in the store so another caller can
// PrepareContinue it after it has been saved.
func (r *SubagentRun) Release() {
	if r.release != nil {
		r.release()
	}
}

// SubagentStore persists sub-agent transcripts to disk so they can be
// continued across parent turns via continue_from.
type SubagentStore struct {
	dir   string
	dirFn func() string
	// mu 串行化同一 store 上全部 transcript/meta 写（并行子代理各自写不同
	// 文件，但 ticker 快照与终态写不能互相覆盖状态）。
	mu sync.Mutex
}

// NewSubagentStore creates a store rooted at dir. Callers should ensure
// dir exists; the store creates the directory on first write.
func NewSubagentStore(dir string) *SubagentStore {
	return &SubagentStore{dir: dir}
}

// NewLazySubagentStore creates a store whose root directory is resolved on
// every write via dirFn. Boot uses this so the store follows the controller's
// current session (the same lazy resolution the event-log sink uses) instead
// of pinning one session directory for the whole process lifetime.
func NewLazySubagentStore(dirFn func() string) *SubagentStore {
	return &SubagentStore{dirFn: dirFn}
}

// EphemeralSubagentRun returns a run with no backing store — it cannot be
// continued later. Use when store is nil or parent session is unknown.
func EphemeralSubagentRun(sysPrompt string) *SubagentRun {
	return &SubagentRun{
		Ref:     "",
		Session: session.New(sysPrompt),
	}
}

// ref prefixes are the stable prefixes for persisted run references.
const (
	refPrefix       = "sa_"
	modelToolPrefix = "mt_"
)

// IsSubagentRef reports whether ref belongs to a spawned sub-agent (sa_).
func IsSubagentRef(ref string) bool { return strings.HasPrefix(ref, refPrefix) }

// IsModelToolRef reports whether ref belongs to a local-model tool run (mt_).
func IsModelToolRef(ref string) bool { return strings.HasPrefix(ref, modelToolPrefix) }

// ValidRunRef reports whether ref is a well-formed persisted run reference of
// any family (sa_ sub-agent or mt_ model-tool). The app binding layer reuses
// this to keep path-traversal protection single-sourced.
func ValidRunRef(ref string) bool {
	return validRefWithPrefix(ref, refPrefix) || validRefWithPrefix(ref, modelToolPrefix)
}

// validRefWithPrefix validates prefix + length + safe character set.
func validRefWithPrefix(ref, prefix string) bool {
	if len(ref) == 0 || len(ref) > 80 || !strings.HasPrefix(ref, prefix) {
		return false
	}
	for _, r := range ref {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '_' || r == '-' || r == '.'
		if !ok {
			return false
		}
	}
	return true
}

// newRef generates a unique reference id: <prefix>YYYYMMDD_HHMMSS_nnnnnnnnnn_<hex>.
func newRef(prefix string, now time.Time) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s%s_%010d_%s",
		prefix, now.UTC().Format("20060102_150405"), now.Nanosecond(), hex.EncodeToString(b))
}

// ── Lifecycle persistence ──────────────────────────────────────────

// baseMeta loads an existing sidecar (ignoring absence/corruption) so the
// write paths can preserve CreatedAt/Kind/Title/Tool/Space across status
// transitions instead of resetting them on every save.
func (s *SubagentStore) baseMeta(ref string) SubagentMeta {
	if m, err := s.loadMeta(ref); err == nil {
		return m
	}
	return SubagentMeta{Ref: ref}
}

// MarkRunning writes the .meta sidecar with Status=running so a
// background sub-agent announces its existence before the transcript
// JSONL is written. Safe to call multiple times; idempotent.
func (s *SubagentStore) MarkRunning(run *SubagentRun) error {
	if run == nil || run.Ref == "" {
		return errors.New("mark running: empty ref")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := s.baseMeta(run.Ref)
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.Ref = run.Ref
	meta.Status = SubagentRunning
	meta.Kind = kindOr(run.Kind(), meta.Kind)
	if run.Title != "" {
		meta.Title = run.Title
	}
	if run.Tool != "" {
		meta.Tool = run.Tool
	}
	meta.Space = spaces.Normalize(run.Space)
	return s.saveMetaUnlocked(meta)
}

// Kind returns the run's kind (default subagent for zero-value runs).
func (r *SubagentRun) Kind() string {
	if r == nil {
		return SubagentKindSubagent
	}
	if r.store != nil && IsModelToolRef(r.Ref) {
		return SubagentKindModelTool
	}
	return SubagentKindSubagent
}

// kindOr picks the first non-empty kind value.
func kindOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// SaveCompleted persists the transcript JSONL and marks the run as completed.
func (s *SubagentStore) SaveCompleted(run *SubagentRun) error {
	if run == nil || run.Ref == "" {
		return nil // ephemeral runs have nothing to persist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.Session == nil || !run.Session.HasContent() {
		// Mark the sidecar completed even without content so the status
		// transition is never lost (mirrors legacy behaviour which saved
		// nothing but kept running meta forever).
		return s.finalizeMetaUnlocked(run, SubagentCompleted)
	}
	if err := run.Session.Save(s.transcriptPathUnlocked(run.Ref)); err != nil {
		return fmt.Errorf("save transcript: %w", err)
	}
	return s.finalizeMetaUnlocked(run, SubagentCompleted)
}

// SaveFailed marks the run as failed. A partial transcript is persisted when
// the session already carries content so the UI can show what happened.
func (s *SubagentStore) SaveFailed(run *SubagentRun) error {
	if run == nil || run.Ref == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.Session != nil && run.Session.HasContent() {
		_ = run.Session.Save(s.transcriptPathUnlocked(run.Ref))
	}
	return s.finalizeMetaUnlocked(run, SubagentFailed)
}

// finalizeMetaUnlocked writes the terminal sidecar, preserving the run's
// creation timestamp and identity fields.
func (s *SubagentStore) finalizeMetaUnlocked(run *SubagentRun, status SubagentStatus) error {
	meta := s.baseMeta(run.Ref)
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.Ref = run.Ref
	meta.Status = status
	meta.Kind = kindOr(run.Kind(), meta.Kind)
	if run.Title != "" {
		meta.Title = run.Title
	}
	if run.Tool != "" {
		meta.Tool = run.Tool
	}
	meta.Space = spaces.Normalize(run.Space)
	return s.saveMetaUnlocked(meta)
}

// ── Prepare entry points ────────────────────────────────────────────

// PrepareFresh creates a new sub-agent transcript. Caller must Release()
// after the run finishes. space 记录到 run 与子会话（S3 前瞻）：空值经
// Normalize 归一为 work，与 ctx 注入的缺省语义一致。
func (s *SubagentStore) PrepareFresh(sysPrompt, space string) (*SubagentRun, error) {
	return s.prepareFresh(sysPrompt, space, SubagentKindSubagent, "")
}

// PrepareFreshWithTitle is PrepareFresh plus an explicit task summary stored
// in the sidecar (skill sub-agents pass the user's task text so the UI title
// does not start with the skill body).
func (s *SubagentStore) PrepareFreshWithTitle(sysPrompt, space, title string) (*SubagentRun, error) {
	return s.prepareFresh(sysPrompt, space, SubagentKindSubagent, title)
}

func (s *SubagentStore) prepareFresh(sysPrompt, space, kind, title string) (*SubagentRun, error) {
	_ = kind // kind 由 ref 前缀推导（sa_），保留参数以显式表达家族
	ref := newRef(refPrefix, time.Now())
	sess := session.New(sysPrompt)
	sess.SetSpace(spaces.Normalize(space))
	return &SubagentRun{
		Ref:     ref,
		Session: sess,
		Title:   title,
		Space:   spaces.Normalize(space),
		store:   s,
	}, nil
}

// PrepareContinue loads an existing sub-agent transcript by ref so the
// model can pick up where it left off. The ref must belong to a completed
// or failed run — a running ref cannot be continued concurrently.
// space 是发起续跑一侧的请求空间（SpaceFromContext）：与 ref meta 记录的
// 空间不一致时报错拒绝（S3 防穿越 C，fail-closed）——play 子代理的转录
// 不能被 work 会话续写，反之亦然。旧 meta 无 space 字段按 work 降级。
func (s *SubagentStore) PrepareContinue(ref, space string) (*SubagentRun, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("continue_from is empty")
	}
	if !validRefWithPrefix(ref, refPrefix) {
		return nil, fmt.Errorf("invalid subagent reference %q: must start with %q", ref, refPrefix)
	}

	// Load meta to verify the ref is valid and not currently running.
	meta, err := s.loadMeta(ref)
	if err != nil {
		return nil, fmt.Errorf("subagent %s not found: %w", ref, err)
	}
	if meta.Status == SubagentRunning {
		return nil, fmt.Errorf("subagent %s is still running; wait for it to complete before continuing", ref)
	}
	if meta.Kind == SubagentKindModelTool {
		return nil, fmt.Errorf("subagent %s is a model-tool run and cannot be continued", ref)
	}

	// S3 防穿越 C：请求空间与 ref 空间一致性校验（读端降级：空值 = work）。
	if refSpace := spaces.SpaceOr(meta.Space, spaces.SpaceWork); refSpace != spaces.Normalize(space) {
		return nil, fmt.Errorf("subagent %s space mismatch: recorded space %q != request space %q (fail-closed)", ref, refSpace, spaces.Normalize(space))
	}

	// Load the transcript.
	s.mu.Lock()
	path := s.transcriptPathUnlocked(ref)
	s.mu.Unlock()
	sess, err := session.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load transcript for subagent %s: %w", ref, err)
	}

	return &SubagentRun{
		Ref:     ref,
		Session: sess,
		Title:   meta.Title,
		Space:   spaces.Normalize(space),
		store:   s,
	}, nil
}

// ── Model-tool (本地模型工具) lifecycle ─────────────────────────────

// NewModelToolRun opens a model-tool record: running sidecar + transcript
// whose first user message is the human-readable tool-call label. The caller
// keeps only the returned ref and closes the record with FinishModelTool.
// tool 是发起工具名（vision / summarize_file …），label 是 UI 标题。
func (s *SubagentStore) NewModelToolRun(label, tool, space string) (*SubagentRun, error) {
	space = spaces.Normalize(space)
	ref := newRef(modelToolPrefix, time.Now())
	sess := session.New("")
	sess.SetSpace(space)
	if strings.TrimSpace(label) != "" {
		sess.Add(provider.Message{Role: provider.RoleUser, Content: strings.TrimSpace(label)})
	}
	run := &SubagentRun{
		Ref:     ref,
		Session: sess,
		Title:   strings.TrimSpace(label),
		Tool:    tool,
		Space:   space,
		store:   s,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := SubagentMeta{
		Ref: ref, CreatedAt: time.Now(), Status: SubagentRunning,
		Kind: SubagentKindModelTool, Title: run.Title, Tool: tool, Space: space,
	}
	if err := s.saveMetaUnlocked(meta); err != nil {
		return nil, err
	}
	if run.Session.HasContent() {
		if err := run.Session.Save(s.transcriptPathUnlocked(ref)); err != nil {
			return nil, fmt.Errorf("save model-tool transcript: %w", err)
		}
	}
	return run, nil
}

// UpdateModelToolTitle rewrites a running model-tool record's title/label to
// the fully-resolved tool-call description once the model has streamed the
// complete arguments (the record opens when only the tool name is known).
func (s *SubagentStore) UpdateModelToolTitle(ref, label string) error {
	if !validRefWithPrefix(ref, modelToolPrefix) {
		return fmt.Errorf("invalid model-tool reference %q", ref)
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.transcriptPathUnlocked(ref)
	if err := replaceFirstUserLine(path, label); err != nil {
		return err
	}
	meta := s.baseMeta(ref)
	meta.Ref = ref
	meta.Title = label
	if meta.Kind == "" {
		meta.Kind = SubagentKindModelTool
	}
	meta.UpdatedAt = time.Now()
	return s.saveMetaUnlocked(meta)
}

// FinishModelTool closes a model-tool record: appends the final output (or
// error) as the assistant line, then marks the sidecar completed/failed.
func (s *SubagentStore) FinishModelTool(ref, output string, toolErr error) error {
	if !validRefWithPrefix(ref, modelToolPrefix) {
		return fmt.Errorf("invalid model-tool reference %q", ref)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.transcriptPathUnlocked(ref)
	content := strings.TrimSpace(output)
	if toolErr != nil {
		content = "错误：" + toolErr.Error()
	}
	if err := appendMessageLine(path, provider.Message{
		Role: provider.RoleAssistant, Content: content,
	}); err != nil {
		return fmt.Errorf("append model-tool result: %w", err)
	}
	meta := s.baseMeta(ref)
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.Ref = ref
	meta.Status = SubagentCompleted
	if toolErr != nil {
		meta.Status = SubagentFailed
	}
	meta.Kind = SubagentKindModelTool
	meta.UpdatedAt = time.Now()
	return s.saveMetaUnlocked(meta)
}

// appendMessageLine appends one JSONL line to path (creating the file and
// directory as needed). Locking is the caller's responsibility.
func appendMessageLine(path string, m provider.Message) error {
	if path == "" {
		return errors.New("append: empty transcript path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	n, err := f.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return errors.New("short write on transcript append")
	}
	return nil
}

// replaceFirstUserLine rewrites the first user-message line of a JSONL
// transcript with label (keeps all other lines byte-identical).
func replaceFirstUserLine(path, label string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to relabel yet
		}
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m provider.Message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		if m.Role != provider.RoleUser {
			continue
		}
		m.Content = label
		upd, err := json.Marshal(m)
		if err != nil {
			return err
		}
		lines[i] = string(upd)
		break
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// ── Progress (live transcript) ─────────────────────────────────────

// progressFlushInterval is how often a running sub-agent's transcript
// snapshot is flushed to disk. The UI polls at 3s, so a 1s writer keeps
// the open thread visually current between model rounds.
const progressFlushInterval = time.Second

// TrackProgress starts a goroutine that snapshots run.Session to the
// transcript JSONL every interval and updates the running sidecar's
// UpdatedAt. The returned stop function performs a final flush and blocks
// until the goroutine exits — callers must invoke it before the terminal
// SaveCompleted/SaveFailed write, or the final meta could be overwritten by
// a stale running snapshot.
func (s *SubagentStore) TrackProgress(run *SubagentRun, interval time.Duration) func() {
	if run == nil || run.Ref == "" || run.Session == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = progressFlushInterval
	}
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-done:
				s.saveProgress(run)
				return
			case <-t.C:
				s.saveProgress(run)
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			<-exited
		})
	}
}

// saveProgress flushes one running transcript snapshot + meta touch.
func (s *SubagentStore) saveProgress(run *SubagentRun) {
	if run == nil || run.Ref == "" || run.Session == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.Session.HasContent() {
		if err := run.Session.Save(s.transcriptPathUnlocked(run.Ref)); err != nil {
			return
		}
	}
	meta := s.baseMeta(run.Ref)
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.Ref = run.Ref
	meta.Status = SubagentRunning
	meta.Kind = kindOr(run.Kind(), meta.Kind)
	if run.Title != "" {
		meta.Title = run.Title
	}
	if run.Tool != "" {
		meta.Tool = run.Tool
	}
	meta.Space = spaces.Normalize(run.Space)
	_ = s.saveMetaUnlocked(meta)
}

// ── Path helpers ────────────────────────────────────────────────────

func (s *SubagentStore) dirResolved() (string, error) {
	if s.dirFn != nil {
		d := strings.TrimSpace(s.dirFn())
		if d == "" {
			return "", errors.New("subagent store directory not resolved (session path empty)")
		}
		return d, nil
	}
	if s.dir == "" {
		return "", errors.New("subagent store directory is empty")
	}
	return s.dir, nil
}

func (s *SubagentStore) transcriptPathUnlocked(ref string) string {
	dir, err := s.dirResolved()
	if err != nil {
		return "" // callers surface their own error
	}
	return filepath.Join(dir, ref+".jsonl")
}

func (s *SubagentStore) metaPathUnlocked(ref string) string {
	dir, err := s.dirResolved()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, ref+".meta.json")
}

// metaPath 保留给同包测试/续跑校验：按当前目录解析（无锁读取安全）。
func (s *SubagentStore) metaPath(ref string) string {
	dir, err := s.dirResolved()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, ref+".meta.json")
}

// ── Meta persistence ────────────────────────────────────────────────

func (s *SubagentStore) saveMetaUnlocked(meta SubagentMeta) error {
	dir, err := s.dirResolved()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create subagents dir: %w", err)
	}
	meta.UpdatedAt = time.Now()
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, meta.Ref+".meta.json"), b, 0o644)
}

// saveMeta 是无锁写入的公开薄封装（ticker 外的单次写场景）。
func (s *SubagentStore) saveMeta(ref string, meta SubagentMeta) error {
	meta.Ref = ref
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveMetaUnlocked(meta)
}

func (s *SubagentStore) loadMeta(ref string) (SubagentMeta, error) {
	dir, err := s.dirResolved()
	if err != nil {
		return SubagentMeta{}, err
	}
	b, err := os.ReadFile(filepath.Join(dir, ref+".meta.json"))
	if err != nil {
		return SubagentMeta{}, err
	}
	var meta SubagentMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		return SubagentMeta{}, fmt.Errorf("corrupt meta: %w", err)
	}
	return meta, nil
}

// ── Startup cleanup ─────────────────────────────────────────────────

// CleanupStaleRunning scans the store directory and marks any runs still in
// "running" state as "failed" — they were interrupted by a crash or
// shutdown. Returns the count of cleaned-up entries.
func (s *SubagentStore) CleanupStaleRunning() (int, error) {
	dir, err := s.dirResolved()
	if err != nil {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		ref := strings.TrimSuffix(e.Name(), ".meta.json")
		meta, err := s.loadMeta(ref)
		if err != nil {
			continue
		}
		if meta.Status == SubagentRunning {
			meta.Status = SubagentFailed
			if err := s.saveMeta(ref, meta); err == nil {
				count++
			}
		}
	}
	return count, nil
}

// ── Result formatting ────────────────────────────────────────────────

// FormatSubagentReference returns the "Subagent reference: sa_xxx" line
// appended to a sub-agent result, so the parent model can cite the ref
// in a future continue_from call.
func FormatSubagentReference(run *SubagentRun) string {
	if run == nil || run.Ref == "" {
		return ""
	}
	return "\nSubagent reference: " + run.Ref
}
