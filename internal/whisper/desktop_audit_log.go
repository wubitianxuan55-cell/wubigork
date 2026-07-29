// Package whisper — desktop_audit_log.go
// 100% 对齐 ackem desktop-agent/auditLog.ts
// JSONL 审计日志：记录所有 use_computer 操作
package whisper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DesktopAgentAuditEntry 审计日志条目
type DesktopAgentAuditEntry struct {
	TS      string `json:"ts"`
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	PathTo  string `json:"path_to,omitempty"`
	Target  string `json:"target,omitempty"`
	URL     string `json:"url,omitempty"`
	Result  string `json:"result"` // allowed/denied/blocked/error/timeout
	Summary string `json:"summary"`
}

// auditLogPath 审计日志路径
func auditLogPath(dataRoot string) string {
	return filepath.Join(dataRoot, "desktop-agent", "audit.jsonl")
}

// AppendDesktopAgentAudit 追加一条审计记录
func AppendDesktopAgentAudit(dataRoot string, entry DesktopAgentAuditEntry) error {
	path := auditLogPath(dataRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(b, '\n'))
	return err
}

// ReadAuditEntriesSince 读取某时间点之后的审计条目
func ReadAuditEntriesSince(dataRoot, sinceISO string) ([]DesktopAgentAuditEntry, error) {
	path := auditLogPath(dataRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	since, err := time.Parse(time.RFC3339, sinceISO)
	if err != nil {
		return nil, nil
	}

	var entries []DesktopAgentAuditEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry DesktopAgentAuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		ts, err := time.Parse(time.RFC3339, entry.TS)
		if err != nil {
			continue
		}
		if !ts.Before(since) {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}
