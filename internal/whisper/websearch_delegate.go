package whisper

// websearch_delegate.go — 联网检索委托缝（众神殿 2026-11-05 网络搜索刀1：
// 四入口收敛）。whisper 不 import 办公 builtin（依赖面收窄）：boot 由
// internal/app 注入统一检索核执行器，本包只留缝与诚实回执。
//
// 边界：模型提交的结构化 query（含 site: 操作符）原样透传不清洗——口语清洗
// 只归聊天 adapter（internal/app/chat_search_adapter.go），与办公 web_search
// 工具的「query 原样」契约一致。自研 Bing/DDG 解析（web_search.go）已删除，
// whisper.WebSearch 符号不存在 = 绕过 adapter 即编译红的源级守卫。

import (
	"context"
	"fmt"
	"strings"
)

// WebSearchToolExec 统一检索核委托缝：boot 由 app 注入办公 builtin web_search
// 执行器。nil（未接线）时 fail-loud 返回错误文本，不静默伪装成功。
var WebSearchToolExec func(ctx context.Context, query string) (string, error)

// whisperRunWebSearch agent loop 的 web_search 工具执行：统一核回执 + 诚实
// 失败文案（原「搜索「x」失败」分支因 whisper.WebSearch 几乎不返 error 而
// 恒死，委托后 Execute 真报错，该分支成为活路径）。
func whisperRunWebSearch(ctx context.Context, query string) string {
	if WebSearchToolExec == nil {
		return fmt.Sprintf("搜索「%s」失败：联网检索未接线（web_search 工具不可用）", query)
	}
	result, err := WebSearchToolExec(ctx, query)
	if err != nil {
		return fmt.Sprintf("搜索「%s」失败：%v", query, err)
	}
	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("搜索「%s」无结果", query)
	}
	return result
}
