package context

import (
	"fmt"
	"sort"
	"sync"
)

// 上下文章节化（codex 蒸馏 T7，world_state/sections 同构，基于 T6 碎片契约）：
// 状态型注入按稳定 Kind 分章节，快照差分决定「本轮说什么」——
//   - 与上一轮完全一致 → 什么都不发（省 token 的唯一来源，这条写不出等于没差分）；
//   - 有旧内容、现在撤销 → 恰好一条移除通知（否则模型继续遵守已失效指引）；
//   - 历史查不到（Unknown）且现在非空 → 替换通知 + 内容（宁多一句通知，
//     不让模型以为现在有两条指引）。
//
// 三个坑（上游注释踩过，照抄规避）：三态 Previous 而非 *T 两态（「历史里没有」
// 与「历史里是空」必须分得开）；快照只放比较所需的可比较数据（字符串）；
// 通知文案固定（替换/移除各一句），不逐轮生成——否则 diff 永不收敛。

// PrevKind 是上一轮快照的三态：Absent=历史里没有该章节；Known=有快照；
// Unknown=历史里有旧片段但快照不可考（老会话迁移）。
type PrevKind int

const (
	PrevAbsent PrevKind = iota
	PrevKnown
	PrevUnknown
)

// Previous 是上一轮章节快照的三态视图。
type Previous struct {
	Kind  PrevKind
	Value string // 仅 PrevKnown 有意义
}

// sectionNoticeReplace / sectionNoticeRemove 是固定的通知文案（文案自写，
// 固定不逐轮生成——diff 收敛的前提）。
const (
	sectionNoticeReplace = "以下内容替换此前给出的同类指引。"
	sectionNoticeRemove  = "此前给出的同类指引已失效。"
)

// Section 是一个可差分的上下文章节。ID 必须稳定（跨轮、跨会话恢复）；
// Snapshot 只放比较所需数据；Body 为当前注入内容，空串=本章节撤销。
type Section interface {
	ID() string
	Snapshot() string
	Body() string
}

// sectionFragmentKind 是章节片段的 Kind 前缀（T6 稳定 Kind 契约）。
const sectionFragmentKindPrefix = "section."

// SectionFragment 把章节内容包成 T6 碎片（user 角色 + ProvHarness——role 与
// 归属是两件事，授权判定走 Prov 不走 role）。
func SectionFragment(id, body string) Fragment {
	start, end := TypeMarkers(sectionFragmentKindPrefix + id)
	return Fragment{
		Role:  "user",
		Kind:  sectionFragmentKindPrefix + id,
		Start: start, End: end,
		Body: body,
		Prov: ProvHarness,
	}
}

// RenderSectionDiff 是章节差分核（纯函数，三情形）：
// 返回（新快照, 本轮应注入的片段）。prev 三态语义见上。
func RenderSectionDiff(id string, prev Previous, cur Section) (string, []Fragment) {
	snap := cur.Snapshot()
	body := cur.Body()
	// ① 相同就不发。
	if prev.Kind == PrevKnown && prev.Value == snap {
		return snap, nil
	}
	// 上一轮是否可能有旧内容：Known 看旧快照是否非空；Unknown 当有（保守）。
	hadBefore := false
	switch prev.Kind {
	case PrevKnown:
		hadBefore = prev.Value != ""
	case PrevUnknown:
		hadBefore = true
	}
	switch {
	case body == "" && !hadBefore:
		// 之前没有、现在也没有：只更新快照，不发言。
		return snap, nil
	case body == "":
		// 之前有、现在撤销：恰好一条移除通知。
		return snap, []Fragment{SectionFragment(id, sectionNoticeRemove)}
	case hadBefore:
		// 之前有（或不可考）、现在有新内容：替换通知 + 内容。
		return snap, []Fragment{
			SectionFragment(id, sectionNoticeReplace+"\n\n"+body),
		}
	default:
		// 首次出现：直出内容。
		return snap, []Fragment{SectionFragment(id, body)}
	}
}

// SectionStore 是会话内的章节快照账本（内存态，随会话生命周期）。
// 快照只存比较所需字符串；unknowns 记「历史有旧片段但快照不可考」的章节
// （老会话迁移：下次 diff 走 Unknown=当有 → 替换通知，不会叠加）。
type SectionStore struct {
	mu       sync.RWMutex
	snaps    map[string]string
	unknowns map[string]bool
}

// NewSectionStore 构造空账本。
func NewSectionStore() *SectionStore {
	return &SectionStore{snaps: map[string]string{}, unknowns: map[string]bool{}}
}

// Reset 清空账本。新会话必须从空账本重新开始，否则上一会话的同值快照会让
// 新 transcript 首轮误判「内容未变化」而漏注入。
func (s *SectionStore) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snaps = map[string]string{}
	s.unknowns = map[string]bool{}
	s.mu.Unlock()
}

// ResetUnknown 把已有章节标成「历史有旧内容、快照不可考」，用于恢复会话：
// transcript 已带上一轮注入，但新进程里的内存账本没有可比较快照。
func (s *SectionStore) ResetUnknown(ids ...string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.snaps = map[string]string{}
	s.unknowns = make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != "" {
			s.unknowns[id] = true
		}
	}
	s.mu.Unlock()
}

// MarkUnknown 标记某章节「历史有旧片段但快照不可考」（老会话迁移入口）。
func (s *SectionStore) MarkUnknown(id string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.unknowns == nil {
		s.unknowns = map[string]bool{}
	}
	s.unknowns[id] = true
	s.mu.Unlock()
}

// PreviousOf 取章节的上一轮三态视图。
func (s *SectionStore) PreviousOf(id string) Previous {
	if s == nil {
		return Previous{Kind: PrevAbsent}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.previousOfLocked(id)
}

func (s *SectionStore) previousOfLocked(id string) Previous {
	if s.unknowns[id] {
		return Previous{Kind: PrevUnknown}
	}
	v, ok := s.snaps[id]
	if !ok {
		return Previous{Kind: PrevAbsent}
	}
	return Previous{Kind: PrevKnown, Value: v}
}

// RenderAll 对一组章节做差分并更新账本：返回本轮应注入的全部片段，
// 按章节 ID 字典序稳定排列（同输入同输出，片段顺序可断言）。
func (s *SectionStore) RenderAll(sections []Section) []Fragment {
	if s == nil {
		return nil
	}
	sorted := make([]Section, len(sections))
	copy(sorted, sections)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID() < sorted[j].ID() })
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snaps == nil {
		s.snaps = map[string]string{}
	}
	if s.unknowns == nil {
		s.unknowns = map[string]bool{}
	}
	var out []Fragment
	for _, sec := range sorted {
		next, frags := RenderSectionDiff(sec.ID(), s.previousOfLocked(sec.ID()), sec)
		s.snaps[sec.ID()] = next
		delete(s.unknowns, sec.ID()) // 已考得快照，Unknown 态结束
		out = append(out, frags...)
	}
	return out
}

// SnapshotOf 返回章节当前快照（观测/视图用，不改状态）。
func (s *SectionStore) SnapshotOf(id string) (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.snaps[id]
	return v, ok
}

// String 是调试视图（章节 ID 序）。
func (s *SectionStore) String() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.snaps))
	for id := range s.snaps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := ""
	for _, id := range ids {
		out += fmt.Sprintf("%s=%q\n", id, s.snaps[id])
	}
	return out
}
