package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/chat"
)

// newSinTestApp 复用统一聊天测试 App（mock LLM 返回「你好呀」），并把原罪
// 功能级绑定指向同一 mock 引擎。
func newSinTestApp(t *testing.T) *App {
	t.Helper()
	a := newChatServiceTestApp(t)
	if err := a.core.SetFeatureModel("sin", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(sin): %v", err)
	}
	return a
}

// TestSinTopicsIsolationAndGuard 原罪故事与聊天话题同表不同域：
// SinTopicsList 只见 sin；ChatTopicsList（聊天板块）不见 sin；
// 非 sin 话题经原罪绑定操作必须被拒绝（fail-closed，防跨板块改写）。
func TestSinTopicsIsolationAndGuard(t *testing.T) {
	a := newSinTestApp(t)
	chatTopic, err := a.ChatTopicCreate("闲聊", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	sinTopic, err := a.SinTopicCreate("")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if sinTopic.Mode != sinTopicMode {
		t.Fatalf("新故事 mode = %q, want %q", sinTopic.Mode, sinTopicMode)
	}
	if sinTopic.Title != "新故事" {
		t.Errorf("空标题应回退「新故事」，got %q", sinTopic.Title)
	}

	sins, err := a.SinTopicsList()
	if err != nil {
		t.Fatalf("SinTopicsList: %v", err)
	}
	if len(sins) != 1 || sins[0].ID != sinTopic.ID {
		t.Fatalf("原罪故事列表 = %+v, want 仅 %s", sins, sinTopic.ID)
	}
	chats, err := a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}
	if len(chats) != 1 || chats[0].ID != chatTopic.ID {
		t.Fatalf("聊天话题列表 = %+v, want 仅 %s（sin 话题不应混入）", chats, chatTopic.ID)
	}

	// 跨板块守卫：对聊天话题做原罪操作必须报错，且话题未被改动。
	if err := a.SinTopicRename(chatTopic.ID, "改名"); err == nil {
		t.Error("SinTopicRename(聊天话题) 应报错")
	}
	if err := a.SinTopicDelete(chatTopic.ID); err == nil {
		t.Error("SinTopicDelete(聊天话题) 应报错")
	}
	if _, err := a.SinMessages(chatTopic.ID); err == nil {
		t.Error("SinMessages(聊天话题) 应报错")
	}
	if _, err := a.SinTopicCreate("x"); err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if err := a.SinTopicRename(sinTopic.ID, "夜行电车"); err != nil {
		t.Fatalf("SinTopicRename(原罪话题): %v", err)
	}
	got, err := a.chatStore.GetTopic(sinTopic.ID)
	if err != nil || got.Title != "夜行电车" {
		t.Errorf("重命名未生效: %+v, err=%v", got, err)
	}
}

// TestSinStreamPersistsAndKeepsMessageID 故事流式生成：runID 返回、消息落库、
// 助手消息 id 可定位（前端插图回写依赖它）。
func TestSinStreamPersistsAndKeepsMessageID(t *testing.T) {
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("流式故事")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	runID, err := a.SinStream(topic.ID, "写一个开场")
	if err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	if !strings.HasPrefix(runID, "ss_") {
		t.Errorf("runID = %q, want ss_ 前缀", runID)
	}

	deadline := time.Now().Add(3 * time.Second)
	var msgs []chat.Message
	for time.Now().Before(deadline) {
		msgs, err = a.SinMessages(topic.ID)
		if err == nil && len(msgs) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2（用户 + 助手）", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "写一个开场" {
		t.Errorf("user 消息 = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "你好呀" {
		t.Errorf("assistant 消息 = %+v", msgs[1])
	}
	if msgs[1].ID <= 0 {
		t.Fatalf("助手消息 id = %d, want > 0", msgs[1].ID)
	}
}

// TestSinStreamRejectsUnknownTopic 未注册话题（空/不存在）不得进入生成路径。
func TestSinStreamRejectsUnknownTopic(t *testing.T) {
	a := newSinTestApp(t)
	if _, err := a.SinStream("", "写"); err == nil {
		t.Error("空 storyID 应报错")
	}
	if _, err := a.SinStream("sin_missing", "写"); err == nil {
		t.Error("不存在的话题应报错")
	}
}

// TestSinIllustrationAttachAndExport 插图回写 + 图文 Markdown 导出：
// extra.illustrations 落库后导出为 Markdown 图片，未生成插图的标记保留占位。
func TestSinIllustrationAttachAndExport(t *testing.T) {
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	story := "雨落在窗上。\n" + sinIllustrationCueOpen + "穿黑风衣的女人站在灯下" + sinIllustrationCueClose +
		"\n她回过头。\n" + sinIllustrationCueOpen + "近景特写" + sinIllustrationCueClose
	if err := a.chatStore.AppendExchange(topic.ID, "开始", story, `{"reasoning":"r"}`); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}
	msgs, err := a.SinMessages(topic.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("SinMessages = %+v, err=%v", msgs, err)
	}
	assistant := msgs[1]

	// 第一张插图生成完成：按 cue 键（出现次序，0 起）回写。
	if err := a.sinAttachIllustration(assistant.ID, "0", "C:/tmp/sin-1.png"); err != nil {
		t.Fatalf("sinAttachIllustration: %v", err)
	}
	got, err := a.chatStore.GetMessage(assistant.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(got.Extra), &extra); err != nil {
		t.Fatalf("extra 不是合法 JSON: %q", got.Extra)
	}
	if extra["reasoning"] != "r" {
		t.Errorf("回写插图不应丢失既有 extra 字段: %+v", extra)
	}
	arts, _ := extra[sinIllustrationsExtraKey].(map[string]interface{})
	if arts["0"] != "C:/tmp/sin-1.png" {
		t.Errorf("illustrations = %+v", arts)
	}

	md, err := a.SinExportMarkdown(topic.ID)
	if err != nil {
		t.Fatalf("SinExportMarkdown: %v", err)
	}
	if !strings.Contains(md, "# 雨夜") {
		t.Errorf("导出缺标题: %s", md)
	}
	if !strings.Contains(md, "![穿黑风衣的女人站在灯下](C:/tmp/sin-1.png)") {
		t.Errorf("已生成插图应导出为 Markdown 图片: %s", md)
	}
	if !strings.Contains(md, "（插图未生成：近景特写）") {
		t.Errorf("未生成插图应保留占位: %s", md)
	}
	if strings.Contains(md, sinIllustrationCueOpen) {
		t.Errorf("导出不应残留插图标记: %s", md)
	}
}

// TestBuildSinUserPromptStripsCuesAndCapsHistory 前情装配：插图标记换成
// 占位语、历史条数封顶、超长截断，本次指令原样保留。
func TestBuildSinUserPromptStripsCuesAndCapsHistory(t *testing.T) {
	history := make([]chat.Message, 0, sinHistoryTurns+3)
	for i := 0; i < sinHistoryTurns+3; i++ {
		history = append(history, chat.Message{Role: "assistant", Content: "段" + string(rune('A'+i))})
	}
	history = append(history, chat.Message{
		Role:    "assistant",
		Content: "他推开门。\n" + sinIllustrationCueOpen + "门缝里的光" + sinIllustrationCueClose,
	})
	prompt := buildSinUserPrompt(history, "继续写", nil, sinNotesDoc{})
	if strings.Contains(prompt, sinIllustrationCueOpen) {
		t.Errorf("前情不应携带插图标记: %s", prompt)
	}
	if !strings.Contains(prompt, "（已配图）") {
		t.Errorf("插图标记应替换为占位语: %s", prompt)
	}
	if !strings.Contains(prompt, "【本次指令】\n继续写") {
		t.Errorf("本次指令缺失: %s", prompt)
	}
	// 条数封顶：窗口取最近 sinHistoryTurns 条，最早的几条（段A～段D）应被裁掉。
	if strings.Contains(prompt, "段A") || strings.Contains(prompt, "段D") {
		t.Errorf("超出 %d 条的历史应被裁掉: %s", sinHistoryTurns, prompt)
	}
	if !strings.Contains(prompt, "段E") {
		t.Errorf("窗口内的历史不应丢失: %s", prompt)
	}
}

// TestSinSystemPromptHasHardBoundaries 提示词必须写明硬边界与插图协议
// （改文案时防回归：这两段是产品口径的落点）。
func TestSinSystemPromptHasHardBoundaries(t *testing.T) {
	p := sinSystemPrompt()
	for _, want := range []string{"成年人", "合意", "真实在世人物", sinIllustrationCueOpen, "600～1200 字"} {
		if !strings.Contains(p, want) {
			t.Errorf("系统提示词缺少 %q", want)
		}
	}
}
