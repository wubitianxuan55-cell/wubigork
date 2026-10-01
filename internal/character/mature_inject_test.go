package character

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/prompt"
)

// matureCapturingClient 记录 prompt 并返回带代码围栏的合法角色文件 JSON
// （RetryJSON 的判据把「纯 JSON 无包裹」当未找到，共享桩 capturingClient 的
// 裸 JSON 回复过不了它——本桩包一层 ```json 围栏）。
type matureCapturingClient struct {
	capturingClient
}

func (c *matureCapturingClient) ChatSimpleStreamWithOptions(_ context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	c.model = model
	c.engine = opts.EngineID
	c.system = systemPrompt
	c.user = userMsg
	return "```json\n{\"characters\": []}\n```", nil
}

// TestGenerateCharactersInjectsMatureCharacter 项目级批量角色生成链注入断言：
// 成人向项目 user prompt 必带角色纪律（为亲密戏供材）；非成人向项目零出现。
// 全局角色库级（charlib）调用不带项目档位，同样零渲染——本用例的 plain 分支即其等价形态。
func TestGenerateCharactersInjectsMatureCharacter(t *testing.T) {
	run := func(t *testing.T, mature string) string {
		t.Helper()
		agent, _, pm := newTestAgent(t)
		client := &matureCapturingClient{}
		agent.client = client
		pm.Meta.Mature = mature
		// 换上带 mature_craft 槽的批量模板（newTestAgent 只写了 character-agent/detail）
		tplDir := filepath.Join(t.TempDir(), "prompts")
		if err := os.MkdirAll(tplDir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := `{"name":"character-generate-batch","system":"sys","task":"task",` +
			`"input_sections":{"genre_and_count":{"priority":"P0","label":"题材与数量"},` +
			`"worldview":{"priority":"P1","label":"世界观"},` +
			`"mature_craft":{"priority":"P1","order":90,"label":"本书成人向角色纪律"}},` +
			`"output":{"format":"json"}}`
		if err := os.WriteFile(filepath.Join(tplDir, "character-generate-batch.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		agent.eng = prompt.NewEngine(tplDir)
		if _, err := agent.GenerateCharacters(context.Background(), 2, "都市"); err != nil {
			t.Fatalf("GenerateCharacters: %v", err)
		}
		return client.user
	}

	user := run(t, "sensual")
	for _, want := range []string{"成人向角色纪律", "含蓄", "欲望线与亲密张力来源"} {
		if !strings.Contains(user, want) {
			t.Errorf("成人向项目角色 prompt 缺「%s」\n%s", want, user)
		}
	}

	plain := run(t, "")
	if strings.Contains(plain, "成人向") {
		t.Errorf("非成人向项目角色 prompt 不得出现纪律区段:\n%s", plain)
	}
}
