package app

// 流式探针（D3-4 补测评缺口·流式中断恢复观察）：对 Herdsman 模型发起一次
// 真实 SSE 流式请求，观察首 token 延迟（TTFT）、分块间隔（卡顿/断流）与
// 完整度（是否收到 [DONE]）。用于模型中心「受控测评」里的快速连通性/稳定性
// 检查，不需要跑完整 benchmark。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// StreamProbeResult 一次流式探针的结果。
type StreamProbeResult struct {
	Model         string  `json:"model"`
	OK            bool    `json:"ok"`
	TTFTMS        int64   `json:"ttft_ms"`        // 首块延迟（首 token 前）
	Chunks        int     `json:"chunks"`         // 收到的数据分块数（不含 [DONE]）
	Tokens        int64   `json:"tokens"`         // usage.completion_tokens（有则填）
	DurationMS    int64   `json:"duration_ms"`    // 从请求发出到 [DONE]
	MaxGapMS      int64   `json:"max_gap_ms"`     // 相邻分块最大间隔（卡顿指示）
	AvgGapMS      int64   `json:"avg_gap_ms"`     // 平均分块间隔
	Completed     bool    `json:"completed"`      // 收到 [DONE] 正常结束
	Interrupted   bool    `json:"interrupted"`    // 提前断流（未收 [DONE] 但连接关闭）
	Error         string  `json:"error,omitempty"`
	ResponseStart string  `json:"response_start,omitempty"` // 回答开头（预览）
}

// GaeaBenchmarkStreamProbe 对指定 Herdsman 模型做一次流式探针（约 10-20s）。
// 模型名取模型库 installed 条目名；未运行/未安装时后端返回明确错误。
func (a *App) GaeaBenchmarkStreamProbe(model string) (StreamProbeResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return StreamProbeResult{}, errors.New("模型不能为空")
	}
	base := a.herdsmanBaseURL()
	if base == "" {
		return StreamProbeResult{}, errors.New("Herdsman 引擎未配置")
	}
	req := provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "请用 3 句话介绍你自己，然后停。保持流式输出稳定。"},
		},
		Temperature: 0.3,
		MaxTokens:   256,
	}
	body, err := json.Marshal(map[string]any{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
	})
	if err != nil {
		return StreamProbeResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		v1Join(base)+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return StreamProbeResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := herdsmanBenchHTTP.Do(httpReq)
	if err != nil {
		return StreamProbeResult{}, fmt.Errorf("发起流式请求失败（模型未运行？）: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return StreamProbeResult{}, fmt.Errorf("Herdsman 返回 HTTP %d: %s",
			resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	out := StreamProbeResult{Model: model}
	var firstChunkAt time.Time
	var prevChunkAt time.Time
	var maxGap int64
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		now := time.Now()
		if firstChunkAt.IsZero() {
			firstChunkAt = now
			out.TTFTMS = now.Sub(start).Milliseconds()
		} else if !prevChunkAt.IsZero() {
			if gap := now.Sub(prevChunkAt).Milliseconds(); gap > maxGap {
				maxGap = gap
			}
		}
		prevChunkAt = now

		if data == "[DONE]" {
			out.Completed = true
			break
		}
		out.Chunks++
		var ev struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				CompletionTokens int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &ev) == nil {
			if len(ev.Choices) > 0 && out.ResponseStart == "" {
				if c := strings.TrimSpace(ev.Choices[0].Delta.Content); c != "" {
					r := []rune(c)
					if len(r) > 80 {
						r = r[:80]
					}
					out.ResponseStart = string(r)
				}
			}
			if ev.Usage != nil {
				out.Tokens = ev.Usage.CompletionTokens
			}
		}
	}
	out.DurationMS = time.Since(start).Milliseconds()
	if out.Chunks > 0 && !prevChunkAt.IsZero() && !firstChunkAt.IsZero() {
		out.AvgGapMS = prevChunkAt.Sub(firstChunkAt).Milliseconds() / int64(out.Chunks)
	}
	out.MaxGapMS = maxGap
	if err := sc.Err(); err != nil && !out.Completed {
		out.Interrupted = true
		out.Error = err.Error()
	}
	if !out.Completed && !out.Interrupted && out.Chunks == 0 {
		out.Error = "流式响应为空"
	}
	out.OK = out.Completed && out.Chunks > 0
	return out, nil
}
