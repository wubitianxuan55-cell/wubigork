package app

// ── 原罪书级记忆管理面（升格刀4）──────────────────────────────────────
//
// 存储 = 引擎同一套 facts 面（memory.SQLiteStoreFor，project slug=书路径，
// space_id='play'——与办公 work 记忆硬隔离，AI 侧经 remember/memory_get 工具
// 读写同一存储）。本文件是**管理面**绑定（记忆卡的列表/新增/删除），语义：
//   - 读走 ListInSpace("play")（空间谓词，绝不回流办公 work facts）；
//   - 写显式 Space="play"；
//   - 删除 fail-closed：先 GetInSpace("play") 确认归属，跨空间名直接拒绝
//     （防卡片误删办公记忆）。

import (
	"fmt"
	"sort"
	"strings"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// sinMemoryStoreFor 书级记忆存储（SQLite 后端，slug=书路径）。
func (a *App) sinMemoryStoreFor(bookPath string) memory.Store {
	gdb := db.GetDatabase(gaeaConfig.MemoryUserDir())
	return memory.SQLiteStoreFor(gdb, gaeaConfig.MemoryUserDir(), bookPath)
}

// SinMemoryEntry 记忆卡条目视图。
type SinMemoryEntry struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Body        string `json:"body"`
	Pinned      bool   `json:"pinned"`
	// RFC3339（前端按时间排序展示）。
	UpdatedAt string `json:"updatedAt"`
}

// SinMemoryList 书级记忆清单（play 空间；query 非空时对 name/title/body 做
// 不区分大小写子串过滤）。引擎未绑书 = 空清单（卡片引导先选书）。
func (a *App) SinMemoryList(query string) []SinMemoryEntry {
	out := []SinMemoryEntry{}
	book, err := sinEngineBookRoot()
	if err != nil {
		return out
	}
	store := a.sinMemoryStoreFor(book)
	items := store.ListInSpace("play")
	q := strings.ToLower(strings.TrimSpace(query))
	for _, m := range items {
		if q != "" &&
			!strings.Contains(strings.ToLower(m.Name), q) &&
			!strings.Contains(strings.ToLower(m.Title), q) &&
			!strings.Contains(strings.ToLower(m.Body), q) {
			continue
		}
		out = append(out, SinMemoryEntry{
			Name:        m.Name,
			Title:       m.Title,
			Description: m.Description,
			Body:        m.Body,
			Pinned:      m.Pinned,
			UpdatedAt:   m.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

// sinMemoryNameNormalize 记忆键归一：空白折叠为 '-'，封顶 64 字符（rune）。
func sinMemoryNameNormalize(raw string) string {
	name := strings.Join(strings.Fields(strings.TrimSpace(raw)), "-")
	return truncateRunes(name, 64)
}

// SinMemorySave 新增/修订一条书级记忆（play 空间）。name 空 = 按 title 归一生成。
func (a *App) SinMemorySave(name string, title string, body string) (SinMemoryEntry, error) {
	book, err := sinEngineBookRoot()
	if err != nil {
		return SinMemoryEntry{}, err
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return SinMemoryEntry{}, fmt.Errorf("记忆标题不能为空")
	}
	if body == "" {
		return SinMemoryEntry{}, fmt.Errorf("记忆内容不能为空")
	}
	name = sinMemoryNameNormalize(name)
	if name == "" {
		name = sinMemoryNameNormalize(title)
	}
	if name == "" {
		return SinMemoryEntry{}, fmt.Errorf("记忆键归一后为空（title/正文需含可见字符）")
	}
	store := a.sinMemoryStoreFor(book)
	m := memory.Memory{
		Name:        name,
		Title:       title,
		Description: truncateRunes(strings.TrimSpace(sinMemoryFirstLine(body)), 120),
		Type:        memory.TypeProject,
		Kind:        memory.KindSemantic,
		Body:        body,
		Space:       "play",
	}
	if _, err := store.Save(m); err != nil {
		return SinMemoryEntry{}, fmt.Errorf("记忆保存失败: %w", err)
	}
	return SinMemoryEntry{Name: name, Title: title, Body: body, UpdatedAt: m.UpdatedAt.Format("2006-01-02 15:04")}, nil
}

// SinMemoryDelete 删除一条书级记忆（fail-closed：先确认 play 空间归属，
// 拒绝误删办公 work facts）。
func (a *App) SinMemoryDelete(name string) error {
	book, err := sinEngineBookRoot()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("记忆键为空")
	}
	store := a.sinMemoryStoreFor(book)
	if _, ok := store.GetInSpace(name, "play"); !ok {
		return fmt.Errorf("本书 play 空间不存在记忆 %q（跨空间删除已拒绝）", name)
	}
	return store.Delete(name)
}

// SinMemoryPinToggle 固化/取消固化（固化条免疫衰减、注入加权）。
func (a *App) SinMemoryPinToggle(name string, pinned bool) error {
	book, err := sinEngineBookRoot()
	if err != nil {
		return err
	}
	store := a.sinMemoryStoreFor(book)
	if _, ok := store.GetInSpace(strings.TrimSpace(name), "play"); !ok {
		return fmt.Errorf("本书 play 空间不存在记忆 %q", name)
	}
	if pinned {
		return store.Pin(strings.TrimSpace(name))
	}
	return store.Unpin(strings.TrimSpace(name))
}

// sinMemoryFirstLine 取文本第一行（Description 摘要用；与 app 包 firstLine 撞名避让）。
func sinMemoryFirstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
