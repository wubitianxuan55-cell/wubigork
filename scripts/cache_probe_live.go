//go:build ignore

// cache_probe_live.go — 真 DeepSeek API 缓存命中率实测探针（2026-11-21）。
//
// 用 gaea 的真实序列化形态（ai.ChatRequest json.Marshal=线上字节同源）向
// https://api.deepseek.com 发 3 轮对话：固定 system+tools 前缀，逐轮追加
// user/assistant 消息，打印每轮 DeepSeek 返回的 prompt_cache_hit/miss_tokens。
// key 经 config.Load()+secure.DecryptString 取（与 app 同一读取链），不落盘。
//
// 运行：go run scripts/cache_probe_live.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin" // 真实 builtin 工具 schema
)

func main() {
	cfg := config.Load()
	key, _ := secure.DecryptString(cfg.DeepseekAPIKey)
	if key == "" {
		fmt.Println("DeepSeek key 为空（config.Load 未取到）——探针无法真连")
		os.Exit(1)
	}

	// 真实 builtin 工具 schema（注册表排序导出）。
	var tools []ai.ChatToolSchema
	for _, tl := range tool.Builtins() {
		tools = append(tools, ai.ChatToolSchema{
			Type: "function",
			Function: ai.ChatToolFunctionSpec{
				Name:        tl.Name(),
				Description: tl.Description(),
				Parameters:  json.RawMessage(tl.Schema()),
			},
		})
	}
	sys := strings.Repeat("你是办公助手。遵循工作区规范：交付物经 office_generate/docx_apply 结构化入口；"+
		"数字核对用 md_validate；长文档按标题整节读取。", 60)

	mkReq := func(msgs []ai.ChatMessage) *ai.ChatRequest {
		return &ai.ChatRequest{
			Model:    "deepseek-v4-pro",
			Messages: msgs,
			Tools:    tools,
			Stream:   false,
		}
	}

	msgs := []ai.ChatMessage{{Role: "system", Content: sys}}
	cumHit, cumTotal := int64(0), int64(0)
	client := &http.Client{Timeout: 120 * time.Second}

	type turn struct{ q, a string }
	turns := []turn{
		{"第一问：核对报价表口径，列出涉及的工具与步骤。", "报价表口径核对需 cost_search 检索单价、md_validate 校验编号，步骤已列。"},
		{"第二问：把上述步骤整理成实施方案大纲。", "实施方案大纲：一、口径核对；二、清单编制；三、复核与交付。"},
		{"第三问：按大纲估算工期与风险项。", "工期 3 天；风险项：单价快照漂移、编号撞号，均已纳入复核。"},
	}

	fmt.Printf("=== 真 DeepSeek API 缓存命中率实测（3 轮，工具面=%d 个真实 builtin schema）===\n", len(tools))
	for i, t := range turns {
		msgs = append(msgs, ai.ChatMessage{Role: "user", Content: t.q})
		body, _ := json.Marshal(mkReq(msgs))
		req, _ := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("HTTP:", err)
			os.Exit(1)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			fmt.Printf("HTTP %d: %s\n", resp.StatusCode, string(raw[:min(len(raw), 400)]))
			os.Exit(1)
		}
		var out struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				HitTokens        int64 `json:"prompt_cache_hit_tokens"`
				MissTokens       int64 `json:"prompt_cache_miss_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(raw, &out) != nil {
			fmt.Println("响应解析失败:", string(raw[:min(len(raw), 400)]))
			os.Exit(1)
		}
		u := out.Usage
		rate := 0.0
		if u.PromptTokens > 0 {
			rate = float64(u.HitTokens) / float64(u.PromptTokens) * 100
		}
		cumHit += u.HitTokens
		cumTotal += u.PromptTokens
		fmt.Printf("轮 %d：prompt=%d hit=%d miss=%d 命中率=%.1f%%（累计 %.1f%%）\n",
			i+1, u.PromptTokens, u.HitTokens, u.MissTokens, rate,
			func() float64 { if cumTotal > 0 { return float64(cumHit) / float64(cumTotal) * 100 }; return 0 }())
		msgs = append(msgs,
			ai.ChatMessage{Role: "assistant", Content: t.a},
		)
	}
	fmt.Println("=== 实测口径：固定 system+真实 builtin 工具面，逐轮纯追加（无压缩/无重写）===")
}
