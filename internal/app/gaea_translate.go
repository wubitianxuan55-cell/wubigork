package app

// 本地翻译（Herdsman 深挖 P3）：优先 Herdsman 翻译模型（Hunyuan-MT / Hy-MT，
// 模型目录 capability=translation），未安装时回退模型中心「常规办公」绑定模型。
// 文本翻译走 chat/completions（/v1/translations 是语音翻译，不适用）。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// LocalTranslateRequest 本地翻译请求。
type LocalTranslateRequest struct {
	Text       string `json:"text"`
	TargetLang string `json:"target_lang"`
	SourceLang string `json:"source_lang,omitempty"`
	Model      string `json:"model,omitempty"` // 显式指定 Herdsman 翻译模型
}

// LocalTranslateResult 本地翻译结果。
type LocalTranslateResult struct {
	Text         string `json:"text"`
	Model        string `json:"model"`
	Engine       string `json:"engine"`
	UsedFallback bool   `json:"used_fallback"`
	// Partial=true 表示 Text 只含部分成功段落的译文（伴随 error 返回，不静默丢弃）。
	Partial bool `json:"partial,omitempty"`
}

// translateSystemPrompt 只输出译文的翻译系统提示（对齐翻译模型的输出纪律）。
const translateSystemPrompt = "你是一个专业翻译引擎。只输出目标语言的译文，不输出解释、注释或原文。保留专有名词、数字、代码与原有格式。"

// isHerdsmanTranslationModel 判断模型名是否为翻译模型。
func isHerdsmanTranslationModel(id string) bool {
	l := strings.ToLower(id)
	return strings.Contains(l, "hunyuan-mt") || strings.Contains(l, "hy-mt")
}

// resolveTranslationTarget 优先 Herdsman 已装翻译模型；未找到返回 found=false
// （由调用方决定回退）。
func (c *core) resolveTranslationTarget(explicitModel string) (engine, model string, found bool, err error) {
	if c.engineMgr == nil {
		return "", "", false, errors.New("模型引擎管理器不可用")
	}
	e, ok := c.engineMgr.GetEngine("herdsman")
	if !ok || !e.Enabled || e.BaseURL == "" {
		return "", "", false, nil
	}
	if explicitModel != "" {
		return "herdsman", explicitModel, true, nil
	}
	// /v1/models 只返回已安装模型，遍历引擎缓存即可完成发现。
	for _, m := range e.Models {
		if isHerdsmanTranslationModel(m.ID) {
			return "herdsman", m.ID, true, nil
		}
	}
	return "", "", false, nil
}

// v1Join 把引擎 BaseURL 规整为 .../v1 前缀（兼容带/不带 /v1 两种写法）。
func v1Join(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base
	}
	return base + "/v1"
}

// callHerdsmanTranslate 直连 Herdsman /v1/chat/completions 调翻译模型。
func callHerdsmanTranslate(ctx context.Context, baseURL, model, system, prompt string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	body, err := json.Marshal(map[string]any{
		"model":       model,
		"temperature": 0.3,
		"max_tokens":  4096,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("构造翻译请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v1Join(baseURL)+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造翻译请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 Herdsman 翻译模型失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("读取翻译响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("翻译模型返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("解析翻译响应失败: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("翻译模型未返回内容")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// translateMaxSegmentRunes 单段翻译的最大字符数（按 rune 计）。超出时按段落
// 切段逐段翻译——长文档不再一次性塞进上下文（超时/超上下文的失败可见且不
// 丢失已完成段落的成果）。
const translateMaxSegmentRunes = 1500

// LocalTranslate 本地翻译：翻译模型优先，常规办公模型兜底。
// 可取消 ctx（继承 a.ctx，工具/测试可注入）；长文本分段翻译，单段失败重试
// 一次，仍失败时保留已成功段落并返回明确错误（不静默吞错）。
func (a *App) LocalTranslate(req LocalTranslateRequest) (LocalTranslateResult, error) {
	return a.localTranslate(a.backgroundCtx(), req)
}

// backgroundCtx 返回应用上下文（未启动/测试时为 Background，保证可取消语义）。
func (a *App) backgroundCtx() context.Context {
	if a != nil && a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// splitTranslateSegments 把待翻译文本切成不超过 translateMaxSegmentRunes 的段：
// 优先按换行段落边界分组；无换行的长文本按 rune 硬切。
func splitTranslateSegments(text string) []string {
	rs := []rune(text)
	if len(rs) <= translateMaxSegmentRunes {
		return []string{text}
	}
	if !strings.Contains(text, "\n") {
		var out []string
		for i := 0; i < len(rs); i += translateMaxSegmentRunes {
			end := i + translateMaxSegmentRunes
			if end > len(rs) {
				end = len(rs)
			}
			out = append(out, string(rs[i:end]))
		}
		return out
	}
	var segments []string
	var cur []rune
	for _, para := range strings.Split(text, "\n") {
		pr := []rune(para)
		if len(cur) > 0 && len(cur)+1+len(pr) > translateMaxSegmentRunes {
			segments = append(segments, string(cur))
			cur = nil
		}
		if len(cur) > 0 {
			cur = append(cur, '\n')
		}
		cur = append(cur, pr...)
	}
	if len(cur) > 0 {
		segments = append(segments, string(cur))
	}
	if len(segments) == 0 {
		segments = append(segments, text)
	}
	return segments
}

// translateOne 单段翻译：命中翻译模型走 Herdsman 直连；否则走办公模型兜底。
func (a *App) translateOne(ctx context.Context, engine, model string, found bool, prompt string) (string, error) {
	if found {
		return callHerdsmanTranslate(ctx, a.herdsmanBaseURL(), model, translateSystemPrompt, prompt)
	}
	if a.client == nil {
		return "", errors.New("未安装翻译模型（Hunyuan-MT / Hy-MT），且办公模型不可用；请先在模型库下载翻译模型")
	}
	return a.client.ChatSimpleStreamWithOptions(ctx, model, translateSystemPrompt, prompt, ai.ChatSimpleOptions{
		EngineID:       engine,
		Temperature:    0.3,
		MaxTokens:      4096,
		TimeoutMinutes: 3,
	})
}

// localTranslate 内部实现（ctx 可取消；测试可直接注入 ctx 验证取消/分段/重试）。
func (a *App) localTranslate(ctx context.Context, req LocalTranslateRequest) (LocalTranslateResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return LocalTranslateResult{}, errors.New("待翻译文本不能为空")
	}
	target := strings.TrimSpace(req.TargetLang)
	if target == "" {
		target = "zh"
	}

	engine, model, found, err := a.resolveTranslationTarget(strings.TrimSpace(req.Model))
	if err != nil {
		return LocalTranslateResult{}, err
	}
	usedFallback := !found
	if !found {
		// 未装翻译模型：回退常规办公模型（解析一次，供全部段落复用）。
		if a.client == nil {
			return LocalTranslateResult{}, errors.New("未安装翻译模型（Hunyuan-MT / Hy-MT），且办公模型不可用；请先在模型库下载翻译模型")
		}
		engine, model, err = a.resolveRoutineTarget("", "")
		if err != nil {
			return LocalTranslateResult{}, fmt.Errorf("未安装翻译模型，且回退办公模型失败: %w", err)
		}
	}

	buildPrompt := func(seg string) string {
		var p strings.Builder
		if src := strings.TrimSpace(req.SourceLang); src != "" {
			fmt.Fprintf(&p, "源语言：%s；", src)
		}
		fmt.Fprintf(&p, "目标语言：%s。\n%s", target, seg)
		return p.String()
	}

	segments := splitTranslateSegments(text)
	parts := make([]string, 0, len(segments))
	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			if len(parts) == 0 {
				return LocalTranslateResult{}, err
			}
			return LocalTranslateResult{
				Text: strings.Join(parts, "\n"), Model: model, Engine: engine,
				UsedFallback: usedFallback, Partial: true,
			}, fmt.Errorf("翻译已取消（已完成 %d/%d 段）: %w", len(parts), len(segments), err)
		}
		out, terr := a.translateOne(ctx, engine, model, found, buildPrompt(seg))
		if terr != nil && ctx.Err() == nil {
			// 单段失败重试一次（网络抖动/偶发超时），不静默吞错。
			out, terr = a.translateOne(ctx, engine, model, found, buildPrompt(seg))
		}
		if terr != nil {
			if len(parts) == 0 {
				return LocalTranslateResult{}, fmt.Errorf("第 %d/%d 段翻译失败: %w", i+1, len(segments), terr)
			}
			// 部分结果保留：已成功段落随错误一起返回。
			return LocalTranslateResult{
				Text: strings.Join(parts, "\n"), Model: model, Engine: engine,
				UsedFallback: usedFallback, Partial: true,
			}, fmt.Errorf("第 %d/%d 段翻译失败（已完成 %d 段）: %w", i+1, len(segments), len(parts), terr)
		}
		parts = append(parts, strings.TrimSpace(out))
	}
	return LocalTranslateResult{Text: strings.Join(parts, "\n"), Model: model, Engine: engine, UsedFallback: usedFallback}, nil
}

// herdsmanBaseURL 取 Herdsman 引擎 BaseURL（空时返回 ""）。
func (a *App) herdsmanBaseURL() string {
	if a.core == nil || a.engineMgr == nil {
		return ""
	}
	if e, ok := a.engineMgr.GetEngine("herdsman"); ok {
		return e.BaseURL
	}
	return ""
}

// translateTextTool 办公专业工具：本地翻译（翻译模型优先，未安装回退办公模型）。
type translateTextTool struct {
	a *App
}

func (t translateTextTool) Name() string { return "translate_text" }

func (t translateTextTool) Description() string {
	return "翻译文本到目标语言：优先 Herdsman 本地翻译模型（Hunyuan-MT / Hy-MT，免费），" +
		"未安装时自动回退「常规办公」绑定模型。适合文档段落、合同条款、邮件等文本翻译；" +
		"不消耗主模型 token（翻译模型路径）。"
}

func (t translateTextTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "text":{"type":"string","description":"待翻译文本"},
  "target_lang":{"type":"string","description":"目标语言代码，如 zh/en/ja，默认 zh"},
  "source_lang":{"type":"string","description":"可选：源语言代码，如 en/zh，缺省自动识别"},
  "model":{"type":"string","description":"可选：显式指定 Herdsman 翻译模型名（如 Hy-MT2:7B）"}
},
"required":["text"]
}`)
}

func (t translateTextTool) ReadOnly() bool { return true }

func (t translateTextTool) CompactDescription() string {
	return "本地翻译（翻译模型优先，未安装回退办公模型）"
}

func (t translateTextTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"},"target_lang":{"type":"string"}},"required":["text"]}`)
}

func (t translateTextTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.a == nil {
		return "", errors.New("translate_text: 应用实例不可用")
	}
	var p LocalTranslateRequest
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("translate_text: 参数无效: %w", err)
	}
	res, err := t.a.localTranslate(ctx, p)
	if err != nil {
		// 部分结果保留：错误信息里带上已翻译片段，调用方可见而非静默丢弃。
		if res.Text != "" {
			return "", fmt.Errorf("translate_text: %w（已翻译部分：%s）", err, res.Text)
		}
		return "", fmt.Errorf("translate_text: %w", err)
	}
	label := "翻译模型"
	if res.UsedFallback {
		label = "办公模型（翻译模型未安装）"
	}
	return fmt.Sprintf("%s（%s）", res.Text, label), nil
}

var _ tool.Tool = translateTextTool{}
