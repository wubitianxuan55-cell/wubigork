// config_realtime.go — 实时语音 Realtime 域（P4 纯结构拆分）。
// RealtimeProvider / RealtimeModel / RealtimeAPIKey 读取方法原样保留。

package config

// GetRealtimeProvider 读取实时语音 provider kind（空 = 未配置实时语音档）。
func (c *Config) GetRealtimeProvider() string {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.RealtimeProvider
}

// GetRealtimeModel 读取实时语音模型 ID（空 = 供应商默认）。
func (c *Config) GetRealtimeModel() string {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.RealtimeModel
}

// GetRealtimeAPIKey 读取实时语音 API Key（存储口径 = secure.EncryptString
// 密文；config 层不做加解密，解密由 app 层 secure.DecryptString 负责）。
func (c *Config) GetRealtimeAPIKey() string {
	funcMu.RLock()
	defer funcMu.RUnlock()
	return c.RealtimeAPIKey
}
