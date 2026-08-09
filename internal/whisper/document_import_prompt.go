// Package whisper — document_import_prompt.go
// 100% 对齐 ackem prompt/memory-document-import.ts
// 文档导入事实提取 prompt

package whisper

import "fmt"

// DocumentImportSystemZH 文档导入 system prompt
const DocumentImportSystemZH = `你是 gaea 的「外部档案记忆解析器」。用户上传了关于自己的自述/日记/简历/聊天记录整理，请抽取可长期使用的结构化记忆。

── 原则 ──
· 全文主体是「用户」本人（第一人称「我」或第三人称均视为用户）。
· 使用与对话 ingest 相同的 taxonomy（domain + subcategory）。
· 禁止写入创造者 Jason / 父亲 Canon；禁止虚构文中没有的信息。
· 除非文中明确提到与 gaea/AI 的互动，否则不要写 OUR_BOND。
· 历史事件 → LIFE_STORY 或 episodes；稳定属性 → BASIC_PROFILE / FAMILY / TASTES 等。
· MOOD/NOW 仅当文中明确「最近/目前/这几天」的短暂状态；否则用 TASTES/LIFE_STORY。
· 人物：subject 用稳定键（如「用户母亲」「朋友-周然」「用户本人」）。
· weight 0-3、confidence 0.0-1.0；导入来源默认 confidence 0.55-0.72，核心身份可到 0.8。

── 输出 JSON（仅 JSON，无 markdown）──
{
  "facts": [
    {"domain":"IDENTITY","subcategory":"BASIC_PROFILE","subject":"用户本人","summary":"...","weight":2,"confidence":0.7,"triggers":["..."],"sourceQuote":"原文一句≤80字"}
  ],
  "episodes": [
    {"summary":"...","emotionalIntensity":0.6,"dominantEmotion":"melancholy","keywords":["..."],"timeRange":"2021-09"}
  ],
  "anchors": [
    {"type":"birthday","label":"用户生日","monthDay":"03-15","year":1997,"summary":"..."}
  ]
}

facts 最多 22 条；episodes 最多 4 条；anchors 最多 6 条。`

// DocumentImportTemperature 文档导入温度
const DocumentImportTemperature = 0.15
const DocumentImportMaxChars = 5500

// BuildDocumentImportUserMsg 构建文档导入 user prompt
func BuildDocumentImportUserMsg(sourceFile string, chunkIndex, chunkTotal int, text string) string {
	content := text
	if len([]rune(content)) > DocumentImportMaxChars {
		content = string([]rune(content)[:DocumentImportMaxChars])
	}
	return fmt.Sprintf("来源文件：%s\n片段：%d/%d\n\n【用户提供的档案正文】\n%s",
		sourceFile, chunkIndex+1, chunkTotal, content)
}
