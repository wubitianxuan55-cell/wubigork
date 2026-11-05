package whisper

import (
	"testing"
	"time"
)

// 众神殿刀2（进度计划/gaea-chat-persona-optimization-20261105.md §1.1）：生日
// 显式化——旧版 PreLLMTurn 把 nil 生日无条件写成当前时间，首聊日第 2 轮起经
// temporal_date.go 确定性注入假事实「我的生日」（存量 6/6 会话
// ackemBirthday==firstMetDate 实证）。修复后 gaea 显式接内置 06-11，其余角色
// 默认不注入。

func TestPreLLMTurn_GaeaBirthdayExplicit(t *testing.T) {
	orch := NewOrchestrator("test_birthday_gaea", *GetPreset("gaea"))
	res := orch.PreLLMTurn("你好")
	if res.NewState.AckemBirthday == nil {
		t.Fatal("gaea 首轮应显式写入内置生日 06-11")
	}
	bd := *res.NewState.AckemBirthday
	if bd.Month() != time.June || bd.Day() != 11 {
		t.Fatalf("gaea 生日应为 06-11，got %s", bd.Format("01-02"))
	}
}

func TestPreLLMTurn_NonGaeaNoImplicitBirthday(t *testing.T) {
	orch := NewOrchestrator("test_birthday_other", *GetPreset("deredere"))
	res := orch.PreLLMTurn("你好")
	if res.NewState.AckemBirthday != nil {
		t.Fatalf("非 gaea 角色不应隐式获得生日，got %v", res.NewState.AckemBirthday)
	}
}

func TestParseAckemBirthdayMMDD(t *testing.T) {
	base := time.Date(2026, 8, 6, 17, 5, 34, 0, time.Local)
	bd, err := ParseAckemBirthdayMMDD("06-11", base)
	if err != nil {
		t.Fatalf("ParseAckemBirthdayMMDD: %v", err)
	}
	if bd.Month() != time.June || bd.Day() != 11 || bd.Year() != 2026 {
		t.Fatalf("解析结果 = %s, want 2026-06-11", bd.Format("2006-01-02"))
	}
	if _, err := ParseAckemBirthdayMMDD("13-11", base); err == nil {
		t.Fatal("月份越界应报错")
	}
	if _, err := ParseAckemBirthdayMMDD("0611", base); err == nil {
		t.Fatal("缺分隔符应报错")
	}
}
