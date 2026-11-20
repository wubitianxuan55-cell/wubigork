// Package feedback 是 AI 反馈库（2026-11-20 用户立项：模型在使用过程中
// 遇到困难、发现不足后，可经 ai_feedback 工具主动提交反馈意见；记忆面板
// 反馈区收集展示，可勾选导出 md 用于改进软件）。
//
// 存储：<userDir>/ai-feedback.jsonl，JSONL 追加（append-only，单行一条）。
// 与 dream-pending 的「待确认队列」不同：反馈不是记忆候选，无需逐条确认，
// 直接入库；条目由用户在面板查看/删除/导出。读取时坏行跳过（单行损坏不
// 炸整库），删除走读-滤-原子重写。
package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// Item 一条 AI 反馈。
type Item struct {
	ID string `json:"id"`
	// Board 提交时的板块（chat | office | wenxue），模型自报——工具层拿
	// 不到可靠的板块标识（wenxue 引擎也标 work），缺省 unspecified。
	Board string `json:"board,omitempty"`
	// Category 问题分类：tool-friction（工具不好用）/ bug（行为不对）/
	// missing-capability（缺能力）/ workflow（工作流摩擦）/ other。
	Category string `json:"category,omitempty"`
	// Severity low | medium | high。
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	// CreatedAt RFC3339。
	CreatedAt string `json:"createdAt"`
}

// mu 串行化并发追加与删除重写（单进程内；文件级原子性由 O_APPEND 单行与
// AtomicWrite 保证）。
var mu sync.Mutex

// Path 返回反馈库文件路径。
func Path(userDir string) string {
	return filepath.Join(userDir, "ai-feedback.jsonl")
}

var idSeq struct {
	sync.Mutex
	n uint64
}

// NewID 生成条目 ID（fb<纳秒>-<序号>，进程内唯一）。
func NewID() string {
	idSeq.Lock()
	idSeq.n++
	n := idSeq.n
	idSeq.Unlock()
	return fmt.Sprintf("fb%d-%d", time.Now().UnixNano(), n)
}

// Append 追加一条反馈（自动补 ID/CreatedAt），返回入库条目。
// detail 上限 8KiB（防误把整段文档塞进来）；title 必填。
func Append(userDir string, it Item) (Item, error) {
	if userDir == "" {
		return Item{}, fmt.Errorf("feedback: userDir 为空")
	}
	if it.Title == "" {
		return Item{}, fmt.Errorf("feedback: title is required")
	}
	if len(it.Detail) > 8<<10 {
		return Item{}, fmt.Errorf("feedback: detail 超 8KiB 上限（%d bytes），请提炼后重提", len(it.Detail))
	}
	if it.ID == "" {
		it.ID = NewID()
	}
	if it.CreatedAt == "" {
		it.CreatedAt = time.Now().Format(time.RFC3339)
	}
	line, err := json.Marshal(it)
	if err != nil {
		return Item{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return Item{}, err
	}
	f, err := os.OpenFile(Path(userDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Item{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return Item{}, err
	}
	return it, nil
}

// List 读全量反馈（新→旧）。坏行跳过（json 解析失败/缺 ID 的行不炸整库）。
func List(userDir string) []Item {
	if userDir == "" {
		return nil
	}
	data, err := os.ReadFile(Path(userDir))
	if err != nil {
		return nil
	}
	var items []Item
	for _, line := range splitLines(data) {
		var it Item
		if json.Unmarshal(line, &it) != nil || it.ID == "" {
			continue
		}
		items = append(items, it)
	}
	// 新→旧（追加序为旧→新，倒排）
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items
}

// Delete 按 ID 删除一条（读-滤-原子重写）。不存在返回 false, nil。
func Delete(userDir, id string) (bool, error) {
	if userDir == "" || id == "" {
		return false, fmt.Errorf("feedback: userDir/id 为空")
	}
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(Path(userDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var kept [][]byte
	found := false
	for _, line := range splitLines(data) {
		var it Item
		if json.Unmarshal(line, &it) == nil && it.ID == id {
			found = true
			continue
		}
		kept = append(kept, line)
	}
	if !found {
		return false, nil
	}
	var out []byte
	for _, line := range kept {
		out = append(out, line...)
		out = append(out, '\n')
	}
	if err := fileutil.AtomicWrite(Path(userDir), out, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// splitLines 按 \n 切行并剔 \r（CRLF 宽容），空行丢弃。
func splitLines(data []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := data[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if len(line) > 0 {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		line := data[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			out = append(out, line)
		}
	}
	return out
}
