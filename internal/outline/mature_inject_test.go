package outline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
)

// capturingLLM 记录最近一次 ChatSimpleStreamWithOptions 的 system/user（maturecraft
// 注入断言专用；story_thread_test.go 的 stubLLM 不带捕获，互不影响）。
type capturingLLM struct{ system, user string }

func (c *capturingLLM) ChatStream(context.Context, *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	return nil, nil
}
func (c *capturingLLM) ChatSimpleStream(context.Context, string, string, string) (string, error) {
	return "", nil
}
func (c *capturingLLM) ChatSimpleStreamWithOptions(_ context.Context, _, systemPrompt, userMsg string, _ ai.ChatSimpleOptions) (string, error) {
	c.system = systemPrompt
	c.user = userMsg
	return `{"story_thread":"","nodes":[]}`, nil
}
func (c *capturingLLM) GenerateImage(context.Context, *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

// newMatureTestAgent 构造带 mature_craft 槽 Continue 模板的测试 Agent；
// mature 为项目档位（""=非成人向）。
func newMatureTestAgent(t *testing.T, mature string) (*Agent, *capturingLLM) {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试书", "都市", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	pm.Meta.Mature = mature

	dir := filepath.Join(t.TempDir(), "prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"name":"outline-continue","system":"sys","task":"task",` +
		`"input_sections":{"story_thread":{"priority":"P0","label":"故事主线"},` +
		`"mature_craft":{"priority":"P1","order":90,"label":"本书成人向大纲纪律"}},` +
		`"output":{"format":"json"}}`
	if err := os.WriteFile(filepath.Join(dir, "outline-continue.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	client := &capturingLLM{}
	return New(client, pm, &config.Config{}, prompt.NewEngine(dir)), client
}

// TestContinueInjectsMatureOutline 大纲续写链注入断言：成人向项目 user prompt
// 必带大纲纪律与档位口径；非成人向项目零出现。
func TestContinueInjectsMatureOutline(t *testing.T) {
	agent, client := newMatureTestAgent(t, "explicit")
	if _, err := agent.Continue(context.Background(), 3); err != nil {
		t.Fatalf("Continue: %v", err)
	}
	for _, want := range []string{"成人向大纲纪律", "直白", "欲望线是一等叙事线"} {
		if !strings.Contains(client.user, want) {
			t.Errorf("成人向项目大纲 prompt 缺「%s」\n%s", want, client.user)
		}
	}

	plainAgent, plainClient := newMatureTestAgent(t, "")
	if _, err := plainAgent.Continue(context.Background(), 3); err != nil {
		t.Fatalf("Continue(plain): %v", err)
	}
	if strings.Contains(plainClient.user, "成人向") {
		t.Errorf("非成人向项目大纲 prompt 不得出现纪律区段:\n%s", plainClient.user)
	}
}
