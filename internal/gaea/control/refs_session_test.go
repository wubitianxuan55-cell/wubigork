package control

// 阶段十 10.3 session-reference 测试：@会话:<关键词> 匹配（标题/文件名、唯一
// 命中、歧义列候选、未找到报错）+ 快照投影（user 全文/assistant 截断/截断注
// 明）+ dsh 原案「label 取最新标题」。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func seedSessionRefEnv(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	// 双文件形态（与生产一致）：主 .jsonl 用旧格式行（role/content——List 的
	// previewSession 按它数轮次，turns=0 的会话被跳过）；事件日志 .gaea-log.jsonl
	// 用新格式（kind/payload——ReadEntriesFor 优先读它做投影）。
	path := filepath.Join(dir, "20261014-deploy.jsonl")
	logPath := filepath.Join(dir, "20261014-deploy.gaea-log.jsonl")

	var legacy strings.Builder
	for _, l := range []struct{ role, content string }{
		{"user", "帮我把部署脚本改成幂等"},
		{"assistant", strings.Repeat("部署脚本已改，要点如下。", 60)},
		{"user", "回滚步骤也补一下"},
		{"assistant", "回滚步骤已补充。"},
	} {
		raw, _ := json.Marshal(map[string]string{"role": l.role, "content": l.content})
		legacy.Write(raw)
		legacy.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(legacy.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []session.LogEntry{
		{Seq: 1, Kind: session.KindUserMessage, Payload: mustJSON(map[string]string{"content": "帮我把部署脚本改成幂等"})},
		{Seq: 2, Kind: session.KindAssistantMessage, Payload: mustJSON(map[string]string{"text": strings.Repeat("部署脚本已改，要点如下。", 60)})},
		{Seq: 3, Kind: session.KindUserMessage, Payload: mustJSON(map[string]string{"content": "回滚步骤也补一下"})},
		{Seq: 4, Kind: session.KindAssistantMessage, Payload: mustJSON(map[string]string{"text": "回滚步骤已补充。"})},
	}
	var b strings.Builder
	for i, e := range entries {
		e.Seq = int64(i + 1)
		e.Ts = int64(1700000000 + i)
		raw, _ := json.Marshal(e)
		b.Write(raw)
		b.WriteString("\n")
	}
	if err := os.WriteFile(logPath, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	// 标题注册表（dsh 原案「label 取最新标题」）
	if err := os.WriteFile(filepath.Join(dir, ".titles.json"),
		[]byte(`{"`+filepath.Base(path)+`": "部署脚本改造"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

func newSessionRefController(dir string) *Controller {
	return &Controller{sessionDir: dir}
}

func TestSessionRef_TitleHit(t *testing.T) {
	dir, _ := seedSessionRefEnv(t)
	c := newSessionRefController(dir)

	block, errMsg := c.resolveSessionRef("@会话:部署脚本")
	if errMsg != "" {
		t.Fatalf("标题命中应解析: %s", errMsg)
	}
	if !strings.Contains(block, `title="部署脚本改造"`) {
		t.Fatalf("应取 titles 注册表最新标题:\n%s", block)
	}
	if !strings.Contains(block, "[user] 帮我把部署脚本改成幂等") {
		t.Fatalf("user 消息应全文注入:\n%s", block)
	}
	// assistant 长文本截断（512 rune 上限）
	if !strings.Contains(block, "…（截断）") {
		t.Fatalf("超长 assistant 应截断:\n%s", block)
	}
	if !strings.Contains(block, "共 2 条用户消息") {
		t.Fatalf("应报用户消息数:\n%s", block)
	}
}

func TestSessionRef_NameHitAndMissAndAmbiguous(t *testing.T) {
	dir, _ := seedSessionRefEnv(t)
	// 第二个会话（文件名命中样本）：旧格式行有用户消息（List 数轮次，空会话被跳过）
	path2 := filepath.Join(dir, "20261015-review.jsonl")
	if err := os.WriteFile(path2, []byte(`{"role":"user","content":"review 一下"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := newSessionRefController(dir)

	// 文件名唯一命中
	if block, errMsg := c.resolveSessionRef("@会话:20261015-review"); errMsg != "" || !strings.Contains(block, "referenced-session") {
		t.Fatalf("文件名命中: block=%q err=%q", block, errMsg)
	}
	// 未找到 → 人可读错误
	if _, errMsg := c.resolveSessionRef("@会话:不存在的会话"); errMsg == "" || !strings.Contains(errMsg, "未找到") {
		t.Fatalf("未找到应报错: %q", errMsg)
	}
	// 歧义：两个会话标题/名字都含「2026」→ 列出候选
	_, errMsg := c.resolveSessionRef("@会话:2026")
	if errMsg == "" || !strings.Contains(errMsg, "候选") {
		t.Fatalf("歧义应列候选: %q", errMsg)
	}
}

func TestSessionRef_WiredThroughResolveRefs(t *testing.T) {
	dir, _ := seedSessionRefEnv(t)
	c := newSessionRefController(dir)
	// ResolveRefs 全链：@会话:xxx 与 @文件 引用同线解析（会话分支不破坏既有引用）
	block, errs := c.ResolveRefs(nil, "参考 @会话:部署脚本")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if !strings.Contains(block, "referenced-session") {
		t.Fatalf("ResolveRefs 应含会话引用块:\n%s", block)
	}
	// classifyRef：非会话前缀的未知 token 不误判为会话
	if _, ok := classifyRef("部署脚本", nil, func(string) bool { return false }); ok {
		t.Fatal("无前缀未知 token 不得判为会话引用")
	}
	if _, ok := classifyRef("会话:部署脚本", nil, func(string) bool { return false }); !ok {
		t.Fatal("会话前缀应判为会话引用（显式前缀优先于 exists 检查）")
	}
	// 守卫：非 user/assistant 条目（如 tool_result）不投影
	e := session.LogEntry{Kind: session.KindToolResult, Payload: mustJSON(map[string]string{"x": "y"})}
	if _, _, ok := sessionContentOf(e); ok {
		t.Fatal("tool_result 不得投影")
	}
}
