package cost

// 组价判定参数数据资产（v4.227 规范知识出内核第三刀，用户拍板红线：行业判定
// 参数不写死在 Go 代码里；拍板池 §5.5 形态 a=Apply 确认流不动）。默认表从
// checkparams.json go:embed（数据与逻辑分离，genui/novelstyle 先例）；运行时
// 可被覆盖文件**部分字段覆盖**——惯例路径 .gaea/skills/cost-compose/params.json
// （app 层 ensureCostCheckParams 懒加载），改判定口径不改代码不发版。
// 校验引擎（R1-R3/含量分位带计算）是通用机制，留码。

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// CheckParams 组价校验的可调判定参数。
type CheckParams struct {
	// R1 金额一致性容差：|amount−quantity×price| > max(amountAbsFloor,
	// |amount|×amountRelTol) 才 warn（小额看绝对差，大额看相对差）。
	AmountAbsFloor float64 `json:"amountAbsFloor"`
	AmountRelTol   float64 `json:"amountRelTol"`
	// R3 全局合计两档：Σ>recommended×totalOverRatio → warn 超推荐价；
	// Σ<recommended×totalUnderRatio → info 提管理费/利润/税金口径。
	TotalOverRatio  float64 `json:"totalOverRatio"`
	TotalUnderRatio float64 `json:"totalUnderRatio"`
	// 含量对照：同标题同单位样本 < minContentSamples 不比对（宁缺勿误）。
	MinContentSamples int `json:"minContentSamples"`
	// 含量对照退化带容差：P25==P75（样本同值）时按 ±degenerateBandTol 判带外。
	DegenerateBandTol float64 `json:"degenerateBandTol"`
}

//go:embed checkparams.json
var checkParamsJSON []byte

var checkParams = newCheckParamsSnapshot()

func newCheckParamsSnapshot() *atomic.Pointer[CheckParams] {
	p := &atomic.Pointer[CheckParams]{}
	p.Store(mustLoadCheckParams())
	return p
}

func mustLoadCheckParams() *CheckParams {
	t := &CheckParams{}
	if err := json.Unmarshal(checkParamsJSON, t); err != nil {
		// 内置参数表随构建走，损坏属构建事故——fail-fast 暴露，不带病运行。
		panic(fmt.Sprintf("cost 内置判定参数损坏: %v", err))
	}
	return t
}

func currentCheckParams() *CheckParams { return checkParams.Load() }

// LoadCheckParams 用覆盖文件**部分字段**更新判定参数（未写出的字段保持当前值；
// 与词表的整表替换不同——数值参数逐项覆盖语义更自然）。非法值（≤0）拒绝并
// 报错，不带病运行。
func LoadCheckParams(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ov := &CheckParams{}
	if err := json.Unmarshal(b, ov); err != nil {
		return fmt.Errorf("判定参数覆盖文件解析失败 %s: %w", path, err)
	}
	cur := *checkParams.Load()
	if ov.AmountAbsFloor > 0 {
		cur.AmountAbsFloor = ov.AmountAbsFloor
	}
	if ov.AmountRelTol > 0 {
		cur.AmountRelTol = ov.AmountRelTol
	}
	if ov.TotalOverRatio > 0 {
		cur.TotalOverRatio = ov.TotalOverRatio
	}
	if ov.TotalUnderRatio > 0 {
		cur.TotalUnderRatio = ov.TotalUnderRatio
	}
	if ov.MinContentSamples > 0 {
		cur.MinContentSamples = ov.MinContentSamples
	}
	if ov.DegenerateBandTol > 0 {
		cur.DegenerateBandTol = ov.DegenerateBandTol
	}
	checkParams.Store(&cur)
	return nil
}
