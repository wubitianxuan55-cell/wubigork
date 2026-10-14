package app

// ── 小说写作档防重复采样（阶段八 8.7）────────────────────────
//
// 前置核实（2026-10-15 真机+源码）：
//   - herdsman（llama.cpp server 系）：DRY 全套键 + xtc_probability **真机验证
//     接受**（HTTP 200 正常生成）；
//   - strata：引擎内核 CUDA 采样链含 DRY+XTC（sampler.cu），但服务层每请求只
//     透传 temperature/top_p/top_k/seed ——发了无效，不注入（上游加键后接）；
//   - 云端 OpenAI 兼容服务对未知字段不保证容忍——不注入。
//   ⇒ 注入条件 = 本地引擎 且 Type != strata。
//
// 默认档 = llama.cpp DRY 缺省语义（0.8/1.75/2/4096），XTC 缺省关（激进）。
// 可配面：config 键 novel_dry_disabled（一键关）/ novel_sampler_json（全量
// 覆盖档）；生成 done 事件回显生效状态（作者可见）。四步自检段（SelfCheck）
// 是采样的提示词侧配合：模型收尾前自查句式/实体/伏笔/时间线。

import (
	"encoding/json"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// novelSampler 默认档（llama.cpp DRY 缺省语义）。
type novelSampler struct {
	DryMultiplier    float64 `json:"dry_multiplier"`     // 0.8
	DryBase          float64 `json:"dry_base"`           // 1.75
	DryAllowedLength int     `json:"dry_allowed_length"` // 2
	DryPenaltyLastN  int     `json:"dry_penalty_last_n"` // 4096
	XtcProbability   float64 `json:"xtc_probability"`    // 0=关
	XtcThreshold     float64 `json:"xtc_threshold"`      // 0.1
	SelfCheck        *bool   `json:"self_check"`         // nil=开
}

// defaultNovelSampler 内置默认档。
func defaultNovelSampler() novelSampler {
	return novelSampler{
		DryMultiplier:    0.8,
		DryBase:          1.75,
		DryAllowedLength: 2,
		DryPenaltyLastN:  4096,
		XtcProbability:   0,
		XtcThreshold:     0.1,
	}
}

// resolveNovelSampler 解析生效档：config 覆盖 JSON 优先（字段级合并到默认档上
// ——覆盖档只写想改的字段），未配置回落内置默认档。解析失败诚实回落默认档并
// 返回 warn（不静默吞配置错误）。
func resolveNovelSampler(cfg *config.Config) (novelSampler, string) {
	s := defaultNovelSampler()
	if cfg == nil || strings.TrimSpace(cfg.NovelSamplerJSON) == "" {
		return s, ""
	}
	var over novelSampler
	if err := json.Unmarshal([]byte(cfg.NovelSamplerJSON), &over); err != nil {
		return s, "novel_sampler_json 解析失败，已回落默认档: " + err.Error()
	}
	if over.DryMultiplier > 0 {
		s.DryMultiplier = over.DryMultiplier
	}
	if over.DryBase > 0 {
		s.DryBase = over.DryBase
	}
	if over.DryAllowedLength > 0 {
		s.DryAllowedLength = over.DryAllowedLength
	}
	if over.DryPenaltyLastN != 0 {
		s.DryPenaltyLastN = over.DryPenaltyLastN
	}
	if over.XtcProbability > 0 {
		s.XtcProbability = over.XtcProbability
		s.XtcThreshold = over.XtcThreshold
	}
	if over.SelfCheck != nil {
		s.SelfCheck = over.SelfCheck
	}
	return s, ""
}

// samplerAppliesToEngine 注入条件：本地引擎且非 strata（前置核实口径，见文件头）。
// engineMgr 不可用/引擎未登记（测试裸 App）时按「不支持」处理——宁缺勿误，
// 不对未知引擎发非标键。
func samplerAppliesToEngine(engineMgr *modelengine.Manager, engineID string) bool {
	if engineMgr == nil {
		return false
	}
	eng, ok := engineMgr.GetEngine(engineID)
	if !ok {
		return false
	}
	return eng.Type.IsLocal() && eng.Type != modelengine.EngineStrata
}

// applyNovelSampler 生成链注入：满足条件时把 DRY/XTC 键写入请求 + 返回四步
// 自检段（SelfCheck 开启时；调用方拼到 userPrompt 尾部）。返回生效摘要
// （done 事件回显）与 warn。
func applyNovelSampler(req *ai.ChatRequest, cfg *config.Config, engineMgr *modelengine.Manager, engineID string) (summary map[string]interface{}, selfCheckBlock string, warn string) {
	if cfg != nil && cfg.NovelDryDisabled {
		return map[string]interface{}{"enabled": false}, "", ""
	}
	if !samplerAppliesToEngine(engineMgr, engineID) {
		return map[string]interface{}{"enabled": false, "reason": "engine"}, "", ""
	}
	s, werr := resolveNovelSampler(cfg)
	req.DryMultiplier = s.DryMultiplier
	req.DryBase = s.DryBase
	req.DryAllowedLength = s.DryAllowedLength
	req.DryPenaltyLastN = s.DryPenaltyLastN
	if s.XtcProbability > 0 {
		req.XtcProbability = s.XtcProbability
		req.XtcThreshold = s.XtcThreshold
	}
	selfCheck := s.SelfCheck == nil || *s.SelfCheck
	if selfCheck {
		selfCheckBlock = novelSelfCheckBlock
	}
	return map[string]interface{}{
		"enabled":    true,
		"dry":        true,
		"multiplier": s.DryMultiplier,
		"xtc":        s.XtcProbability > 0,
		"selfCheck":  selfCheck,
	}, selfCheckBlock, werr
}

// novelSelfCheckBlock 四步自检（提示词侧防重复配合；「prompt 预设可选」的落点）。
const novelSelfCheckBlock = "\n\n【收尾前四步自检（写完正文后逐项自查，只输出修正后的正文，不输出自查过程）】\n" +
	"1. 复读本章开头 300 字：新写段落的句式与措辞不得与开头雷同，不得复用同一比喻超过一次；\n" +
	"2. 角色名/称谓/地名与设定完全一致，不得出现未设定的实体；\n" +
	"3. 本章伏笔动作与事实一致：埋了的要可追溯，收了的必须与既有伏笔对应；\n" +
	"4. 时间线与前文连续：不跳跃、不倒流、不与已写章节冲突。"
