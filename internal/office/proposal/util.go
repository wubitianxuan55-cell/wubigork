// Package proposal — 工具函数
package proposal

import (
	"os"
	"path/filepath"
)

func filepathInDir(dir, filename string) string {
	return filepath.Join(dir, filename)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func extractJSON(s string) string {
	s = trimSpace(s)
	if hasPrefix(s, "```json") {
		s = trimPrefix(s, "```json")
		if idx := lastIndex(s, "```"); idx >= 0 { s = s[:idx] }
	} else if hasPrefix(s, "```") {
		s = trimPrefix(s, "```")
		if idx := lastIndex(s, "```"); idx >= 0 { s = s[:idx] }
	}
	return trimSpace(s)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen { return s }
	return string(runes[:maxLen]) + "…"
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t') { s = s[1:] }
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == '\t') { s = s[:len(s)-1] }
	return s
}
func hasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }
func trimPrefix(s, prefix string) string { if hasPrefix(s, prefix) { return s[len(prefix):] }; return s }
func lastIndex(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- { if s[i:i+len(sub)] == sub { return i } }
	return -1
}
