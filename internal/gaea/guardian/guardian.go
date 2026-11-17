// Package guardian 实现动作级自动评审（codex 蒸馏 T11，guardian-reviewer
// 同构）：在「本该问人」的审批点，先用一次独立的评审模型调用预判——
// allow 则放行（账本记明是机器放行）、deny 则带着理由拒绝（可追认）、
// 评审失败则回落用户审批卡（红线：错误绝不冒充风险，也绝不自动放行）。
//
// 施工难点不在提示词，而在四条边界（asset-07 §九）：
//  1. 输出容错：拒绝但缺字段按保守方向补默认；
//  2. 失败与风险分家：解析失败/超时/预算超限是「评审失败」，不是「高风险」；
//  3. 超时可取消：截止时间从评审入口就生效，父取消始终有效；
//  4. 拒绝计数熔断：连续拒绝过多 → 请求中断回合，由宿主执行（本包只计数）。
package guardian

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// RiskLevel 是评审风险档（上游同枚举）。
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// Authorization 是评审对「用户授权程度」的判断（上游同枚举）。
type Authorization string

const (
	AuthUnknown Authorization = "unknown"
	AuthLow     Authorization = "low"
	AuthMedium  Authorization = "medium"
	AuthHigh    Authorization = "high"
)

// Outcome 是评审结论：只有两值，缺省即评审失败。
type Outcome string

const (
	OutcomeAllow Outcome = "allow"
	OutcomeDeny  Outcome = "deny"
)

// Assessment 是一次完成的评审结论（解析补默认后的归一形态）。
type Assessment struct {
	Risk          RiskLevel
	Authorization Authorization
	Outcome       Outcome
	Rationale     string
}

// ResponseSchema 是评审输出的 JSON Schema（response_format 支持 openings 时
// 直接下发；模板与它对齐——改这里必须同步 review.go 的提示词）。
const ResponseSchema = `{
"type":"object",
"additionalProperties":false,
"properties":{
  "risk_level":         {"enum":["low","medium","high","critical"]},
  "user_authorization": {"enum":["unknown","low","medium","high"]},
  "outcome":            {"enum":["allow","deny"]},
  "rationale":          {"type":"string"}
},
"required":["outcome"]
}`

// 兜底理由文案（固定两句，不给 UI 留空白理由；文案自写非上游原文）。
const (
	fallbackRationaleAllow = "自动评审判定为低风险动作，予以放行。"
	fallbackRationaleDeny  = "自动评审给出拒绝结论，但未附理由。"
)

// Parse 解析评审模型输出，三层容错（asset-07 §二）：
//  1. 严格 JSON；
//  2. 失败则取首个 '{' 到末个 '}' 的切片重试（容忍模型包了一层散文）；
//  3. 仍失败 → 返回错误（评审失败，不是高风险）。
//
// 默认值纪律：outcome 必填（缺即失败）；allow 缺 risk → low；deny 缺 risk →
// high（保守方向）；缺 authorization → unknown；缺/空 rationale → 固定兜底句。
func Parse(text string) (Assessment, error) {
	payload, err := parseJSONLoose(text)
	if err != nil {
		return Assessment{}, err
	}
	if payload.Outcome == nil {
		return Assessment{}, errors.New("guardian: 评审输出缺必填字段 outcome")
	}
	outcome := Outcome(*payload.Outcome)
	if outcome != OutcomeAllow && outcome != OutcomeDeny {
		return Assessment{}, fmt.Errorf("guardian: 评审 outcome 非法值 %q", *payload.Outcome)
	}
	a := Assessment{Outcome: outcome}
	// 风险缺省按结论方向补：允许按低风险、拒绝按高风险（保守）。
	a.Risk = RiskLow
	if outcome == OutcomeDeny {
		a.Risk = RiskHigh
	}
	if payload.RiskLevel != nil {
		r := RiskLevel(*payload.RiskLevel)
		switch r {
		case RiskLow, RiskMedium, RiskHigh, RiskCritical:
			a.Risk = r
		default:
			return Assessment{}, fmt.Errorf("guardian: 评审 risk_level 非法值 %q", *payload.RiskLevel)
		}
	}
	a.Authorization = AuthUnknown
	if payload.UserAuthorization != nil {
		z := Authorization(*payload.UserAuthorization)
		switch z {
		case AuthUnknown, AuthLow, AuthMedium, AuthHigh:
			a.Authorization = z
		default:
			return Assessment{}, fmt.Errorf("guardian: 评审 user_authorization 非法值 %q", *payload.UserAuthorization)
		}
	}
	a.Rationale = fallbackRationaleAllow
	if outcome == OutcomeDeny {
		a.Rationale = fallbackRationaleDeny
	}
	if payload.Rationale != nil && strings.TrimSpace(*payload.Rationale) != "" {
		a.Rationale = strings.TrimSpace(*payload.Rationale)
	}
	return a, nil
}

// rawPayload 是模型原始输出（全部可缺，除 outcome 外由 Parse 补默认）。
type rawPayload struct {
	RiskLevel         *string `json:"risk_level"`
	UserAuthorization *string `json:"user_authorization"`
	Outcome           *string `json:"outcome"`
	Rationale         *string `json:"rationale"`
}

// parseJSONLoose 两层 JSON 提取：原文严格解析 → 首个 { 到末个 } 切片重试。
func parseJSONLoose(text string) (rawPayload, error) {
	var p rawPayload
	if err := json.Unmarshal([]byte(text), &p); err == nil {
		return p, nil
	}
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(text[start:end+1]), &p); err == nil {
			return p, nil
		}
	}
	return rawPayload{}, errors.New("guardian: 评审输出不是合法 JSON")
}
