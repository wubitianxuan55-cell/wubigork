package app

// 自动做梦「待确认建议队列」（v4.377 建议制口径：未经用户确认不写记忆）。
//
// suggest 模式下，每轮提炼出的 facts/notes 不再直写长期记忆与项目文档，
// 而是进入本队列（<userDir>/dream-pending.json，FIFO 上限 30 条），由
// 记忆面板「建议」标签页展示；用户逐条「接受」才入库（source=explicit
// 落审计日志）、「忽略」即丢弃。队列持久化（重启不丢）、损坏时整体作废
// 重建（建议本就是可再生候选，宁丢不阻）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// dreamPendingMax 是待确认队列上限（FIFO：超限丢最旧）。
const dreamPendingMax = 30

// dreamPendingMu 保护队列文件读写（单进程内串行化；文件本身原子写）。
var dreamPendingMu sync.Mutex

// dreamPendingItem 是待确认队列的一条建议：fact 型（Kind="fact"）来自
// dream facts，note 型（Kind="note"）来自 dream notes。
type dreamPendingItem struct {
	ID   string `json:"id"`
	Kind string `json:"kind"` // "fact" | "note"
	// fact 字段（note 型为空）
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	MemoryKind  string `json:"memoryKind,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body,omitempty"`
	// note 字段（fact 型为空）
	NoteScope string `json:"noteScope,omitempty"`
	Note      string `json:"note,omitempty"`
	// Space 是触发会话的空间（建议只在同空间面板展示/接受）。
	Space string `json:"space,omitempty"`
	// CreatedAt 是入队时间（RFC3339）。
	CreatedAt string `json:"createdAt"`
}

// dreamPendingPath 返回待确认队列文件路径（与 dream-audit.jsonl 同目录）。
func dreamPendingPath(userDir string) string {
	return filepath.Join(userDir, "dream-pending.json")
}

// dreamPendingRead 读取队列（文件不存在/损坏返回空队列——损坏整体作废）。
func dreamPendingRead(userDir string) []dreamPendingItem {
	if userDir == "" {
		return nil
	}
	data, err := os.ReadFile(dreamPendingPath(userDir))
	if err != nil {
		return nil
	}
	var items []dreamPendingItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	return items
}

// dreamPendingWrite 原子覆写队列文件。
func dreamPendingWrite(userDir string, items []dreamPendingItem) error {
	if userDir == "" {
		return fmt.Errorf("dream pending: userDir 为空")
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(items, "", " ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(dreamPendingPath(userDir), b, 0o644)
}

// dreamPendingAppend 把建议追加进队列（FIFO 截断到上限），返回追加条数。
func dreamPendingAppend(userDir string, add []dreamPendingItem) (int, error) {
	n := 0
	for _, it := range add {
		if strings.TrimSpace(it.ID) == "" {
			continue
		}
		if it.Kind != "note" && strings.TrimSpace(it.Name) == "" {
			continue
		}
		if it.Kind == "note" && strings.TrimSpace(it.Note) == "" {
			continue
		}
		n++
	}
	if n == 0 || userDir == "" {
		return 0, nil
	}
	dreamPendingMu.Lock()
	defer dreamPendingMu.Unlock()
	items := dreamPendingRead(userDir)
	items = append(items, add...)
	if over := len(items) - dreamPendingMax; over > 0 {
		items = items[over:]
	}
	if err := dreamPendingWrite(userDir, items); err != nil {
		return 0, err
	}
	return n, nil
}

// dreamPendingList 返回当前空间可见的待确认建议（space 过滤：建议只在
// 触发会话的同空间面板展示——work 面板不见 play 建议，反之亦然；空
// space（mode=off 平铺）不过滤，与 dream 写侧 Normalize 兜底一致）。
func dreamPendingList(userDir, space string) []dreamPendingItem {
	sp := spaces.Normalize(space)
	var out []dreamPendingItem
	for _, it := range dreamPendingRead(userDir) {
		if sp != "" && spaces.Normalize(it.Space) != sp {
			continue
		}
		out = append(out, it)
	}
	return out
}

// dreamPendingTake 按 ID 取出并移除一条建议。不存在返回 ok=false、err=nil；
// 存在但出队失败（文件写坏）返回 err——调用方不得静默落入其他写入路径
// （会把用户尚未确认的建议写进记忆且队列残留）。
func dreamPendingTake(userDir, id string) (dreamPendingItem, bool, error) {
	dreamPendingMu.Lock()
	defer dreamPendingMu.Unlock()
	items := dreamPendingRead(userDir)
	for i, it := range items {
		if it.ID == id {
			rest := append(items[:i:i], items[i+1:]...)
			if err := dreamPendingWrite(userDir, rest); err != nil {
				return dreamPendingItem{}, false, err
			}
			return it, true, nil
		}
	}
	return dreamPendingItem{}, false, nil
}

// newDreamPendingID 生成建议 ID（时间戳纳秒 + 计数器，进程内唯一）。
var dreamPendingSeq struct {
	sync.Mutex
	n uint64
}

func newDreamPendingID() string {
	dreamPendingSeq.Lock()
	dreamPendingSeq.n++
	n := dreamPendingSeq.n
	dreamPendingSeq.Unlock()
	return fmt.Sprintf("d%d-%d", time.Now().UnixNano(), n)
}
