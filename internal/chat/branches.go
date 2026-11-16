package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// branches.go —— 消息分支版本树（A 批，chat-demo branches.ts 语义对齐）。
// 设计文档 docs/gaea-chat-branch-tree-design-2026-11.md：
//   - 分支字段进 extra.bt（零 schema 迁移）：p=父消息 id（根为 0），c=选中子消息
//     id（选择指针挂父上，DeepSeek 同位语义——切版本不丢各分支自身选择）；
//   - 未规范化话题（任一消息缺 bt）可见路径=seq 序，旧行为零；
//   - 首次分支操作触发整话题规范化（按 seq 线性回填 bt）；
//   - 可见路径用「叶子回溯」替代 DeepSeek 的根前向走：分页窗口下只需已加载
//     集合即可走链，父缺失（在更早分页）时自然截断，loadOlder 追加后延长。

// BT 消息分支树字段（extra.bt）。
type BT struct {
	P int64 `json:"p"` // 父消息 id；根为 0
	C int64 `json:"c"` // 选中子消息 id；0=无子/按 seq 最大子兜底
}

// ParseBT 从 extra JSON 取 bt；缺失或解析失败返回 ok=false（=未规范化语义）。
func ParseBT(extra string) (BT, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(extra), &m); err != nil {
		return BT{}, false
	}
	raw, ok := m["bt"]
	if !ok {
		return BT{}, false
	}
	var bt BT
	if err := json.Unmarshal(raw, &bt); err != nil {
		return BT{}, false
	}
	return bt, true
}

// mergeExtraBT 把 bt 写进 extra JSON，保留其余键（search_summary/answered_by/
// reasoning 等）；extra 为空时从零构造。
func MergeExtraBT(extra string, bt BT) (string, error) {
	m := map[string]json.RawMessage{}
	if extra != "" {
		if err := json.Unmarshal([]byte(extra), &m); err != nil {
			return "", fmt.Errorf("extra 非合法 JSON: %w", err)
		}
	}
	b, err := json.Marshal(bt)
	if err != nil {
		return "", err
	}
	m["bt"] = b
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// BTNeeded 是否需要规范化：任一消息缺 bt。
func BTNeeded(msgs []Message) bool {
	for i := range msgs {
		if _, ok := ParseBT(msgs[i].Extra); !ok {
			return true
		}
	}
	return false
}

// LinearBTs 按 seq 序线性回填 bt 表：父=前一条，子 `c` 指向后一条，末条 c=0。
func LinearBTs(msgs []Message) map[int64]BT {
	out := make(map[int64]BT, len(msgs))
	for i := range msgs {
		bt := BT{P: 0, C: 0}
		if i > 0 {
			bt.P = msgs[i-1].ID
		}
		if i+1 < len(msgs) {
			bt.C = msgs[i+1].ID
		}
		out[msgs[i].ID] = bt
	}
	return out
}

// VisiblePath 可见路径：叶子回溯。
// 线性话题（任一消息缺 bt）按 seq 直通。树形话题：候选叶子 = 无子（bt.c==0）
// 的消息，按 seq 新→旧取第一个「回溯链全过选中校验」的——即链上每个节点的
// 父 bt.c 都指向它（父在窗口外时按可见截断，分页追加后自然延长链）。
func VisiblePath(msgs []Message) []Message {
	if BTNeeded(msgs) {
		return msgs
	}
	byID := make(map[int64]Message, len(msgs))
	for i := range msgs {
		byID[msgs[i].ID] = msgs[i]
	}
	// chainVisible 从 cur 沿 bt.p 回溯，校验每步父的选中指针；父缺失（更早
	// 分页）按可见处理（窗口截断）。
	chainVisible := func(cur int64) bool {
		for cur != 0 {
			m, ok := byID[cur]
			if !ok {
				return true
			}
			bt, ok := ParseBT(m.Extra)
			if !ok {
				return true
			}
			if bt.P == 0 {
				return true
			}
			parent, ok := byID[bt.P]
			if !ok {
				return true
			}
			pbt, ok := ParseBT(parent.Extra)
			if !ok {
				return true
			}
			if pbt.C != cur {
				return false
			}
			cur = bt.P
		}
		return true
	}
	leafID := int64(0)
	for i := len(msgs) - 1; i >= 0; i-- {
		bt, ok := ParseBT(msgs[i].Extra)
		if !ok || bt.C != 0 {
			continue
		}
		if chainVisible(msgs[i].ID) {
			leafID = msgs[i].ID
			break
		}
	}
	if leafID == 0 {
		return nil
	}
	path := make([]Message, 0, len(msgs))
	cur := leafID
	for cur != 0 {
		m, ok := byID[cur]
		if !ok {
			break // 父在窗口外（更早分页），链截断
		}
		path = append(path, m)
		bt, ok := ParseBT(m.Extra)
		if !ok {
			break
		}
		cur = bt.P
	}
	// 反转（旧→新）
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// UpdateMessageBT merge 写回单消息的 bt（保留 extra 其余键）。
func (s *Store) UpdateMessageBT(id int64, bt BT) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	var extra string
	if err := s.db.QueryRow("SELECT extra FROM chat_messages WHERE id = ?", id).Scan(&extra); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("消息不存在: %d", id)
		}
		return err
	}
	merged, err := MergeExtraBT(extra, bt)
	if err != nil {
		return err
	}
	res, err := s.db.Exec("UPDATE chat_messages SET extra = ? WHERE id = ?", merged, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("消息不存在: %d", id)
	}
	return nil
}

// NormalizeTopicBranches 首次分支操作前的整话题规范化：按 seq 线性回填 bt，
// 逐条 merge 写回（保留 extra 既有键）。已规范化（无缺失）时空操作。
func (s *Store) NormalizeTopicBranches(topicID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	msgs, err := s.ListMessages(topicID)
	if err != nil {
		return err
	}
	if !BTNeeded(msgs) {
		return nil
	}
	bts := LinearBTs(msgs)
	for i := range msgs {
		if _, ok := ParseBT(msgs[i].Extra); ok {
			continue
		}
		if err := s.UpdateMessageBT(msgs[i].ID, bts[msgs[i].ID]); err != nil {
			return err
		}
	}
	return nil
}

// SetSelectedChild 改父消息的选中子指针（bt.c）。校验三件事防越枝改写：
// parent、child 都存在且同话题；child.bt.p 确实指向 parent。
func (s *Store) SetSelectedChild(parentID, childID int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	parent, err := s.GetMessage(parentID)
	if err != nil {
		return err
	}
	child, err := s.GetMessage(childID)
	if err != nil {
		return err
	}
	if parent.TopicID != child.TopicID {
		return fmt.Errorf("父子消息不同话题: %d/%d", parentID, childID)
	}
	cbt, ok := ParseBT(child.Extra)
	if !ok || cbt.P != parentID {
		return fmt.Errorf("子消息 %d 的父不是 %d", childID, parentID)
	}
	return s.UpdateMessageBT(parentID, BT{P: func() int64 {
		pbt, ok := ParseBT(parent.Extra)
		if ok {
			return pbt.P
		}
		return 0
	}(), C: childID})
}
