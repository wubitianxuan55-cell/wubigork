// config_features.go — 功能级模型绑定域（P4 纯结构拆分）。
// GetFeatureModel / SetFeatureModel / Get·SetFeatureModelEnabled 原样保留。

package config

// GetFeatureModel 读取功能绑定的 (engine, model)，空 = 用全局激活
func (c *Config) GetFeatureModel(feature string) (engine, model string) {
	funcMu.RLock()
	defer funcMu.RUnlock()
	switch feature {
	case "chat":
		return c.FuncChatEngine, c.FuncChatModel
	case "whisper":
		// 2.x 聊天/轻语合并：轻语绑定并入聊天，查询走 chat 别名
		return c.FuncChatEngine, c.FuncChatModel
	case "novel":
		return c.FuncNovelEngine, c.FuncNovelModel
	case "office":
		return c.FuncOfficeEngine, c.FuncOfficeModel
	case "gaea":
		return c.FuncGaeaEngine, c.FuncGaeaModel
	case "characterlib":
		return c.FuncCharLibEngine, c.FuncCharLibModel
	case "routine":
		return c.FuncRoutineEngine, c.FuncRoutineModel
	case "sin":
		// 原罪（闲庭·图文故事创作）：故事文本生成的独立绑定
		return c.FuncSinEngine, c.FuncSinModel
	}
	return "", ""
}

// SetFeatureModel 写入功能绑定的 (engine, model)
func (c *Config) SetFeatureModel(feature, engine, model string) {
	funcMu.Lock()
	defer funcMu.Unlock()
	switch feature {
	case "chat":
		c.FuncChatEngine, c.FuncChatModel = engine, model
	case "whisper":
		// 2.x 聊天/轻语合并：写入 chat 绑定
		c.FuncChatEngine, c.FuncChatModel = engine, model
	case "novel":
		c.FuncNovelEngine, c.FuncNovelModel = engine, model
	case "office":
		c.FuncOfficeEngine, c.FuncOfficeModel = engine, model
	case "gaea":
		c.FuncGaeaEngine, c.FuncGaeaModel = engine, model
	case "characterlib":
		c.FuncCharLibEngine, c.FuncCharLibModel = engine, model
	case "routine":
		c.FuncRoutineEngine, c.FuncRoutineModel = engine, model
	case "sin":
		c.FuncSinEngine, c.FuncSinModel = engine, model
	}
}

// GetFeatureModelEnabled 读取功能级启停状态（未显式配置时默认启用）。
func (c *Config) GetFeatureModelEnabled(feature string) bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	switch feature {
	case "chat":
		return c.FuncChatEnabled
	case "whisper":
		return c.FuncChatEnabled
	case "novel":
		return c.FuncNovelEnabled
	case "office":
		return c.FuncOfficeEnabled
	case "gaea":
		return c.FuncGaeaEnabled
	case "characterlib":
		return c.FuncCharLibEnabled
	case "routine":
		return c.FuncRoutineEnabled
	case "sin":
		return c.FuncSinEnabled
	}
	return true
}

// SetFeatureModelEnabled 写入功能级启停状态。
func (c *Config) SetFeatureModelEnabled(feature string, enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	switch feature {
	case "chat":
		c.FuncChatEnabled = enabled
	case "whisper":
		c.FuncChatEnabled = enabled
	case "novel":
		c.FuncNovelEnabled = enabled
	case "office":
		c.FuncOfficeEnabled = enabled
	case "gaea":
		c.FuncGaeaEnabled = enabled
	case "characterlib":
		c.FuncCharLibEnabled = enabled
	case "routine":
		c.FuncRoutineEnabled = enabled
	case "sin":
		c.FuncSinEnabled = enabled
	}
}
