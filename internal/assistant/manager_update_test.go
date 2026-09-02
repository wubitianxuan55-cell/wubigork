package assistant

import (
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

// TestUpdate_EmptyExtensionFieldsPreserveExisting 扩展字段空=保留现值：
// 部分保存（表单只回传核心字段）不得把已有的 WxBotID/PortraitURL/VoiceGuide/
// Gender/Tags/Dims 清空；Name/PersonalityID/WxToken/WxUserID/Enabled 维持照写。
func TestUpdate_EmptyExtensionFieldsPreserveExisting(t *testing.T) {
	m := NewEmpty(t.TempDir())
	if err := m.Add(Assistant{
		ID: "a1", Name: "助手A", PersonalityID: "gaea",
		WxToken: "tok-1", WxUserID: "u1", Enabled: true,
		WxBotID:     "bot-1",
		PortraitURL: "https://example.com/p.png",
		VoiceGuide:  "温柔标准的女声口吻",
		Gender:      "female",
		Tags:        []string{"女主", "古装"},
		Dims:        whisper.PersonalityDims{T: 80, I: 60, S: 40, O: 70, R: 30},
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// 部分保存：扩展字段全空（零值/nil），仅核心字段 + 启停切换
	if err := m.Update("a1", Assistant{
		ID: "a1", Name: "助手A改", PersonalityID: "tsundere",
		WxToken: "tok-2", WxUserID: "u1", Enabled: false,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := m.Get("a1")
	if got == nil {
		t.Fatal("助手 a1 应存在")
	}
	// 核心字段照写
	if got.Name != "助手A改" || got.PersonalityID != "tsundere" || got.WxToken != "tok-2" || got.Enabled {
		t.Fatalf("核心字段应照写, got %+v", got)
	}
	// 扩展字段空=保留现值
	if got.WxBotID != "bot-1" {
		t.Errorf("空 WxBotID 应保留现值, got %q", got.WxBotID)
	}
	if got.PortraitURL != "https://example.com/p.png" {
		t.Errorf("空 PortraitURL 应保留现值, got %q", got.PortraitURL)
	}
	if got.VoiceGuide != "温柔标准的女声口吻" {
		t.Errorf("空 VoiceGuide 应保留现值, got %q", got.VoiceGuide)
	}
	if got.Gender != "female" {
		t.Errorf("空 Gender 应保留现值, got %q", got.Gender)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "女主" || got.Tags[1] != "古装" {
		t.Errorf("nil Tags 应保留现值, got %v", got.Tags)
	}
	if got.Dims != (whisper.PersonalityDims{T: 80, I: 60, S: 40, O: 70, R: 30}) {
		t.Errorf("零值 Dims 应保留现值, got %+v", got.Dims)
	}
}

// TestUpdate_NonEmptyExtensionFieldsWriteBack 扩展字段非空/非 nil 写回生效。
func TestUpdate_NonEmptyExtensionFieldsWriteBack(t *testing.T) {
	m := NewEmpty(t.TempDir())
	if err := m.Add(Assistant{
		ID: "a1", Name: "助手A", PersonalityID: "gaea", Enabled: true,
		WxBotID:     "bot-old",
		PortraitURL: "https://example.com/old.png",
		VoiceGuide:  "旧口吻",
		Gender:      "female",
		Tags:        []string{"旧标签"},
		Dims:        whisper.PersonalityDims{T: 10, I: 10, S: 10, O: 10, R: 10},
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := m.Update("a1", Assistant{
		ID: "a1", Name: "助手A", PersonalityID: "gaea", Enabled: true,
		WxBotID:     "bot-new",
		PortraitURL: "https://example.com/new.png",
		VoiceGuide:  "冷静克制的少年音",
		Gender:      "male",
		Tags:        []string{"侠客", "江湖"},
		Dims:        whisper.PersonalityDims{T: 90, I: 20, S: 30, O: 80, R: 40},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := m.Get("a1")
	if got == nil {
		t.Fatal("助手 a1 应存在")
	}
	if got.WxBotID != "bot-new" {
		t.Errorf("非空 WxBotID 应写回, got %q", got.WxBotID)
	}
	if got.PortraitURL != "https://example.com/new.png" {
		t.Errorf("非空 PortraitURL 应写回, got %q", got.PortraitURL)
	}
	if got.VoiceGuide != "冷静克制的少年音" {
		t.Errorf("非空 VoiceGuide 应写回, got %q", got.VoiceGuide)
	}
	if got.Gender != "male" {
		t.Errorf("非空 Gender 应写回, got %q", got.Gender)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "侠客" || got.Tags[1] != "江湖" {
		t.Errorf("非 nil Tags 应写回, got %v", got.Tags)
	}
	if got.Dims != (whisper.PersonalityDims{T: 90, I: 20, S: 30, O: 80, R: 40}) {
		t.Errorf("非零 Dims 应写回, got %+v", got.Dims)
	}
}

// TestUpdate_WxUserID_MigratesByWxUserMap WxUserID 变更后 byWxUser 映射正确迁移：
// 旧微信用户映射删除、新微信用户命中更新后的助手。
func TestUpdate_WxUserID_MigratesByWxUserMap(t *testing.T) {
	m := NewEmpty(t.TempDir())
	if err := m.Add(Assistant{ID: "a1", Name: "A", WxUserID: "u-old", Enabled: true}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got := m.FindByWxUser("u-old"); got == nil || got.ID != "a1" {
		t.Fatalf("初始映射应命中 a1, got %+v", got)
	}

	if err := m.Update("a1", Assistant{ID: "a1", Name: "A", WxUserID: "u-new", Enabled: true}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := m.FindByWxUser("u-old"); got != nil {
		t.Errorf("旧微信用户映射应删除, got %+v", got)
	}
	if got := m.FindByWxUser("u-new"); got == nil || got.ID != "a1" {
		t.Errorf("新微信用户 u-new 应命中 a1, got %+v", got)
	}

	// WxUserID 照写语义：更新为空串 = 解除绑定，映射随之清除
	if err := m.Update("a1", Assistant{ID: "a1", Name: "A", WxUserID: "", Enabled: true}); err != nil {
		t.Fatalf("Update(空 WxUserID): %v", err)
	}
	if got := m.FindByWxUser("u-new"); got != nil {
		t.Errorf("清空 WxUserID 后映射应删除, got %+v", got)
	}
}
