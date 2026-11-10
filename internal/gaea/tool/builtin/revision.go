package builtin

// Revision CAS 修订原语（A3，dsh-ops revision 蒸馏，2026-11-10）：
//   - read_file 文本读成功时尾缀 `[revision sha256:<hex>]`（模型取修订值的来源）；
//   - 写族（write_file/edit_file/multi_edit/edit_lines）接受可选 `expected_revision`：
//     不传=零行为变化；"absent"=要求目标不存在（create-only）；64hex=要求当前
//     内容哈希相等。不匹配→结构化 REVISION_CONFLICT（带当前修订值，模型重读
//     即自愈）。
//
// 防「基于陈旧读的写」（多执行体同树 stale-Edit，双会话合流坑在册）；与证据链
// BaselinePath 互补（那是回滚原料，这是前置冲突闸）。TOCTOU 窗口如实记档：校验
// 在 mutate 前一刻执行，进程内单线程 agent 循环下窗口可忽略；跨进程竞争由
// atomicWrite 原子发布兜底。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// RevisionAbsent expected_revision 的 create-only 哨兵值。
const RevisionAbsent = "absent"

// RevisionSHA256 计算文件原始字节的 SHA-256（读不到/目录→错误）。
func RevisionSHA256(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return RevisionOfBytes(raw), nil
}

// RevisionOfBytes 字节哈希（读族校验与尾标同源）。
func RevisionOfBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// checkExpectedRevision CAS 校验。path 仅用于错误文案；current 为当前内容
// （不存在时 nil）；exists 报告目标当前是否存在。expected 语义见文件头注释。
func checkExpectedRevision(path string, current []byte, exists bool, expected string) error {
	if expected == "" {
		return nil
	}
	if expected == RevisionAbsent {
		if exists {
			return fmt.Errorf("REVISION_CONFLICT: %s 已存在（expected_revision=%q 要求 create-only）；先读现有内容再决定覆盖或换路径", path, expected)
		}
		return nil
	}
	if len(expected) != 64 {
		return fmt.Errorf("expected_revision 非法（%q）：须为 64 位 sha256 hex（read_file 尾标 revision 取值）或 %q", expected, RevisionAbsent)
	}
	if !exists {
		return fmt.Errorf("REVISION_CONFLICT: %s 不存在（当前 revision=absent，传入 %s…）；如要新建请不传 expected_revision 或传 %q", path, expected[:12], RevisionAbsent)
	}
	actual := RevisionOfBytes(current)
	if actual != expected {
		return fmt.Errorf("REVISION_CONFLICT: %s 自读取后已被修改（期望 %s…，当前 %s…）；重读文件后再改", path, expected[:12], actual[:12])
	}
	return nil
}
