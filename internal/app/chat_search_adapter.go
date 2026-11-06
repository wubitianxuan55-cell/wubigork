package app

// chat_search_adapter.go — 聊天板块联网检索适配器（众神殿 2026-11-05 网络搜索
// 刀1：统一检索核 + 三层失败契约 + 清洗边界）。聊天四入口（plain 注入 / 人格
// 注入 / WhisperWebSearch 绑定 / whisper agent loop 委托缝）全部经统一检索核
// （办公 builtin websearch），whisper 自研 Bing/DDG 解析已删除——
// whisper.WebSearch 符号不存在，任何绕过 adapter 的旧直连都会编译红。
//
// 三层失败契约（会话终稿；刀2 扩 DEGRADED）：
//  1. 返回值状态枚举（chatSearchOutcome.State=ok/degraded/empty/all_failed，
//     策略过滤后空≠ALL_FAILED 的拆分在统一核 SearchAttempt）；
//  2. 注入头按状态选文案（injectChatSearchResult，不再无条件「实时搜索结果」）；
//  3. plain 系统提示词删「不要说自己无法联网搜索」（chat_service.go，仅 plain，
//     人格路径没有此句勿加）。
//
// 清洗边界（会话终稿）：口语清洗只归本 adapter；办公/agent loop 的模型结构化
// query（含 site: 操作符）原样不改写。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// chatSearchTimeout 聊天路径硬超时（会话建议值 8s；办公路径沿用统一核总截止
// 20s，两路 ctx 取更短者生效）。
const chatSearchTimeout = 8 * time.Second

// chatSearchTopK 聊天注入条数（与旧 whisper maxResults 一致）。
const chatSearchTopK = 5

// chatSearchOutcome 三层契约第一层：状态 + 注入文本原料。
type chatSearchOutcome struct {
	State          builtin.SearchState
	DegradedReason string // DEGRADED 时的原因（primary-timeout/coverage-* 等）
	Query          string // 清洗后的查询词（实际发给引擎的）
	RawQuery       string // 用户原始问句
	Text           string // OK/DEGRADED：编号结果列表（title/url/snippet/source）；其余：空
	Engines        string // 逐引擎诊断摘要（一行，EMPTY/ALL_FAILED/DEGRADED 时进注入文案）
}

// chatSearchRun 统一检索核 seam：测试可替换避免真实网络请求（刀2 起返回
// *builtin.SearchOutcome，State 由核内主源保留窗+覆盖门归出）。
var chatSearchRun = builtin.RunSearch

// chatWebSearch 聊天检索 seam（签名=状态契约）：生产走 runChatWebSearch，
// 测试可替换（旧签名 func(string)(string,error) 随三层契约一并退役）。
var chatWebSearch = runChatWebSearch

// runChatWebSearch 清洗口语查询 → 统一检索核（8s 硬超时）→ 状态归集。
func runChatWebSearch(rawQuery string) (chatSearchOutcome, error) {
	query := cleanChatSearchQuery(rawQuery)
	ctx, cancel := context.WithTimeout(context.Background(), chatSearchTimeout)
	defer cancel()
	outcome, err := chatSearchRun(ctx, query, chatSearchTopK)
	if outcome == nil {
		// 统一核自身故障（无产出）：不可用态，如实上报
		reason := ""
		if err != nil {
			reason = err.Error()
		}
		return chatSearchOutcome{State: builtin.SearchStateAllFailed, Query: query, RawQuery: rawQuery, Engines: reason}, nil
	}
	engines := summarizeSearchAttempts(outcome.Attempts)
	if outcome.DegradedReason != "" {
		engines = outcome.DegradedReason + "; " + engines
	}
	if engines == "" && err != nil {
		engines = err.Error()
	}
	out := chatSearchOutcome{
		State:          outcome.State,
		DegradedReason: outcome.DegradedReason,
		Query:          query,
		RawQuery:       rawQuery,
		Engines:        engines,
	}
	if outcome.State == builtin.SearchStateOK || outcome.State == builtin.SearchStateDegraded {
		out.Text = formatChatSearchResults(outcome.Results)
	}
	return out, nil
}

// injectChatSearchResult 三层契约第二层：注入头按状态选文案。EMPTY 与
// ALL_FAILED 分开表述，DEGRADED 的降级标注必须压过「优先依据搜索结果」
// （会话裁决：免费引擎兜底结果不得冒充已核验事实）。
// 众神殿刀3：增广文本与用户原文拆为双通道（chatSearchAppendix），本函数保留
// 原口径供 plain 路径与既有测试使用。
func injectChatSearchResult(message string, out chatSearchOutcome) string {
	return message + chatSearchAppendix(out)
}

// chatSearchAppendix 只取搜索增广块（含四态注入头，不带用户原文）——双通道
// 切片把它拼进本轮模型输入（llmUserMsg），原文走 raw 通道进状态机/记忆。
// searchDateAnchor 检索时间锚（B5 刀二，2026-11-06 实机病例）：plain 板块
// 「明天成都的天气怎么样」的回答写成「明天（2025年9月17日）」——搜索带回
// 的过期缓存页自带旧日期，模型无时间参照就把结果页日期当「今天/明天」。
// 注入块头部固定给出检索当日日期与星期，并明令历史日期不作相对日依据。
func searchDateAnchor(now time.Time) string {
	week := []string{"日", "一", "二", "三", "四", "五", "六"}[int(now.Weekday())]
	return fmt.Sprintf("检索时间：%d年%d月%d日 星期%s。结果中的页面日期早于检索时间的均为历史/缓存信息，不得据此回答「今天/明天/后天」；相对日期一律以检索时间为基准推算。", now.Year(), int(now.Month()), now.Day(), week)
}

func chatSearchAppendix(out chatSearchOutcome) string {
	anchor := searchDateAnchor(time.Now())
	switch out.State {
	case builtin.SearchStateOK:
		return fmt.Sprintf("\n\n[以下是关于此问题的实时搜索结果，请参考这些信息回答。%s]\n%s", anchor, out.Text)
	case builtin.SearchStateDegraded:
		return fmt.Sprintf("\n\n[联网搜索已降级（%s）：以下为兜底引擎结果，可信度有限；请谨慎引用，无法确认时如实说明未获取到可靠实时信息。%s]\n%s",
			degradedReasonLabel(out.DegradedReason), anchor, out.Text)
	case builtin.SearchStateEmpty:
		return fmt.Sprintf("\n\n[联网搜索已完成：各引擎均未返回与「%s」相关的结果%s。请基于已有知识回答，并如实说明未获取到实时信息]",
			out.Query, searchDiagSuffix(out.Engines))
	default: // all_failed
		return fmt.Sprintf("\n\n[联网搜索暂不可用%s。请基于已有知识回答，并如实说明未获取到实时信息]",
			searchDiagSuffix(out.Engines))
	}
}

// degradedReasonLabel 降级原因的用户/模型可读文案。
func degradedReasonLabel(reason string) string {
	switch reason {
	case "primary-timeout":
		return "主源保留窗超时"
	case "primary-error":
		return "主源不可用"
	case "coverage-generic-query":
		return "查询无可核验实体"
	case "coverage-miss":
		return "结果与查询相关性存疑"
	default:
		if reason == "" {
			return "未知原因"
		}
		return reason
	}
}

// searchDiagSuffix 诊断摘要拼进占位文案的括号后缀（空诊断不加括号）。
func searchDiagSuffix(engines string) string {
	if engines == "" {
		return ""
	}
	return "（" + engines + "）"
}

// summarizeSearchAttempts 逐引擎诊断压成一行（可见但控制 token 预算，
// 单条原因截断 40 rune）。
func summarizeSearchAttempts(attempts []builtin.SearchAttempt) string {
	if len(attempts) == 0 {
		return ""
	}
	parts := make([]string, 0, len(attempts))
	for _, a := range attempts {
		reason := ""
		switch a.Outcome {
		case "ok":
			parts = append(parts, fmt.Sprintf("%s:ok(%dms)", a.Name, a.ElapsedMS))
			continue
		case "empty":
			reason = "无结果"
		case "filtered-empty":
			reason = "策略过滤后无可用结果"
		default:
			reason = truncateRunes(a.Err, 40)
			if reason == "" {
				reason = "失败"
			}
		}
		parts = append(parts, fmt.Sprintf("%s:%s(%dms)", a.Name, reason, a.ElapsedMS))
	}
	return strings.Join(parts, "; ")
}

// formatChatSearchResults OK 态的编号结果列表。字段与办公契约一致
// （title/url/snippet/source），source 并入标题行供模型引用来源。
func formatChatSearchResults(results []builtin.SearchResult) string {
	var b strings.Builder
	for i, r := range results {
		src := ""
		if r.Source != "" {
			src = "（来源：" + r.Source + "）"
		}
		b.WriteString(fmt.Sprintf("%d. %s%s\n   %s\n   %s\n", i+1, r.Title, src, r.URL, r.Snippet))
	}
	return strings.TrimSpace(b.String())
}

// cleanChatSearchQuery 从口语化提问中提取适合搜索引擎的关键词（自 whisper
// cleanSearchQuery 下沉，众神殿刀1 清洗边界）：去掉触发前缀（帮我搜索/查一下…）、
// 疑问词（什么是/如何/为什么…）、结尾语气词（怎样/吗/呢…）与礼貌用语（请/麻烦/
// 帮我），再归一化空白与标点。清洗后为空时回退原文。
//
// 刀1 修正（会话终稿验收）：不再剥「在哪里/在哪儿/多少钱」——这些是信息词不是
// 语气词，剥掉会把「成都到上海在哪里买票」削成「成都到上海买票」；口语后缀
// （怎么样/吗/呢）照剥。
func cleanChatSearchQuery(msg string) string {
	s := strings.TrimSpace(msg)

	// B5 刀（2026-11-06 实机病例）：开头关联词/口语连接词剥离——「所以，
	// 明天的天气怎么样」此前被搜成「所以，明天的天气」，领头连接词主导引擎
	// 命中（返回罗大佑歌曲《所以》词条而非天气）。剥词连带剥其后的逗号分顿；
	// 只收多字连接词（单字「那」会误伤「那个/那达慕」类词头，不收）。
	for _, p := range []string{"所以", "那么", "然后", "还有", "另外", "而且", "但是", "可是", "不过", "就是说", "对了", "话说"} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimLeft(strings.TrimSpace(s[len(p):]), "，,、 ")
			break
		}
	}

	// 去掉开头的“帮/请/查”类触发前缀
	for _, p := range []string{
		"帮我搜索一下", "帮我搜一下", "帮我查一下", "请帮我查一下", "搜索一下",
		"帮我搜索", "帮我查", "帮我搜", "上网查一下", "上网查",
		"搜一下", "查一下", "查一查", "查查", "搜索", "查询",
	} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}

	// 去掉开头的疑问词（什么是X → X）
	for _, p := range []string{"什么是", "啥是", "怎么样", "怎样", "如何", "为什么", "怎么"} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}

	// 去掉结尾的疑问/口语后缀（X怎么样 → X）；「在哪里/在哪儿/多少钱」是
	// 信息词，保留不剥（刀1 验收）。
	for {
		if strings.HasSuffix(s, "在哪里") || strings.HasSuffix(s, "在哪儿") || strings.HasSuffix(s, "多少钱") {
			break
		}
		stripped := false
		for _, suf := range []string{
			"怎么样了", "怎么样呢", "怎么样", "怎样", "怎么办", "怎么弄", "如何",
			"是什么", "是啥", "什么时候", "有没有", "有哪些", "多少",
			"吗", "呢", "啊", "呀", "吧",
		} {
			for strings.HasSuffix(s, suf) {
				s = strings.TrimSpace(strings.TrimSuffix(s, suf))
				stripped = true
			}
		}
		if !stripped {
			break
		}
	}

	// 去掉常见礼貌词/冗余词（注意不能用单字“请”，会误删“申请/请假”等）
	for _, w := range []string{"麻烦", "帮我", "我想知道", "告诉我", "介绍一下"} {
		s = strings.ReplaceAll(s, w, "")
	}

	// 去掉开头的第一人称意愿前缀（多轮续接刀，DSH r4 指出的缺口）：
	// 「我要天气」会原样去搜。只剥前缀不全文替换，避免误删句中「我要去…」
	// 的查询内容。
	for _, p := range []string{"我要问", "我想问", "我要查", "我想查", "我想知道", "我想要", "我要", "我想"} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}

	// 归一化空白并去掉首尾标点
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, " ，。？！、,.?!:：;；\"'“”‘’()（）")
	if s == "" {
		return msg
	}
	return s
}
