package session

// 3.0 Step 1: 旧格式迁移。
// Load 检测旧格式（无 .gaea-log.jsonl 而有旧 <id>.jsonl）→ 旧格式读入 →
// 首次保存时写新日志；旧文件保留不清除（3.1 再清理）。
// 迁移在 OpenLog(logPath, legacyPath, space) 里自动触发：日志不存在而旧会话
// 存在时，把旧消息投影为初始日志条目。

import (
	"errors"
	"fmt"
	"os"
)

// HasEventLog 报告会话路径下是否已有事件日志。
func HasEventLog(sessionPath string) bool {
	if sessionPath == "" {
		return false
	}
	_, err := os.Stat(LogPathFor(sessionPath))
	return err == nil
}

// DetectLegacy 判断会话是否处于「旧格式」：没有事件日志、但有旧 <id>.jsonl。
// 返回 (legacy 是否成立, 旧文件路径)。旧文件不存在时返回 false。
func DetectLegacy(sessionPath string) (bool, string, error) {
	if sessionPath == "" {
		return false, "", nil
	}
	if HasEventLog(sessionPath) {
		return false, "", nil
	}
	if _, err := os.Stat(sessionPath); err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, sessionPath, nil
}

// MigrateLegacyToLog 把旧格式会话文件的内容迁移为新事件日志：
// 旧消息 → ToLogEntries → 写入 <logPath>。日志已存在时拒绝（幂等防重）。
// 旧文件保留不清除。space 是迁移条目的空间自描述值（"work"/"play"；""=
// 不写 space 字段），与日志所在目录归属一致（play 分区迁移必须带 play，
// 否则恢复校验会按空间穿越拒绝）。返回迁移的条目数。
func MigrateLegacyToLog(logPath, legacyPath, space string) (int, error) {
	if logPath == "" || legacyPath == "" {
		return 0, errors.New("empty log/legacy path")
	}
	if _, err := os.Stat(logPath); err == nil {
		return 0, errors.New("log already exists, refusing to double-migrate")
	} else if !os.IsNotExist(err) {
		return 0, err
	}
	s, err := Load(legacyPath)
	if err != nil {
		return 0, fmt.Errorf("load legacy session: %w", err)
	}
	entries := ToLogEntries(s.Messages)
	if len(entries) == 0 {
		// 空会话（无内容）不产生日志文件，保持与 listDir 跳过空会话一致。
		return 0, nil
	}
	w, err := OpenLog(logPath, "", space)
	if err != nil {
		return 0, fmt.Errorf("open new log: %w", err)
	}
	defer w.Close()
	for _, e := range entries {
		if _, err := w.AppendRaw(e.Kind, e.Payload); err != nil {
			return 0, fmt.Errorf("append migrated entry: %w", err)
		}
	}
	// 旧文件保留：不做任何删除/改名。
	return len(entries), nil
}
