package browser

// manager_runjs.go — browser_run_js 一次执行范式（ego-lite 蒸馏）：用户代码
// 与 $ 帮手（$read/$snapshot/$click/$fill/$wait/$waitFor/$log）拼进 jsRunJs
// 模板，单次 Runtime.evaluate 往返完成多步页面操作。返回值经 envelope 透传；
// value 截断在 Go 侧做（rune 安全），超长给预览与 value_truncated 标记。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RunJSResult 一次执行的结果：Value 为用户代码返回值（任意 JSON，CDP
// returnByValue 序列化，函数/Symbol 等不可序列化值由页面侧 catch）；超长时
// 降级为文本预览并置 ValueTruncated（rune 安全，不输出半截非法 JSON）。
type RunJSResult struct {
	Title          string   `json:"title"`
	URL            string   `json:"url"`
	Value          any      `json:"value"` // json.RawMessage 原样 或 string 预览
	ValueTruncated bool     `json:"value_truncated,omitempty"`
	Logs           []string `json:"logs"`
}

// RunJS 在 active tab 主文档执行用户代码（一次执行范式）。maxChars 限制返回
// value 的序列化字节数（默认/上限同 Read）。用户代码内调过 $snapshot 时，
// 页面 __gaeaEpoch 已递增且 refs 已登记——回传 epoch 直接登记 m.epoch，后续
// browser_click(ref)/browser_type(ref) 的 guardEpoch 天然通过（ref 生态打通）。
// 页面已导航（epoch 变量清零）时登记 0，guardEpoch fail-closed 拒绝旧 ref。
func (m *Manager) RunJS(ctx context.Context, code string, maxChars int) (RunJSResult, error) {
	if strings.TrimSpace(code) == "" {
		return RunJSResult{}, fmt.Errorf("%w: code 必填", ErrInvalidInput)
	}
	if err := m.Ensure(ctx); err != nil {
		return RunJSResult{}, err
	}
	if maxChars <= 0 {
		maxChars = defaultReadChars
	}
	if maxChars > maxReadChars {
		maxChars = maxReadChars
	}
	var res struct {
		okField
		Title string          `json:"title"`
		URL   string          `json:"url"`
		Epoch int64           `json:"epoch"`
		Logs  []string        `json:"logs"`
		Value json.RawMessage `json:"value"`
	}
	if err := m.evaluate(ctx, fmt.Sprintf(jsRunJs, jsString(code)), &res); err != nil {
		return RunJSResult{}, err
	}
	if !res.OK {
		return RunJSResult{}, jsErr(res.Error)
	}
	m.mu.Lock()
	m.epoch = res.Epoch
	m.mu.Unlock()
	if res.Logs == nil {
		res.Logs = []string{}
	}
	value, truncated := truncateRunJSValue(res.Value, maxChars)
	return RunJSResult{Title: res.Title, URL: res.URL, Value: value, ValueTruncated: truncated, Logs: res.Logs}, nil
}

// truncateRunJSValue 把 value 收进 maxChars：超长时降级为文本预览（rune 安全，
// 不输出半截非法 JSON），truncated=true 供 envelope 标注。
func truncateRunJSValue(v json.RawMessage, maxChars int) (out any, truncated bool) {
	if len(v) == 0 {
		return nil, false
	}
	if len(v) <= maxChars {
		return v, false
	}
	r := []rune(string(v))
	if len(r) > maxChars {
		r = r[:maxChars]
	}
	return string(r), true
}
