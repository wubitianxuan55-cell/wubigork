package app

import (
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/office"
)

// 编译期验证：各子服务方法经 App 嵌入提升（前端 window.go.app.* 绑定不变）。
func TestEmbeddingPromotion(t *testing.T) {
	var _ interface {
		// writingState
		ChatCharacter(userMsg string) (map[string]interface{}, error)
		// mediaState
		GenerateFreeImage(prompt, negative, size, style, model string, seed, n int, lora string) (map[string]interface{}, error)
		// whisperState
		WhisperChat(userMsg string, personalityID string) (map[string]interface{}, error)
		// officeState
		OfficeExecute(act, path, tgt, q, url, content string) office.ExecResult
		// core
		GetEngines() []modelengine.EngineConfig
	} = New()
}
