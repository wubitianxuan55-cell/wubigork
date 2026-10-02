package cost

// 匹配索引与价格解析（审计 GA6-05 收敛）：pricefeed.matchRows 与
// costimport.MatchRows 原各持一份同型「既有条目三索引 + 标题/SlugName 兜底」
// 匹配，价格解析也各有方言副本（costimport.parsePrice / pricefeed.parsePrice）。
// 本文件收编唯一实现，包间方言经参数复现——行为逐字段不变（含 GA6-09 的
// 读失败留痕与日志文案逐字）。
//
// 口径注记：costref 的条目/询价索引是另一方言（标题键=costinquiry.MatchTitle，
// 同键取首而非后写覆盖；带码未命中不回退），不在本类型收敛范围；本文件三
// 索引键口径与收敛前 pricefeed/costimport 逐字段一致。

import (
	"log/slog"
	"strconv"
	"strings"
)

// MatchIndex 既有成本条目的 编码/标题/名称 三索引（价源比对与文件导入共用的
// 匹配基建）。键口径（与收敛前两处方言一致）：
//   - 编码键 = NormalizeCode（全角转半角、去空白、大写）；
//   - 标题键 = lower+trim（「HP300」/「hp300 」同键）；
//   - 名称键 = trim（Name 本身是 SlugName 确定性键）。
//
// 同键多条目后写覆盖先写（收敛前两处即 `m[k]=s` 直写口径；costref 的
// 「同键取首」不经本类型）。
type MatchIndex struct {
	byCode  map[string]Summary
	byTitle map[string]Summary
	byName  map[string]Summary
}

// NewMatchIndex 从条目摘要构建三索引（纯函数：同输入同输出）。
func NewMatchIndex(entries []Summary) *MatchIndex {
	ix := &MatchIndex{
		byCode:  map[string]Summary{},
		byTitle: map[string]Summary{},
		byName:  map[string]Summary{},
	}
	for _, s := range entries {
		if c := NormalizeCode(s.Code); c != "" {
			ix.byCode[c] = s
		}
		if t := strings.ToLower(strings.TrimSpace(s.Title)); t != "" {
			ix.byTitle[t] = s
		}
		if n := strings.TrimSpace(s.Name); n != "" {
			ix.byName[n] = s
		}
	}
	return ix
}

// LoadMatchIndex 读成本库建索引：store 不可用（nil/未开库）=空索引，调用方
// 全部按新增处理；读失败按已读到部分匹配，但必须留痕（审计 GA6-09）——否则
// 会把「读不到」当成「库里没有同名条目」而全部标成新增。logPrefix 标注调用方
// （"pricefeed"/"costimport"），日志文案与收敛前逐字一致。
func LoadMatchIndex(store *Store, logPrefix string) *MatchIndex {
	if store == nil || !store.Available() {
		return NewMatchIndex(nil)
	}
	existing, err := store.List()
	if err != nil {
		slog.Warn(logPrefix+": 成本库读取失败，匹配按已读到部分", "error", err)
	}
	return NewMatchIndex(existing)
}

// MatchByCode 带码行匹配：编码归一化后非空且命中返回条目；未命中返回 false。
// 带码未命中调用方不得回退标题——同标题不同编码=不同子目，宁新增勿误配。
func (ix *MatchIndex) MatchByCode(code string) (Summary, bool) {
	c := NormalizeCode(code)
	if c == "" {
		return Summary{}, false
	}
	s, ok := ix.byCode[c]
	return s, ok
}

// MatchByTitle 无码行匹配：标题归一化（lower+trim）精确优先，名称（标题的
// SlugName 确定性键）兜底，依次回退。
func (ix *MatchIndex) MatchByTitle(title string) (Summary, bool) {
	if t := strings.ToLower(strings.TrimSpace(title)); t != "" {
		if s, ok := ix.byTitle[t]; ok {
			return s, true
		}
	}
	if s, ok := ix.byName[SlugName(title)]; ok {
		return s, true
	}
	return Summary{}, false
}

// PriceOpts 价格解析方言（审计 GA6-05：costimport 与 pricefeed 两份 parsePrice
// 的差异参数化，实现唯一、口径逐字段复现）。
type PriceOpts struct {
	// RequireDigit 要求原文含 ASCII 数字字符（pricefeed 方言）。无此闸时
	// "Inf"/"NaN" 这类 ParseFloat 特殊字面量会被当成价格返回（costimport
	// 方言的历史口径，本刀只钉差异不改行为）。
	RequireDigit bool
	// Round2 结果四舍五入两位（pricefeed 方言；发布价精度）。
	Round2 bool
}

// ParsePrice 归一化价格：去掉 ¥/￥/元/千分位（半/全角逗号）/空白（含全角
// 空格与 nbsp），正数才有效。支持 "3,200.00"、"3200 元"、"￥3181.00"。
// costimport 方言（报价单/测算表：不取整、无数字预检）。
func ParsePrice(s string) (float64, bool) {
	return ParsePriceWithOpts(s, PriceOpts{})
}

// ParsePriceWithOpts 按方言解析价格（唯一实现）。
func ParsePriceWithOpts(s string, opts PriceOpts) (float64, bool) {
	if opts.RequireDigit && !strings.ContainsAny(s, "0123456789") {
		return 0, false
	}
	clean := strings.NewReplacer(",", "", "，", "", "¥", "", "￥", "", "元", "", " ", "", "\u00a0", "").Replace(strings.TrimSpace(s))
	if clean == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	if opts.Round2 {
		v = Round2(v)
	}
	return v, true
}

// Round2 四舍五入两位（价格解析与价差/环比共用；公式与收敛前 pricefeed
// 的 round2 逐字一致）。
func Round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
