package app

import "github.com/gaea/gaea/internal/ai"

// ChatGeneral 通用 AI 聊天（不需要打开项目；使用聊天功能绑定的模型，未绑定用全局）
// 注意：前端未直接调用此绑定，但 module_bindings.go 的「主脑(gaea)」模块
// Handle 仍经 RunModule 派发依赖它（gen_bindings/main.go:145 亦登记）——不得删除。
func (a *App) ChatGeneral(userMsg string) (map[string]interface{}, error) {
	eng, model := a.featureModel("chat")
	reply, err := a.client.ChatSimpleStreamWithOptions(a.ctx, model, "你是一个热心、博学的AI助手，用中文与用户进行日常对话。", userMsg, ai.ChatSimpleOptions{EngineID: eng})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reply": reply,
	}, nil
}
