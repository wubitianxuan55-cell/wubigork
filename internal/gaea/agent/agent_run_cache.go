package agent

import "strings"

// cachedResultMarker 同回合内重复的只读调用被去重时，回给模型的占位文本。
//
// 去重键里含结果哈希：只有新旧结果逐字节相同才会走到这里，数据变了的调用不会被
// 这行遮住。提示语必须把这一点说清楚——只写一句「cached」会被模型读成「读取失败 /
// 改动未生效」而重复改库（2026-11-24 方案库排错实弹：改完库再搜，只见 cached 不见
// 数据）。末尾带结果开头，供模型当场核对条数与字段。
func cachedResultMarker(name, result string) string {
	head := strings.Join(strings.Fields(result), " ")
	if r := []rune(head); len(r) > 160 {
		head = string(r[:160]) + "…"
	}
	return "[cached — 本回合早先同一条 " + name + " 调用（参数相同）返回过逐字节相同的结果，完整内容见上；" +
		"这说明底层数据对该查询没有变化，不是读取失败。结果开头：" + head + "]"
}
