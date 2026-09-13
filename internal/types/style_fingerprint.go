package types

import "encoding/json"

// StyleFingerprintFile 小说工程文风指纹参考档（<projectDir>/fingerprint.json）。
// 记录构建时点与样本规模，Fingerprint 存 novelstyle.Fingerprint 的原样 JSON
// （types 不反向依赖 novelstyle，解析归 app 层 LoadFingerprint）。
type StyleFingerprintFile struct {
	BuiltAt     string          `json:"builtAt"`     // 构建时间（RFC3339）
	Chapters    int             `json:"chapters"`    // 参与构建的章节数
	Chars       int             `json:"chars"`       // 参与构建的非空白字数
	Fingerprint json.RawMessage `json:"fingerprint"` // novelstyle.Fingerprint 原样序列化
}
