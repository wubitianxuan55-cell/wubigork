package types

// 风格学习回灌档案（长篇刀5，规格 docs/gaea-longform-novel-system-2026-09.md §1.9：
// 参考档只用于体检打分，生成时不学习作者表达习惯——本档案把统计指纹编译成
// **可执行写作指令**，生成时注入约束）。存 style_digest.json（项目根）。

// StyleDigestFile 风格摘要档。
type StyleDigestFile struct {
	BuiltAt      string   `json:"builtAt"`              // 构建时间（RFC3339）
	Chapters     int      `json:"chapters"`             // 参与构建的章节数
	Chars        int      `json:"chars"`                // 参与构建的非空白字数
	Instructions string   `json:"instructions"`         // 可执行写作指令（生成时注入的正文）
	SignWords    []string `json:"sign_words,omitempty"` // 作者签名词（提示自然融入）
	// 特征快照（对照实验/面板展示用；不进 prompt）
	SentenceMean      float64 `json:"sentence_mean"`
	SentenceSd        float64 `json:"sentence_sd"`
	DialogRatio       float64 `json:"dialog_ratio"`
	FourCharRatio     float64 `json:"four_char_ratio"`
	ConnectiveDensity float64 `json:"connective_density"`
	AdjAdvDensity     float64 `json:"adjadv_density"`
}
