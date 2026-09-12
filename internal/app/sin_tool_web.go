package app

// ── 原罪工具：联网（web_search / web_fetch）──
//
// 实现不复制第二份：直接委托办公 builtin 的 web_search / web_fetch——
// SSRF 防护、域名 allow/deny 策略、[search] 引擎扇出与优先级、网络代理全部
// 沿用同一条链（boot 注入，一处生效）。原罪只提供故事语境的说明词与自己的
// 工具名，让模型知道「什么时候该查现实资料」。
//
// 空白导入 builtin：工具经 init() 自注册，显式依赖保证注册一定发生（不靠
// 传递导入的偶然性）。

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin"
)

func init() {
	registerSinTool(sinToolWebSearch, func(sinToolContext) sinTool {
		return sinToolWebDelegate{kind: sinToolWebSearch}
	})
	registerSinTool(sinToolWebFetch, func(sinToolContext) sinTool {
		return sinToolWebDelegate{kind: sinToolWebFetch}
	})
}

// sinToolWebDelegate 把联网工具转发给同名办公 builtin。
type sinToolWebDelegate struct{ kind string }

func (t sinToolWebDelegate) Name() string { return t.kind }

func (t sinToolWebDelegate) Description() string {
	if t.kind == sinToolWebFetch {
		return "抓取一个 URL 的正文全文（HTML 会转成可读文本）。搜索结果的摘要不够用时才用它补细节。"
	}
	return "联网搜索公开网页，返回标题/链接/摘要（实时抓取，不是训练记忆）。只用来把故事里的现实细节写准：" +
		"真实地名与年代风物、器物用法、职业与行业术语、真实事件的时间线。纯虚构设定、人物关系与情欲描写不需要搜索；" +
		"查到的内容化进描写里，不要在正文里写「我查了一下」，也不要罗列来源链接。"
}

func (t sinToolWebDelegate) Schema() json.RawMessage {
	if b, ok := tool.LookupBuiltin(t.kind); ok && b != nil {
		return b.Schema()
	}
	// 兜底：内置缺失时给一个空对象 schema，Execute 会如实报「不可用」。
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t sinToolWebDelegate) ReadOnly() bool { return true }

func (t sinToolWebDelegate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	b, ok := tool.LookupBuiltin(t.kind)
	if !ok || b == nil {
		return "", fmt.Errorf("联网工具不可用（内置 %s 未注册）", t.kind)
	}
	return b.Execute(ctx, args)
}
