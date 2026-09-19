package app

// ── 原罪工具：故事便签 / 故事大纲（sin_notes / sin_outline）──
//
// 落点：<用户配置目录>/gaea/sin/notes/<故事 id>.json —— 原罪自有数据面
// （硬隔离口径见 sin_store.go）：不写办公工作区、不进办公记忆与知识库。
// 一份文件装两样东西：便签（设定集，逐条增删改）与大纲（整体替换）。
//
// 容错纪律与 cast.json 同源：文件缺失/损坏只当空文档（辅助数据不阻断故事
// 创作）；写入原子（临时文件 + rename）；进程内单写者由 sinNotesMu 串行化。
//
// 路径安全：故事 id 只允许 [A-Za-z0-9_-] 作纯文件名——落盘路径不接受任何
// 路径分隔与上跳（fail-closed，不靠调用方自觉）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

const (
	// sinNotesVersion 便签文件版本（结构变更时按版本迁移）。
	sinNotesVersion = 1
	// sinNotesMaxItems 单个故事便签条数上限（防设定集无节制膨胀）。
	sinNotesMaxItems = 200
	// sinNoteMaxRunes 单条便签长度上限（rune）。
	sinNoteMaxRunes = 2000
	// sinOutlineMaxRunes 大纲长度上限（rune）。
	sinOutlineMaxRunes = 4000
)

var sinNotesMu sync.Mutex

func init() {
	registerSinTool(sinToolNotes, func(c sinToolContext) sinTool {
		return sinNotesTool{topicID: c.topicID}
	})
	registerSinTool(sinToolOutline, func(c sinToolContext) sinTool {
		return sinOutlineTool{topicID: c.topicID}
	})
}

// sinNotesDir 故事便签目录。
func sinNotesDir() string {
	return filepath.Join(sinRoot(), "notes")
}

// sinNotesPath 便签文件路径（故事 id 非法 → 错误，绝不拼路径）。
func sinNotesPath(topicID string) (string, error) {
	id := strings.TrimSpace(topicID)
	if !sinNotesIDOK(id) {
		return "", fmt.Errorf("故事 ID 非法（不能作为文件名）: %q", topicID)
	}
	return filepath.Join(sinNotesDir(), id+".json"), nil
}

// sinNotesIDOK 故事 id 是否可安全用作文件名：字母/数字/下划线/短横线。
// 真实 id 形如 sin_1726100000000_1，全部落在字符集内。
func sinNotesIDOK(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// sinNotesDoc 便签文件结构：便签按写入顺序排列，大纲整体一份。
type sinNotesDoc struct {
	Version int      `json:"version"`
	Notes   []string `json:"notes"`
	Outline string   `json:"outline"`
}

// loadSinNotes 读取便签文档（不存在/损坏 → 空文档；损坏时告警不阻断）。
func loadSinNotes(path string) sinNotesDoc {
	doc := sinNotesDoc{Version: sinNotesVersion, Notes: []string{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		return doc
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		slog.Warn("原罪便签文件损坏，按空文档继续", "path", path, "error", err)
		return sinNotesDoc{Version: sinNotesVersion, Notes: []string{}}
	}
	if doc.Notes == nil {
		doc.Notes = []string{}
	}
	doc.Version = sinNotesVersion
	return doc
}

// sinDraftForPrompt 流式回合用的底稿快照（fail-open：id 非法/读失败 = 空文档，
// 底稿是辅助数据，不阻断故事创作）。锁内读取——同一时刻模型工具可能正在写。
func (a *App) sinDraftForPrompt(topicID string) sinNotesDoc {
	path, err := sinNotesPath(topicID)
	if err != nil {
		return sinNotesDoc{Version: sinNotesVersion, Notes: []string{}}
	}
	sinNotesMu.Lock()
	defer sinNotesMu.Unlock()
	return loadSinNotes(path)
}

// saveSinNotes 原子写回便签文档（临时文件 + rename；失败向上抛）。
func saveSinNotes(path string, doc sinNotesDoc) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建原罪便签目录失败: %w", err)
	}
	doc.Version = sinNotesVersion
	if doc.Notes == nil {
		doc.Notes = []string{}
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("写入故事便签失败: %w", err)
	}
	if err := fileutil.RenameWithRetry(tmp, path); err != nil {
		return fmt.Errorf("保存故事便签失败: %w", err)
	}
	return nil
}

// ── sin_notes ─────────────────────────────────────────────────

// sinNotesTool 故事便签：list / read / write / set / delete。
// （工具名常量是 sinToolNotes；类型名单独取，避免与常量重名。）
type sinNotesTool struct{ topicID string }

func (sinNotesTool) Name() string { return sinToolNotes }

func (sinNotesTool) Description() string {
	return "故事便签（设定集）：把用户定下、后面必须保持一致的东西记下来——人名与称呼、外貌与关系、时间线、" +
		"伏笔与已用过的桥段。非空时便签全文已随每轮前情附在上下文里，通常无需 read 即可核对，" +
		"只有要确认某条全文时才 read；用户新定一条设定就用 write 追加。" +
		"便签是你的工作底稿，不是正文，不要把它写进故事。"
}

func (sinNotesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","enum":["list","read","write","set","delete"],
    "description":"list=列出全部便签（带序号）；read=读全部（带 index 时读某一条）；write=追加一条（需 content）；set=替换某一条（需 index 与 content）；delete=删除某一条（需 index）"},
  "content":{"type":"string","description":"write/set 的便签正文"},
  "index":{"type":"integer","minimum":0,"description":"便签序号，0 起（list 结果里给出）"}
},
"required":["action"]}`)
}

// ReadOnly=false：本工具含写入动作（前端徽标与提示词据此如实标注）。
func (sinNotesTool) ReadOnly() bool { return false }

func (t sinNotesTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action  string `json:"action"`
		Content string `json:"content"`
		Index   *int   `json:"index"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
	}
	action := strings.ToLower(strings.TrimSpace(p.Action))
	path, err := sinNotesPath(t.topicID)
	if err != nil {
		return "", err
	}

	sinNotesMu.Lock()
	defer sinNotesMu.Unlock()
	doc := loadSinNotes(path)

	switch action {
	case "list":
		return sinNotesListText(doc), nil

	case "read":
		if p.Index != nil {
			i, err := sinNotesIndex(doc, *p.Index)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("便签 #%d：\n%s", i, doc.Notes[i]), nil
		}
		if len(doc.Notes) == 0 {
			return "还没有便签。", nil
		}
		var b strings.Builder
		for i, n := range doc.Notes {
			fmt.Fprintf(&b, "#%d %s\n", i, n)
		}
		return strings.TrimRight(b.String(), "\n"), nil

	case "write":
		content := strings.TrimSpace(p.Content)
		if content == "" {
			return "", fmt.Errorf("write 需要 content（便签正文不能为空）")
		}
		if len(doc.Notes) >= sinNotesMaxItems {
			return "", fmt.Errorf("便签已达上限 %d 条：先合并，或 delete 掉不再需要的条目", sinNotesMaxItems)
		}
		doc.Notes = append(doc.Notes, truncateRunes(content, sinNoteMaxRunes))
		if err := saveSinNotes(path, doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("已记录便签 #%d（共 %d 条）。", len(doc.Notes)-1, len(doc.Notes)), nil

	case "set":
		if p.Index == nil {
			return "", fmt.Errorf("set 需要 index（便签序号，0 起）")
		}
		content := strings.TrimSpace(p.Content)
		if content == "" {
			return "", fmt.Errorf("set 需要 content（要删除请用 delete）")
		}
		i, err := sinNotesIndex(doc, *p.Index)
		if err != nil {
			return "", err
		}
		doc.Notes[i] = truncateRunes(content, sinNoteMaxRunes)
		if err := saveSinNotes(path, doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("已更新便签 #%d。", i), nil

	case "delete":
		if p.Index == nil {
			return "", fmt.Errorf("delete 需要 index（便签序号，0 起）")
		}
		i, err := sinNotesIndex(doc, *p.Index)
		if err != nil {
			return "", err
		}
		doc.Notes = append(doc.Notes[:i:i], doc.Notes[i+1:]...)
		if err := saveSinNotes(path, doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("已删除便签 #%d（剩 %d 条）。", i, len(doc.Notes)), nil

	case "":
		return "", fmt.Errorf("缺少 action（list/read/write/set/delete）")
	default:
		return "", fmt.Errorf("未知 action：%s（可用：list/read/write/set/delete）", p.Action)
	}
}

// sinNotesListText 便签目录（每条截断到 120 rune，长设定用 read 看全文）。
func sinNotesListText(doc sinNotesDoc) string {
	if len(doc.Notes) == 0 {
		return "还没有便签。"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "故事便签（%d 条）：\n", len(doc.Notes))
	for i, n := range doc.Notes {
		fmt.Fprintf(&b, "#%d %s\n", i, truncateRunes(strings.TrimSpace(n), 120))
	}
	return strings.TrimRight(b.String(), "\n")
}

// sinNotesIndex 校验便签序号并给人话错误（越界时如实回列有效范围）。
func sinNotesIndex(doc sinNotesDoc, idx int) (int, error) {
	if idx < 0 || idx >= len(doc.Notes) {
		if len(doc.Notes) == 0 {
			return 0, fmt.Errorf("便签序号 %d 越界：当前没有便签", idx)
		}
		return 0, fmt.Errorf("便签序号 %d 越界：当前有效范围 0~%d", idx, len(doc.Notes)-1)
	}
	return idx, nil
}

// ── sin_outline ───────────────────────────────────────────────

// sinOutlineTool 故事大纲：read / write（整体替换）。
type sinOutlineTool struct{ topicID string }

func (sinOutlineTool) Name() string { return sinToolOutline }

func (sinOutlineTool) Description() string {
	return "故事大纲：分章推进顺序、时间线、伏笔的埋与收。大纲非空时已随每轮前情附在上下文里，" +
		"「按大纲写」「别跑偏」直接按它执行；用户定下整体走向、或故事推进到一个阶段结束时，" +
		"把更新后的大纲整体 write 回去（write 会替换旧大纲）。大纲是工作底稿，不是正文。"
}

func (sinOutlineTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","enum":["read","write"],
    "description":"read=读取当前大纲；write=整体替换大纲（content 传空串=清空）"},
  "content":{"type":"string","description":"write 的大纲正文（章节顺序/时间线/伏笔）"}
},
"required":["action"]}`)
}

// ReadOnly=false：write 会改写大纲（前端徽标据此如实标注）。
func (sinOutlineTool) ReadOnly() bool { return false }

func (t sinOutlineTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action  string `json:"action"`
		Content string `json:"content"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
	}
	action := strings.ToLower(strings.TrimSpace(p.Action))
	path, err := sinNotesPath(t.topicID)
	if err != nil {
		return "", err
	}

	sinNotesMu.Lock()
	defer sinNotesMu.Unlock()
	doc := loadSinNotes(path)

	switch action {
	case "read":
		outline := strings.TrimSpace(doc.Outline)
		if outline == "" {
			return "还没有大纲。", nil
		}
		return "故事大纲：\n" + outline, nil

	case "write":
		doc.Outline = truncateRunes(strings.TrimSpace(p.Content), sinOutlineMaxRunes)
		if err := saveSinNotes(path, doc); err != nil {
			return "", err
		}
		if doc.Outline == "" {
			return "已清空故事大纲。", nil
		}
		return fmt.Sprintf("已更新故事大纲（%d 字）。", len([]rune(doc.Outline))), nil

	case "":
		return "", fmt.Errorf("缺少 action（read/write）")
	default:
		return "", fmt.Errorf("未知 action：%s（可用：read/write）", p.Action)
	}
}
