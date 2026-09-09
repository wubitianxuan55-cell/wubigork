// config_prefs.go — 布尔偏好域（P4 纯结构拆分）。
// SensitiveLocal / OfficeLocal / ReadScreen / IntentsLLM / OfflineMode /
// EngineFailover / KeepWarm / AutoPreload / MorningPreload 读写方法原样保留。

package config

// GetSensitiveLocal 读取敏感域本地化开关（未显式配置时默认开启）。
func (c *Config) GetSensitiveLocal() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.SensitiveLocal
}

// SetSensitiveLocal 写入敏感域本地化开关（true=成本/报价 AI 走本地 Herdsman）。
func (c *Config) SetSensitiveLocal(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.SensitiveLocal = enabled
}

// GetOfficeLocal 读取办公本地优先开关（未显式配置时默认开启）。
func (c *Config) GetOfficeLocal() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.OfficeLocal
}

// SetOfficeLocal 写入办公本地优先开关（true=办公功能级 AI 调用走本地 Herdsman）。
func (c *Config) SetOfficeLocal(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.OfficeLocal = enabled
}

// GetReadScreenSummary 读取读屏摘要开关（未显式配置时默认开启）。
func (c *Config) GetReadScreenSummary() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.ReadScreenSummary
}

// SetReadScreenSummary 写入读屏摘要开关（true=OCR 长文本本地摘要后朗读）。
func (c *Config) SetReadScreenSummary(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.ReadScreenSummary = enabled
}

// GetReadScreenKeepLast 读取读屏留档开关（未显式配置时默认关闭）。
func (c *Config) GetReadScreenKeepLast() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.ReadScreenKeepLast
}

// SetReadScreenKeepLast 写入读屏留档开关（true=最近读屏截图覆盖存 exports）。
func (c *Config) SetReadScreenKeepLast(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.ReadScreenKeepLast = enabled
}

// GetIntentsLLMFallback 读取意图 LLM 兜底分类开关（未显式配置时默认关闭）。
func (c *Config) GetIntentsLLMFallback() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.IntentsLLMFallback
}

// SetIntentsLLMFallback 写入意图 LLM 兜底分类开关（true=规则未命中时轻量
// LLM 分类兜底，白名单+置信门受控）。
func (c *Config) SetIntentsLLMFallback(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.IntentsLLMFallback = enabled
}

// GetIntentsLLMTimeoutMS 读取意图 LLM 兜底硬超时（毫秒，默认 2000）。
func (c *Config) GetIntentsLLMTimeoutMS() int {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.IntentsLLMTimeoutMS
}

// GetOfflineMode 读取全局离线模式开关（未显式配置时默认关闭）。
func (c *Config) GetOfflineMode() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.OfflineMode
}

// SetOfflineMode 写入全局离线模式开关（true=所有 AI 路由只允许本地引擎）。
func (c *Config) SetOfflineMode(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.OfflineMode = enabled
}

// GetEngineFailover 读取引擎故障转移开关（C 刀 v0，未显式配置时默认关闭）。
func (c *Config) GetEngineFailover() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.EngineFailoverEnabled
}

// SetEngineFailover 写入引擎故障转移开关（true=聊天请求首请求网络类失败时
// 换候选引擎重试一次）。
func (c *Config) SetEngineFailover(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.EngineFailoverEnabled = enabled
}

// SetIntentsLLMTimeoutMS 写入意图 LLM 兜底硬超时（毫秒，200-60000）。
func (c *Config) SetIntentsLLMTimeoutMS(ms int) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.IntentsLLMTimeoutMS = ms
}

// GetKeepWarm 读取本地模型保活开关（T5-3a，未显式配置时默认开启）。
func (c *Config) GetKeepWarm() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.KeepWarmEnabled
}

// SetKeepWarm 写入本地模型保活开关（true=周期性探活已运行模型）。
func (c *Config) SetKeepWarm(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.KeepWarmEnabled = enabled
}

// GetAutoPreload 读取启动自动预载开关（T5-3b，未显式配置时默认开启）。
func (c *Config) GetAutoPreload() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.AutoPreload
}

// SetAutoPreload 写入启动自动预载开关。
func (c *Config) SetAutoPreload(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.AutoPreload = enabled
}

// GetMorningPreload 读取晨报预载开关（v4.16 刀④，未显式配置时默认开启）。
// 只影响上下文注入（work 空间预装配高频工作记忆），与晨报卡片无关。
func (c *Config) GetMorningPreload() bool {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.MorningPreload
}

// SetMorningPreload 写入晨报预载开关（仅 config 文件可控，无 UI 绑定）。
func (c *Config) SetMorningPreload(enabled bool) {
	funcMu.Lock()
	defer funcMu.Unlock()
	c.MorningPreload = enabled
}
