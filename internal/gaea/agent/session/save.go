package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// Save writes the session's messages to path in JSONL — one provider.Message
// per line — so a user can resume the conversation later. The file is
// rewritten in full on every save: chat sessions are small (kilobytes), and
// append-only would have to be reconciled with the compaction pass that
// mutates the middle of session.Messages.
func (s *Session) Save(path string) error {
	if s.IsEventMode() {
		return s.saveEventMode(path)
	}
	// Encode the whole JSONL into memory, then write atomically — a crash
	// mid-write can't leave a partial JSONL that won't reload.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, m := range s.Snapshot() { // copy under the lock — a turn may be appending
		if err := enc.Encode(m); err != nil {
			return fmt.Errorf("encode message: %w", err)
		}
	}
	return fileutil.AtomicWrite(path, buf.Bytes(), 0o644)
}

// saveEventMode 是事件日志模式下的 Save：legacy 镜像（整文件 JSONL，与旧行为
// 逐字节一致，供 GaeaListSessions/旧工具读取）+ 确保事件日志存在。日志缺失时
// 由当前消息迁移生成（幂等；已存在时不动——运行期由事件 sink 持续追加，
// 日志本身是恢复/派生的真相源）。
func (s *Session) saveEventMode(path string) error {
	if path == "" {
		return fmt.Errorf("empty session path")
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, m := range s.Snapshot() {
		if err := enc.Encode(m); err != nil {
			return fmt.Errorf("encode message: %w", err)
		}
	}
	if err := fileutil.AtomicWrite(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	logPath := LogPathFor(path)
	if logPath == "" || HasEventLog(path) {
		// 无日志路径（空会话路径）或日志已存在（sink 已持续写入）→ 不动日志。
		return nil
	}
	if _, err := MigrateLegacyToLog(logPath, path); err != nil {
		return fmt.Errorf("event save: migrate log: %w", err)
	}
	return nil
}

// Load reads a JSONL file written by Save into a fresh Session value.
// Missing files surface as os.IsNotExist so callers can fall through to a
// new session.
func Load(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &Session{}
	// Decode a stream of JSON values rather than scanning lines: a single
	// message (e.g. a multi-MiB bash output) can exceed any line-buffer cap, and
	// Save's json.Encoder has no such limit — a Scanner here made sessions that
	// saved fine fail to reload.
	dec := json.NewDecoder(f)
	for {
		var m provider.Message
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		s.Messages = append(s.Messages, m)
	}
	return s, nil
}

// LoadWithFormat 按持久化格式加载会话。logFormat="event" 时优先从事件日志
// Restore（checkpoint 消息 + seq 之后的重放投影，torn-tail 自动修复），
// 无日志时回退 legacy Load（含 os.IsNotExist 语义）。旧格式会话的迁移由
// 调用方（控制器 ResumeFromDisk）先 DetectLegacy → MigrateLegacyToLog 完成，
// 本函数不重复迁移。其余格式（含缺省 ""）与 Load 行为完全一致。
func LoadWithFormat(path, logFormat string) (*Session, error) {
	if !strings.EqualFold(logFormat, "event") || !HasEventLog(path) {
		return Load(path)
	}
	msgs, last, err := Restore(CheckpointPathFor(path), LogPathFor(path))
	if err != nil {
		return nil, err
	}
	return NewFromRestore(msgs, last, "event"), nil
}

// LastLogSeq 返回事件日志最后一条完整条目的 seq（无日志/空日志/损坏返回 0）。
// 检查点 flush 用它确定「已消费 seq」；不做 torn-tail 修复（调用时机在回合
// 边界，日志写入器已关闭，读取与写入互不干扰）。
func LastLogSeq(logPath string) int64 {
	entries, err := ReadLog(logPath)
	if err != nil {
		return 0
	}
	return lastLogSeq(entries)
}

// AppendUserMessage 把一条用户消息追加到事件日志（「模型可见必入日志」：
// 运行期 user_message 由控制器在模型调用前落盘）。日志不存在时自动创建
// （legacyPath 非空且旧会话文件存在时先迁移旧消息）。返回新条目 seq。
func AppendUserMessage(logPath, legacyPath, content string) (int64, error) {
	return appendMessageLog(logPath, legacyPath, userLogPayload{Content: content}, KindUserMessage)
}

// AppendSystemMessage 把一条 system 消息追加到事件日志（中断摘要注入等）。
// 返回新条目 seq。
func AppendSystemMessage(logPath, legacyPath, content string) (int64, error) {
	return appendMessageLog(logPath, legacyPath, userLogPayload{Content: content}, KindSystemMessage)
}

// appendMessageLog 打开（必要时创建/迁移）日志并追加一条消息级事件。
func appendMessageLog(logPath, legacyPath string, payload any, kind string) (int64, error) {
	if logPath == "" {
		return 0, errors.New("empty log path")
	}
	w, err := OpenLog(logPath, legacyPath)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	return w.Append(kind, payload)
}

// Info summarises a saved session for the --resume picker: where it
// is on disk, when it was last touched, the first user message as a preview,
// and a rough turn count.
type Info struct {
	Path    string
	ModTime time.Time
	Preview string
	Turns   int
}

// List returns every *.jsonl session under dir, newest first, each
// with a preview line so the picker can show something the user recognises.
// A missing directory is not an error — it just means there's nothing to
// resume yet.
func List(dir string) ([]Info, error) {
	return listDir(dir)
}

// ListArchived returns archived sessions (dir/archive) newest first, with the
// same Info shape so the sidebar can render a restorable "已归档" group.
func ListArchived(dir string) ([]Info, error) {
	return listDir(filepath.Join(dir, "archive"))
}

// listDir is the shared implementation behind List / ListArchived.
func listDir(dir string) ([]Info, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Info
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(dir, e.Name())
		preview, turns := previewSession(full)
		if turns == 0 {
			// Skip sessions that have never had user interaction — they are
			// empty conversations that should not appear in the history panel
			// or the resume picker.
			continue
		}
		out = append(out, Info{
			Path:    full,
			ModTime: info.ModTime(),
			Preview: preview,
			Turns:   turns,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModTime.After(out[j].ModTime)
	})
	return out, nil
}

// previewSession returns the first user message (truncated) and the number of
// user-role messages so the picker can show "5 turns · 'help me debug the…'".
// Errors are swallowed — a malformed file just shows up with an empty preview.
func previewSession(path string) (string, int) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	first := ""
	turns := 0
	for {
		var m provider.Message
		if err := dec.Decode(&m); err != nil {
			break // EOF or a malformed tail — return the preview gathered so far
		}
		if m.Role == provider.RoleUser {
			turns++
			if first == "" {
				s := strings.TrimSpace(m.Content)
				if r := []rune(s); len(r) > 80 {
					s = string(r[:77]) + "…"
				}
				first = s
			}
		}
	}
	return first, turns
}

// ─── V5.21: Session archive ──────────────────────────────────────────────

// Archive moves a session file (and its .meta sidecar) to an archive/
// subdirectory. Returns nil on success.
func Archive(sessionPath string) error {
	dir := filepath.Dir(sessionPath)
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("archive: mkdir: %w", err)
	}
	base := filepath.Base(sessionPath)
	dest := filepath.Join(archiveDir, base)
	if err := os.Rename(sessionPath, dest); err != nil {
		return fmt.Errorf("archive: rename: %w", err)
	}
	// best-effort: also move the .meta sidecar
	metaPath := sessionPath + ".meta"
	if _, err := os.Stat(metaPath); err == nil {
		_ = os.Rename(metaPath, filepath.Join(archiveDir, base+".meta"))
	}
	return nil
}

// Unarchive moves a session back from archive/ to its parent dir.
func Unarchive(archivePath string) error {
	parent := filepath.Dir(filepath.Dir(archivePath)) // up from archive/
	base := filepath.Base(archivePath)
	dest := filepath.Join(parent, base)
	if err := os.Rename(archivePath, dest); err != nil {
		return fmt.Errorf("unarchive: rename: %w", err)
	}
	metaPath := archivePath + ".meta"
	if _, err := os.Stat(metaPath); err == nil {
		_ = os.Rename(metaPath, filepath.Join(parent, base+".meta"))
	}
	return nil
}

// NewPath returns the path to use for a fresh session, namespaced by
// the model so the filename hints at what the conversation was with. dir is
// typically config.SessionDir().
func NewPath(dir, model string) string {
	safe := strings.NewReplacer("/", "-", "\\", "-").Replace(model)
	if safe == "" {
		safe = "session"
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%s.jsonl", time.Now().UTC().Format("20060102-150405.000000000"), safe))
}
