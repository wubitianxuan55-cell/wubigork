// Package office — audit_log.go
package office

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AuditEntry struct {
	TS      string `json:"ts"`
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	PathTo  string `json:"path_to,omitempty"`
	Target  string `json:"target,omitempty"`
	URL     string `json:"url,omitempty"`
	Result  string `json:"result"`
	Summary string `json:"summary"`
}

func auditLogPath(dataRoot string) string { return filepath.Join(dataRoot, "office", "audit.jsonl") }

func AppendAudit(dataRoot string, entry AuditEntry) error {
	path := auditLogPath(dataRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return err }
	b, err := json.Marshal(entry)
	if err != nil { return err }
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil { return err }
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func ReadAuditSince(dataRoot, sinceISO string) ([]AuditEntry, error) {
	path := auditLogPath(dataRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) { return nil, nil }
		return nil, err
	}
	since, err := time.Parse(time.RFC3339, sinceISO)
	if err != nil { return nil, nil }
	var entries []AuditEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil { continue }
		ts, err := time.Parse(time.RFC3339, entry.TS)
		if err != nil { continue }
		if !ts.Before(since) { entries = append(entries, entry) }
	}
	return entries, nil
}
