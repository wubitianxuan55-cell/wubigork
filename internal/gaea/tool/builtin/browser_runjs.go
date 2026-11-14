package builtin

// browser_runjs.go — browser_run_js 一次执行范式（ego-lite 蒸馏，2026-11-14）：
// agent 写一段 JS 内联执行，单次往返完成多步「看→点→填→等」，替代 11 个
// browser_* 工具的多轮工具调用循环（省 token 省延迟）。页面侧注入 $ 帮手
// （$read/$snapshot/$click/$fill/$wait/$waitFor/$log），$snapshot 复用既有
// data-gaea-ref 标记与 __gaeaEpoch 代数——run_js 里拿到的 ref 可直接喂
// browser_click/browser_type，与既有 ref 生态打通。现有 11 工具原样保留，
// 本工具为纯增量。

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gaea/gaea/internal/gaea/browser"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(browserRunJS{}) }

type browserRunJS struct{}

func (browserRunJS) Name() string { return "browser_run_js" }

func (browserRunJS) Description() string {
	return "在受控浏览器当前页面一次执行一段 JS，单次往返完成多步操作（推荐替代连续多轮 browser_click/browser_type 调用，省时省 token）。页面注入帮手函数：$read(selector?)读文本、$snapshot(max?)列可交互元素（返回 [{ref,tag,text,path}]）、$click(ref或selector)点击、$fill(ref或selector,text,submit?)输入（React 兼容）、$wait(ms)等待、$waitFor(selector,timeoutMs?)等元素出现、$log(x)收集调试信息。代码作为函数体执行（可 await），返回值 JSON 透传（max_chars 截断）。注意：$snapshot 返回的 ref 可直接给后续 browser_click/browser_type 用；代码内不要做页面跳转（location.href 会丢返回值，导航用 browser_navigate）。"
}

func (browserRunJS) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "code":{"type":"string","description":"要执行的 JS 函数体（可 async/await）。可用帮手：$read(selector?) / $snapshot(max?) / $click(ref或selector) / $fill(ref或selector,text,submit?) / $wait(ms) / $waitFor(selector,timeoutMs?) / $log(x)。return 的值 JSON 透传"},
  "max_chars":{"type":"integer","description":"返回 value 最大字符数（默认 6000，超长给文本预览并标记 value_truncated）","minimum":1}
},
"required":["code"]
}`)
}

func (browserRunJS) ReadOnly() bool                 { return false }
func (browserRunJS) SpaceTag() string               { return spaces.SpaceWork }
func (browserRunJS) CompactDescription() string     { return compactDesc["browser_run_js"] }
func (browserRunJS) CompactSchema() json.RawMessage { return compactSchema["browser_run_js"] }

func (browserRunJS) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Code     string `json:"code"`
		MaxChars int    `json:"max_chars"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
	}
	if strings.TrimSpace(p.Code) == "" {
		return tool.WrapError(tool.CodeValidationError, "code 必填（一段 JS 函数体，可用 $read/$snapshot/$click/$fill/$wait/$waitFor/$log 帮手）", nil), nil
	}
	res, err := browser.Default().RunJS(ctx, p.Code, p.MaxChars)
	if err != nil {
		return browserFail(err, nil)
	}
	data := map[string]any{
		"url":   res.URL,
		"title": res.Title,
		"value": res.Value,
		"logs":  res.Logs,
	}
	if res.ValueTruncated {
		data["value_truncated"] = true
	}
	return tool.WrapResult("ok", data), nil
}
