package agent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
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

// SubagentStatus enumerates the lifecycle states of a persisted sub-agent run.
type SubagentStatus string

const (
	SubagentRunning   SubagentStatus = "running"
	SubagentCompleted SubagentStatus = "completed"
	SubagentFailed    SubagentStatus = "failed"
)

// SubagentMeta is the sidecar metadata persisted next to the transcript JSONL.
type SubagentMeta struct {
	Ref       string         `json:"ref"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Status    SubagentStatus `json:"status"`
	ToolScope []string       `json:"toolScope,omitempty"`
	Model     string         `json:"model,omitempty"`
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
	dir string
}

// NewSubagentStore creates a store rooted at dir. Callers should ensure
// dir exists; the store creates the directory on first write.
func NewSubagentStore(dir string) *SubagentStore {
	return &SubagentStore{dir: dir}
}

// EphemeralSubagentRun returns a run with no backing store — it cannot be
// continued later. Use when store is nil or parent session is unknown.
func EphemeralSubagentRun(sysPrompt string) *SubagentRun {
	return &SubagentRun{
		Ref:     "",
		Session: session.New(sysPrompt),
	}
}

// refPrefix is the stable prefix for all sub-agent reference ids.
const refPrefix = "sa_"

// newRef generates a unique reference id: sa_YYYYMMDD_HHMMSS_nnnnnnnnnn_<hex>.
func newRef(now time.Time) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s%s_%010d_%s",
		refPrefix, now.UTC().Format("20060102_150405"), now.Nanosecond(), hex.EncodeToString(b))
}

// ── Lifecycle persistence ──────────────────────────────────────────

// MarkRunning writes the .meta sidecar with Status=running so a
// background sub-agent announces its existence before the transcript
// JSONL is written. Safe to call multiple times; idempotent.
func (s *SubagentStore) MarkRunning(run *SubagentRun) error {
	return s.saveMeta(run.Ref, SubagentMeta{
		Ref: run.Ref, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Status: SubagentRunning,
		Space:  spaces.Normalize(run.Space),
	})
}

// SaveCompleted persists the transcript JSONL and marks the run as completed.
func (s *SubagentStore) SaveCompleted(run *SubagentRun) error {
	if run.Session == nil || !run.Session.HasContent() {
		return nil // nothing to save
	}
	if err := run.Session.Save(s.transcriptPath(run.Ref)); err != nil {
		return fmt.Errorf("save transcript: %w", err)
	}
	return s.saveMeta(run.Ref, SubagentMeta{
		Ref: run.Ref, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Status: SubagentCompleted,
		Space:  spaces.Normalize(run.Space),
	})
}

// SaveFailed marks the run as failed without writing the transcript.
func (s *SubagentStore) SaveFailed(run *SubagentRun) error {
	return s.saveMeta(run.Ref, SubagentMeta{
		Ref: run.Ref, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Status: SubagentFailed,
		Space:  spaces.Normalize(run.Space),
	})
}

// ── Prepare entry points ────────────────────────────────────────────

// PrepareFresh creates a new sub-agent transcript. Caller must Release()
// after the run finishes. space 记录到 run 与子会话（S3 前瞻）：空值经
// Normalize 归一为 work，与 ctx 注入的缺省语义一致。
func (s *SubagentStore) PrepareFresh(sysPrompt, space string) (*SubagentRun, error) {
	ref := newRef(time.Now())
	sess := session.New(sysPrompt)
	sess.SetSpace(spaces.Normalize(space))
	return &SubagentRun{
		Ref:     ref,
		Session: sess,
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
	if !strings.HasPrefix(ref, refPrefix) {
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

	// S3 防穿越 C：请求空间与 ref 空间一致性校验（读端降级：空值 = work）。
	if refSpace := spaces.SpaceOr(meta.Space, spaces.SpaceWork); refSpace != spaces.Normalize(space) {
		return nil, fmt.Errorf("subagent %s space mismatch: recorded space %q != request space %q (fail-closed)", ref, refSpace, spaces.Normalize(space))
	}

	// Load the transcript.
	path := s.transcriptPath(ref)
	sess, err := session.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load transcript for subagent %s: %w", ref, err)
	}

	return &SubagentRun{
		Ref:     ref,
		Session: sess,
		Space:   spaces.Normalize(space),
		store:   s,
	}, nil
}

// ── Path helpers ────────────────────────────────────────────────────

func (s *SubagentStore) transcriptPath(ref string) string {
	return filepath.Join(s.dir, ref+".jsonl")
}

func (s *SubagentStore) metaPath(ref string) string {
	return filepath.Join(s.dir, ref+".meta.json")
}

// ── Meta persistence ────────────────────────────────────────────────

func (s *SubagentStore) saveMeta(ref string, meta SubagentMeta) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create subagents dir: %w", err)
	}
	meta.UpdatedAt = time.Now()
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(ref), b, 0o644)
}

func (s *SubagentStore) loadMeta(ref string) (SubagentMeta, error) {
	b, err := os.ReadFile(s.metaPath(ref))
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

// CleanupStaleRunning scans the store directory and marks any sub-agents
// still in "running" state as "failed" — they were interrupted by a
// crash or shutdown. Returns the count of cleaned-up entries.
func (s *SubagentStore) CleanupStaleRunning() (int, error) {
	entries, err := os.ReadDir(s.dir)
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
		meta, err := s.loadMeta(strings.TrimSuffix(e.Name(), ".meta.json"))
		if err != nil {
			continue
		}
		if meta.Status == SubagentRunning {
			meta.Status = SubagentFailed
			if err := s.saveMeta(meta.Ref, meta); err == nil {
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
