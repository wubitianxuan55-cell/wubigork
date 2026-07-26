package app

// ChatGeneral 通用 AI 聊天（不需要打开项目）
func (a *App) ChatGeneral(userMsg string) (map[string]interface{}, error) {
	reply, err := a.client.ChatSimpleStream(a.ctx, "", "你是一个热心、博学的AI助手，用中文与用户进行日常对话。", userMsg)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reply": reply,
	}, nil
}
