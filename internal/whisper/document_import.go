// Package whisper — document_import.go
// 100% 对齐 ackem memory/documentImport/
// 文档导入管线：解析→分块→LLM抽取→记忆落地
package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// DocumentImportResult 文档导入结果
type DocumentImportResult struct {
	FileName      string `json:"fileName"`
	Chunks        int    `json:"chunks"`
	FactsFound    int    `json:"factsFound"`
	EpisodesFound int    `json:"episodesFound"`
	Error         string `json:"error,omitempty"`
}

// ImportDocument 导入单个文档到记忆系统
// 流程：读取文件 → 分块 → LLM 抽取事实 → 写入 FactStore
func ImportDocument(filePath, dataRoot, sessionID string, llm LlmClient, fs *FactStore) *DocumentImportResult {
	result := &DocumentImportResult{FileName: filepath.Base(filePath)}

	// 1. 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("读取文件失败: %v", err)
		return result
	}

	// 2. 分块
	chunks := chunkDocument(string(data), 2000)
	result.Chunks = len(chunks)
	if len(chunks) == 0 {
		result.Error = "文件内容为空"
		return result
	}

	// 3. LLM 逐块抽取事实
	for i, chunk := range chunks {
		facts, err := extractFactsFromChunk(chunk, i+1, len(chunks), llm, filepath.Base(filePath))
		if err != nil {
			continue
		}

		for _, f := range facts {
			fs.Add(f)
			result.FactsFound++
		}
	}

	return result
}

// chunkDocument 将文本分成指定大小的块（按段落边界）
func chunkDocument(text string, chunkSize int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	paragraphs := strings.Split(text, "\n")
	current := ""

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(current)+len(p) > chunkSize && current != "" {
			chunks = append(chunks, strings.TrimSpace(current))
			current = p
		} else {
			if current != "" {
				current += "\n"
			}
			current += p
		}
	}
	if current != "" {
		chunks = append(chunks, strings.TrimSpace(current))
	}

	return chunks
}

// extractFactsFromChunk 从文档块中抽取记忆事实
func extractFactsFromChunk(chunk string, chunkIdx, totalChunks int, llm LlmClient, fileName string) ([]MemoryFact, error) {
	sysPrompt := `你从用户提供的文档中抽取关于用户本人的结构化记忆事实。

规则：
- 只抽取关于用户本人的信息（身份、经历、偏好、习惯、关系、价值观等）
- 领域选择：BASIC_PROFILE(姓名/年龄/性别)、LIFE_STORY(重要经历)、FAMILY(家庭)、TASTES(喜好)、GOALS(目标)、VALUES_BELIEFS(价值观)、VULNERABILITIES(脆弱面)、OUR_BOND(与gaea关系)
- 每条事实一行，格式：domain|subcategory|subject|summary|weight|confidence
- weight: 1(琐碎)-10(重大人生事件)
- confidence: 0.3(推测)-1.0(明确陈述)
- 不要编造，只抽取明确提到的信息

输出格式（每行一条）：
BASIC_PROFILE|BASIC_PROFILE|用户姓名|张三|8|1.0
LIFE_STORY|LIFE_STORY|大学经历|毕业于清华大学计算机系|5|0.9`

	userPrompt := fmt.Sprintf("文档：%s（第%d/%d块）\n\n%s", fileName, chunkIdx, totalChunks, chunk)

	raw, err := llm.Chat(sysPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return parseDocumentFacts(raw, fileName, chunkIdx), nil
}

// parseDocumentFacts 解析 LLM 输出的管道格式事实
func parseDocumentFacts(raw, fileName string, chunkIdx int) []MemoryFact {
	var facts []MemoryFact
	lines := strings.Split(strings.TrimSpace(raw), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		domain := strings.TrimSpace(parts[0])
		subcategory := strings.TrimSpace(parts[1])
		subject := strings.TrimSpace(parts[2])
		summary := strings.TrimSpace(parts[3])
		weight := 3.0
		confidence := 0.7

		if len(parts) >= 5 {
			if w, err := parseFloat(parts[4]); err == nil && w > 0 {
				weight = w
			}
		}
		if len(parts) >= 6 {
			if c, err := parseFloat(parts[5]); err == nil && c > 0 {
				confidence = c
			}
		}

		if subject == "" || summary == "" {
			continue
		}

		fact := MemoryFact{
			ID:          genHexID(),
			Domain:      domain,
			Subcategory: subcategory,
			Subject:     subject,
			Summary:     summary,
			Weight:      weight,
			Confidence:  confidence,
			Status:      "active",
			FactLayer:   "raw",
			Triggers:    extractTriggerWords(subject + " " + summary),
		}
		facts = append(facts, fact)
	}

	return facts
}

// parseFloat 简单的字符串→float64解析
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// extractTriggerWords 从文本中提取中文触发词
func extractTriggerWords(text string) []string {
	var words []string
	var current []rune
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			if len(current) >= 2 {
				words = append(words, string(current))
			}
			current = nil
		}
	}
	if len(current) >= 2 {
		words = append(words, string(current))
	}

	// 去重 + 限制数量
	seen := make(map[string]bool)
	var result []string
	for _, w := range words {
		if !seen[w] && len(result) < 8 {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}
