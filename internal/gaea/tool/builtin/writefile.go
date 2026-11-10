package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"

	"github.com/gaea/gaea/internal/gaea/evidence"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

func init() { tool.RegisterBuiltin(writeFile{}) }

// writeFile writes a file. roots, when non-empty, confines the target to the
// workspace (see confine); the zero value registered at init is unconfined and
// is overridden per run by ConfineWriters. workDir, when non-empty, is the
// directory a relative path resolves against (see resolveIn).
type writeFile struct {
	roots   []string
	workDir string
}

func (writeFile) Name() string { return "write_file" }

func (writeFile) Description() string {
	return "Write content to a file at the given path (overwriting existing content). Creates parent directories as needed."
}

func (writeFile) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"},"content":{"type":"string","description":"Full content to write"},"expected_revision":{"type":"string","description":"Optional CAS guard: sha256 from read_file trailer, or \"absent\" to require the file does not exist (create-only). Mismatch = REVISION_CONFLICT."}},"required":["path","content"]}`)
}

func (writeFile) ReadOnly() bool { return false }

func (w writeFile) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	return []string{resolveIn(w.workDir, p.Path)}, nil
}

func (writeFile) CompactDescription() string     { return compactDesc["write_file"] }
func (writeFile) CompactSchema() json.RawMessage { return compactSchema["write_file"] }

func (w writeFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path             string `json:"path"`
		Content          string `json:"content"`
		ExpectedRevision string `json:"expected_revision,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	p.Path = resolveIn(w.workDir, p.Path)
	if err := confine(w.roots, p.Path); err != nil {
		return "", err
	}
	// D4 写锁（dsh-ops bind-before-publish 蒸馏）+A3 CAS：锁包「CAS→编码保真→写」
	// 整段；**半写债收口**——原 writeFileEncoded 是原地 os.WriteFile（失败可半写，
	// executeOne 旧注释记载），改走 atomicWriteEncoded（temp+rename 收尾，失败=
	// 目标未动）。父目录创建由 AtomicWrite 内建。
	var (
		before   string // v4.1 证据链：旧文件原文（写前捕获，仅旧文件存在时）
		baseline string // v4.1b：整文件基线快照
	)
	if err := WithFileLock(p.Path, func() error {
		bind, berr := bindFileLockTarget(p.Path)
		if berr != nil {
			return fmt.Errorf("stat %s: %w", p.Path, berr)
		}
		// A3 revision CAS（dsh-ops 蒸馏）：expected_revision 传入时前置校验
		// （""=零行为变化；"absent"=create-only；64hex=内容哈希比对）。
		if p.ExpectedRevision != "" {
			var cur []byte
			exists := false
			if raw, rerr := os.ReadFile(p.Path); rerr == nil {
				cur, exists = raw, true
			} else if !os.IsNotExist(rerr) {
				return fmt.Errorf("stat %s: %w", p.Path, rerr)
			}
			if err := checkExpectedRevision(p.Path, cur, exists, p.ExpectedRevision); err != nil {
				return err
			}
		}
		enc := fileenc.UTF8 // new files default to UTF-8；覆写旧文件时保真原编码
		if _, err := os.Stat(p.Path); err == nil {
			if oldContent, existingEnc, rerr := readFileEncoded(p.Path); rerr == nil {
				enc = existingEnc
				before = oldContent
				baseline = evidence.StageBaseline(ctx, p.Path, []byte(oldContent))
			}
		}
		if verr := bind.verify(p.Path); verr != nil {
			return verr
		}
		if werr := atomicWriteEncoded(p.Path, p.Content, enc); werr != nil {
			return fmt.Errorf("write %s: %w", p.Path, werr)
		}
		return nil
	}); err != nil {
		return "", err
	}
	// v4.1 证据链：整文件覆盖的 Before/After 原文摘要（Before 仅旧文件存在时）。
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "write_file",
		Target:        p.Path,
		BeforeSummary: before,
		AfterSummary:  p.Content,
		BaselinePath:  baseline,
	})
	return fmt.Sprintf("wrote %d bytes to %s", len(p.Content), p.Path), nil
}
