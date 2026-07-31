package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(specJudge{}) }

type specJudge struct{}

func (specJudge) Name() string { return "spec_judge" }

func (specJudge) Description() string {
	return "土壤检测数据超标判定：输入污染物名称和检测浓度，自动对照 GB 36600-2018（建设用地）或 GB 15618-2018（农用地）标准，返回是否超标、超标倍数、对应的筛选值/管控值。"
}

func (specJudge) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "pollutants":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string","description":"污染物名称，如砷、镉、铅、汞、镍、铜、石油烃"},"value":{"type":"number","description":"检测浓度值(mg/kg)"},"unit":{"type":"string","description":"单位，默认mg/kg"}},"required":["name","value"]},"description":"污染物检测数据列表"},
  "land_type":{"type":"string","description":"用地类型：一类用地（居住/学校等敏感用地）、二类用地（工业/商业）、农用地","default":"二类用地"},
  "soil_ph":{"type":"number","description":"土壤pH值（农用地判定需要）"}
},
"required":["pollutants","land_type"]
}`)
}

func (specJudge) ReadOnly() bool { return true }

func (specJudge) CompactDescription() string     { return compactDesc["spec_judge"] }
func (specJudge) CompactSchema() json.RawMessage { return compactSchema["spec_judge"] }

type constructionLimit struct {
	ClassI_Screening  float64
	ClassI_Control    float64
	ClassII_Screening float64
	ClassII_Control   float64
	Aliases           []string
}

var constructionLimits = map[string]constructionLimit{
	"砷":            {20, 120, 60, 140, []string{"as", "As"}},
	"镉":            {20, 47, 65, 172, []string{"cd", "Cd"}},
	"六价铬":          {3.0, 30, 5.0, 78, []string{"cr6+", "Cr6+"}},
	"铜":            {2000, 8000, 18000, 36000, []string{"cu", "Cu"}},
	"铅":            {400, 800, 800, 2500, []string{"pb", "Pb"}},
	"汞":            {8, 33, 38, 82, []string{"hg", "Hg"}},
	"镍":            {150, 600, 900, 2000, []string{"ni", "Ni"}},
	"四氯化碳":         {0.9, 5, 2.8, 36, []string{"ccl4"}},
	"氯仿":           {0.3, 2, 0.9, 10, []string{"chcl3"}},
	"苯":            {1, 4, 4, 40, []string{"benzene"}},
	"甲苯":           {260, 520, 1200, 1200, []string{"toluene"}},
	"乙苯":           {7.2, 36, 28, 80, []string{"ethylbenzene"}},
	"苯并[a]芘":       {0.1, 0.5, 1.5, 15, []string{"bap"}},
	"萘":            {25, 50, 70, 700, []string{"naphthalene"}},
	"石油烃(c10-c40)": {826, 2500, 4500, 9000, []string{"tph"}},
	"氰化物":          {22, 88, 135, 270, []string{"cn-"}},
}

type agriLimit struct {
	PHLe65 float64
	PH5_65 float64
	PH6_75 float64
	PHGt75 float64
}

var agriLimits = map[string]agriLimit{
	"镉": {0.3, 0.4, 0.6, 0.8},
	"汞": {1.3, 1.8, 2.4, 3.4},
	"砷": {40, 40, 30, 25},
	"铅": {70, 90, 120, 170},
	"铬": {150, 150, 200, 250},
	"铜": {50, 50, 100, 100},
	"镍": {60, 70, 100, 190},
	"锌": {200, 200, 250, 300},
}

type pollutantInput struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type judgeInput struct {
	Pollutants []pollutantInput `json:"pollutants"`
	LandType   string           `json:"land_type"`
	SoilPH     *float64         `json:"soil_ph,omitempty"`
}

type judgeResultItem struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Standard    string  `json:"standard"`
	Screening   float64 `json:"screening,omitempty"`
	Control     float64 `json:"control,omitempty"`
	ExceedType  string  `json:"exceed_type"`
	ExceedRatio float64 `json:"exceed_ratio"`
	Conclusion  string  `json:"conclusion"`
}

func (specJudge) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p judgeInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if len(p.Pollutants) == 0 {
		return "", fmt.Errorf("pollutants 不能为空")
	}

	landType := strings.TrimSpace(p.LandType)
	isAgri := strings.Contains(landType, "农")
	isClassI := strings.Contains(landType, "一类") || strings.Contains(landType, "敏感")
	isClassII := !isAgri && !isClassI
	_ = isClassII

	var items []judgeResultItem
	exceeded := 0

	for _, pol := range p.Pollutants {
		if pol.Value <= 0 {
			continue
		}
		item := judgeResultItem{
			Name:       pol.Name,
			Value:      pol.Value,
			ExceedType: "none",
		}
		if isAgri {
			limit, ok := findAgriLimit(pol.Name)
			if !ok {
				item.Conclusion = fmt.Sprintf("「%s」不在 GB 15618-2018 农用地标准列表内", pol.Name)
				items = append(items, item)
				continue
			}
			item.Standard = "GB 15618-2018 农用地土壤污染风险筛选值"
			screening := getAgriScreening(limit, p.SoilPH)
			item.Screening = screening
			if pol.Value > screening {
				item.ExceedType = "screening"
				item.ExceedRatio = roundTo(pol.Value/screening, 2)
				item.Conclusion = fmt.Sprintf("⚠ 超过农用地风险筛选值 %.2f 倍", item.ExceedRatio)
				exceeded++
			} else {
				item.Conclusion = "✓ 未超过农用地风险筛选值"
			}
		} else {
			limit, ok := findConstructionLimit(pol.Name)
			if !ok {
				item.Conclusion = fmt.Sprintf("「%s」不在 GB 36600-2018 标准列表内", pol.Name)
				items = append(items, item)
				continue
			}
			item.Standard = "GB 36600-2018 建设用地土壤污染风险筛选值和管控值"
			screening := limit.ClassII_Screening
			control := limit.ClassII_Control
			if isClassI {
				screening = limit.ClassI_Screening
				control = limit.ClassI_Control
			}
			item.Screening = screening
			item.Control = control
			if pol.Value > control {
				item.ExceedType = "control"
				item.ExceedRatio = roundTo(pol.Value/control, 2)
				item.Conclusion = fmt.Sprintf("🔴 超过管控值 %.2f 倍 — 必须采取管控或修复措施", item.ExceedRatio)
				exceeded++
			} else if pol.Value > screening {
				item.ExceedType = "screening"
				item.ExceedRatio = roundTo(pol.Value/screening, 2)
				item.Conclusion = fmt.Sprintf("🟡 超过筛选值 %.2f 倍 — 需开展详细调查", item.ExceedRatio)
				exceeded++
			} else {
				item.Conclusion = "✓ 未超过筛选值"
			}
		}
		items = append(items, item)
	}

	var b strings.Builder
	if exceeded == 0 {
		fmt.Fprintf(&b, "📊 超标判定结果（%s）\n全部 %d 项污染物均未超标\n\n", landTypeLabel(landType), len(items))
	} else {
		fmt.Fprintf(&b, "📊 超标判定结果（%s）\n%d 项污染物中共 %d 项超标\n\n", landTypeLabel(landType), len(items), exceeded)
	}
	for _, item := range items {
		fmt.Fprintf(&b, "━━━ %s = %.4f mg/kg ━━━\n", item.Name, item.Value)
		if item.Standard != "" {
			fmt.Fprintf(&b, "对标标准：%s\n", item.Standard)
		}
		if item.Screening > 0 {
			fmt.Fprintf(&b, "筛选值：%.4f mg/kg", item.Screening)
			if item.Control > 0 {
				fmt.Fprintf(&b, " | 管控值：%.4f mg/kg", item.Control)
			}
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "判定结果：%s\n\n", item.Conclusion)
	}
	return tool.WrapText(b.String()), nil
}

func findConstructionLimit(name string) (constructionLimit, bool) {
	name = strings.ToLower(name)
	for key, val := range constructionLimits {
		if strings.EqualFold(key, name) {
			return val, true
		}
		for _, alias := range val.Aliases {
			if strings.EqualFold(alias, name) {
				return val, true
			}
		}
	}
	return constructionLimit{}, false
}

func findAgriLimit(name string) (agriLimit, bool) {
	for key, val := range agriLimits {
		if strings.EqualFold(key, name) {
			return val, true
		}
	}
	return agriLimit{}, false
}

func getAgriScreening(limit agriLimit, ph *float64) float64 {
	if ph == nil {
		return limit.PHGt75
	}
	v := *ph
	switch {
	case v <= 5.5:
		return limit.PHLe65
	case v <= 6.5:
		return limit.PH5_65
	case v <= 7.5:
		return limit.PH6_75
	default:
		return limit.PHGt75
	}
}

func landTypeLabel(t string) string {
	if strings.Contains(t, "农") {
		return "农用地"
	}
	if strings.Contains(t, "一类") {
		return "一类用地（敏感）"
	}
	return "二类用地（工业/商业）"
}

func roundTo(v float64, decimals int) float64 {
	p := 1.0
	for i := 0; i < decimals; i++ {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
}
