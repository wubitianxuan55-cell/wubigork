package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
)

// TestNovelReadingAsk_Guards 覆盖 AI 伴读的守卫分支（不触发真实 LLM）。
// 第 6 参 historyJSON 传空串 = 旧签名单轮行为，守卫语义不变。
func TestNovelReadingAsk_Guards(t *testing.T) {
	a := newCharacterLibTestApp(t) // client 为 nil 的干净 App

	if _, err := a.NovelReadingAsk("summary", "第1章", "正文", "", "", ""); err == nil || !strings.Contains(err.Error(), "AI 客户端未初始化") {
		t.Fatalf("client 为空时应报未初始化: %v", err)
	}

	// 提供 client 骨架后走内容守卫（守卫分支在调用 LLM 之前返回）
	a.client = &ai.Client{}
	if _, err := a.NovelReadingAsk("summary", "第1章", "   ", "", "", ""); err == nil || !strings.Contains(err.Error(), "本章暂无内容") {
		t.Fatalf("空正文应报错: %v", err)
	}
	if _, err := a.NovelReadingAsk("ask", "第1章", "正文", "摘选", "", ""); err == nil || !strings.Contains(err.Error(), "请输入问题") {
		t.Fatalf("空问题应报错: %v", err)
	}
	if _, err := a.NovelReadingAsk("ask", "第1章", "正文", "", "问题", ""); err == nil || !strings.Contains(err.Error(), "请先摘选原文") {
		t.Fatalf("空摘选应报错: %v", err)
	}
	if _, err := a.NovelReadingAsk("unknown", "第1章", "正文", "", "", ""); err == nil || !strings.Contains(err.Error(), "未知的伴读类型") {
		t.Fatalf("未知类型应报错: %v", err)
	}
}

// TestParseReadingHistory 覆盖历史 JSON 解析：空串/脏数据退回单轮，合法数组解析并清洗。
func TestParseReadingHistory(t *testing.T) {
	// 空串与纯空白 = 单轮（兼容旧签名）
	for _, in := range []string{"", "   "} {
		if got := parseReadingHistory(in); got != nil {
			t.Fatalf("空历史 %q 应返回 nil，得到 %v", in, got)
		}
	}
	// 解析失败/非数组一律忽略历史按单轮走
	for _, in := range []string{"not-json", `{"q":"x","a":"y"}`, `[1,2]`, `["a"]`, `null`} {
		if got := parseReadingHistory(in); got != nil {
			t.Fatalf("非法历史 %q 应忽略返回 nil，得到 %v", in, got)
		}
	}
	// 合法数组：清洗首尾空白，跳过无问题轮次
	got := parseReadingHistory(`[{"q":"  他是谁？ ","a":" 主角。 "},{"a":"没有问题的轮次"},{"q":"","a":"x"},{"q":"后来呢","a":""}]`)
	if len(got) != 2 {
		t.Fatalf("应保留 2 轮（跳过空问题轮），得到 %d 轮: %+v", len(got), got)
	}
	if got[0].Q != "他是谁？" || got[0].A != "主角。" {
		t.Fatalf("轮次字段应去首尾空白: %+v", got[0])
	}
	if got[1].Q != "后来呢" || got[1].A != "" {
		t.Fatalf("空回答轮次应保留: %+v", got[1])
	}
}

// TestTrimReadingHistory 覆盖历史截断：只留最近 6 轮，每轮回答截 500 rune。
func TestTrimReadingHistory(t *testing.T) {
	turns := make([]readingTurn, 0, 8)
	for i := 0; i < 8; i++ {
		turns = append(turns, readingTurn{Q: fmt.Sprintf("问题%d", i), A: fmt.Sprintf("回答%d", i)})
	}
	got := trimReadingHistory(turns)
	if len(got) != readingMaxHistoryTurns {
		t.Fatalf("应只保留最近 %d 轮，得到 %d 轮", readingMaxHistoryTurns, len(got))
	}
	if got[0].Q != "问题2" || got[len(got)-1].Q != "问题7" {
		t.Fatalf("应保留最近轮次（问题2..问题7）: %+v", got)
	}
	// 回答超长截断到 500 rune（含省略号，对齐 truncateRunes 语义）
	long := strings.Repeat("长", 600)
	trimmed := trimReadingHistory([]readingTurn{{Q: "q", A: long}})
	if n := len([]rune(trimmed[0].A)); n != readingMaxHistoryARunes {
		t.Fatalf("回答应截为 %d rune（含省略号），实际 %d rune", readingMaxHistoryARunes, n)
	}
	if !strings.HasSuffix(trimmed[0].A, "…") {
		t.Fatalf("截断回答应以省略号结尾: %q", trimmed[0].A)
	}
	// 500 rune 以内不截断
	ok := trimReadingHistory([]readingTurn{{Q: "q", A: strings.Repeat("短", 100)}})
	if ok[0].A != strings.Repeat("短", 100) {
		t.Fatalf("短回答不应截断")
	}
}

// TestReadingSelectionWindow 覆盖划线窗口截断：全文放得下原样返回；超限时保完整
// 划线 + 前后各 ~3000 rune；定位不到返回空串退回旧行为。
func TestReadingSelectionWindow(t *testing.T) {
	// 短文（窗口内）原样返回
	if got := readingSelectionWindow("短正文", "正文"); got != "短正文" {
		t.Fatalf("窗口内全文应原样返回: %q", got)
	}

	// 长文：划线在正中，应保完整划线 + 前后 ~3000 rune
	var b strings.Builder
	for i := 0; i < 12000; i++ {
		b.WriteRune(rune('甲' + i%26))
	}
	b.WriteString("【目标划线】")
	for i := 0; i < 12000; i++ {
		b.WriteRune(rune('乙' + i%26))
	}
	text := b.String()
	sel := "【目标划线】"
	got := readingSelectionWindow(text, sel)
	if !strings.Contains(got, sel) {
		t.Fatalf("窗口必须完整包含划线")
	}
	runes := []rune(got)
	if len(runes) > readingMaxContextRunes+len("……（前文略）\n")+len("\n……（后文略）") {
		t.Fatalf("窗口超上限: %d rune", len(runes))
	}
	idx := strings.Index(got, sel)
	if idx < 0 || idx < readingSelWindowRunes-len("……（前文略）\n")-100 {
		t.Fatalf("划线前应保留 ~%d rune 上下文（实际前缀 %d 字节）", readingSelWindowRunes, idx)
	}

	// 划线贴着开头：不越界，窗口从 0 开始
	head := "【开头划线】" + strings.Repeat("丙", 20000)
	got = readingSelectionWindow(head, "【开头划线】")
	if !strings.HasPrefix(got, "【开头划线】") {
		t.Fatalf("开头划线应从 0 开始: %q", truncateRunes(got, 30))
	}

	// 划线贴着结尾：不越界
	tail := strings.Repeat("丁", 20000) + "【结尾划线】"
	got = readingSelectionWindow(tail, "【结尾划线】")
	if !strings.HasSuffix(got, "【结尾划线】") {
		t.Fatalf("结尾划线应保留到文末")
	}

	// 定位不到 → 空串（调用方退回仅摘选原文）
	if got := readingSelectionWindow(text, "不存在的划线"); got != "" {
		t.Fatalf("定位不到应返回空串: %q", truncateRunes(got, 30))
	}

	// 跨行选区：前端把换行折叠为空格，归一化匹配应能定位并保住窗口
	para1 := strings.Repeat("戊", 11000)
	para2 := "需要被找到的句子" + strings.Repeat("己", 11000) + "末段。"
	multiline := para1 + "\n\n" + para2
	crossSel := strings.Repeat("戊", 5) + " 需要被找到的句子" // 折叠后的跨行选区
	got = readingSelectionWindow(multiline, crossSel)
	if got == "" || !strings.Contains(got, "需要被找到的句子") {
		t.Fatalf("归一化匹配应定位跨行选区: %q", truncateRunes(got, 40))
	}
}

// TestReadingAskUserPrompt_SingleTurnCompat 空历史时 user prompt 与旧单轮格式逐字一致；
// 带历史时包含【此前对话】段。
func TestReadingAskUserPrompt_SingleTurnCompat(t *testing.T) {
	got := readingAskUserPrompt("第1章", "摘选内容", "这是什么？", nil)
	want := "章节：第1章\n\n【摘选原文】\n摘选内容\n\n【问题】这是什么？"
	if got != want {
		t.Fatalf("空历史应与旧单轮格式一致:\n got=%q\nwant=%q", got, want)
	}

	history := trimReadingHistory(parseReadingHistory(`[{"q":"他是谁","a":"主角"},{"q":"那他后来呢","a":"远行了"}]`))
	got = readingAskUserPrompt("第1章", "窗口", "那后来呢？", history)
	if !strings.Contains(got, "【此前对话】") {
		t.Fatalf("带历史应包含【此前对话】段: %q", got)
	}
	if !strings.Contains(got, "用户：他是谁\n助手：主角") || !strings.Contains(got, "用户：那他后来呢\n助手：远行了") {
		t.Fatalf("历史对白渲染不完整: %q", got)
	}
	if !strings.Contains(got, "【问题】那后来呢？") {
		t.Fatalf("当前问题应保留: %q", got)
	}
}
