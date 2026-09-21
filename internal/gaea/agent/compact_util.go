package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// mechanicalFoldDigest produces a fallback message when summarization failed.
func mechanicalFoldDigest(n int, archive string) string {
	where := "."
	if archive != "" {
		where = " (archived to " + archive + ")."
	}
	return fmt.Sprintf("%d earlier message(s) were folded here to free context, but the automatic summary was unavailable%s Ask the user if you need details from before this point.", n, where)
}

// ─── Token estimation ───

func estimateTextTokens(s string) int {
	if s == "" {
		return 0
	}
	bytes := len(s)
	runes := utf8.RuneCountInString(s)
	// ASCII / European text: ~4 bytes per token
	// CJK / multi-byte text: ~2 characters per token (BPE tokenizer)
	byBytes := (bytes + 3) / 4
	byRunes := (runes + 1) / 2
	if byRunes > byBytes {
		return byRunes
	}
	return byBytes
}

func foldEconomics(region []provider.Message) bool {
	const minFoldTokens = 400
	return estimateMessagesTokens(region) >= minFoldTokens
}

func estimateMessagesTokens(msgs []provider.Message) int {
	total := 0
	for _, m := range msgs {
		total += 4
		total += estimateTextTokens(m.Content)
		total += estimateTextTokens(m.ReasoningContent)
		total += estimateTextTokens(m.Name)
		total += estimateTextTokens(m.ToolCallID)
		for _, tc := range m.ToolCalls {
			total += 8
			total += estimateTextTokens(tc.ID)
			total += estimateTextTokens(tc.Name)
			total += estimateTextTokens(tc.Arguments)
		}
	}
	return total
}

// ─── Transcript rendering ───
// （缓存对齐摘要改造后 transcript 平铺已弃用：摘要请求直接逐字重放被折叠
// 区间的原消息，见 compact.go summarize——平铺文本既破坏 provider 缓存
// 对齐，又丢掉对话内结构。archiveMessages 仍是剪枝/压缩的落盘底座。）

func archiveMessages(dir string, msgs []provider.Message) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, time.Now().Format("20060102-150405.000")+".jsonl")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, m := range msgs {
		if err := enc.Encode(m); err != nil {
			return "", err
		}
	}
	return path, nil
}

// readProgressFile reads the agent todo persistence file (.gaea/todos.md,
// v4.27.4 改名) from the project root (found by walking up from cwd), falling
// back to the legacy .gaea/progress.md name for existing workspaces. Returns
// "" if neither exists or can't be read.
func readProgressFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, name := range []string{"todos.md", "progress.md"} {
			if data, err := os.ReadFile(filepath.Join(dir, ".gaea", name)); err == nil {
				return string(data)
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
