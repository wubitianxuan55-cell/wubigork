package app

// ── 图像模型目录 · 能力/档位/成本静态视图（模型中心侧 owner）──
//
// 长期规划 §2.1 / T0 设计 §5：模型目录事实源归模型中心，图像域只读本视图。
// 本文件是「模型中心侧」的最小扩展：
//   - Modes/Tier/UnitCost/LicenseHint 用静态映射表维护（不做动态猜测）；
//   - 未知模型诚实留空（cost 返回 ""，展示层显示「未定价」而非 0）；
//   - 档位/价格随 market-research 轻扫更新（2026-09-05 调研口径，未核实标「未定价」）。

import "strings"

// 档位常量。
const (
	imageTierLocalFree    = "local_free"
	imageTierCloudPaid    = "cloud_paid"
	imageTierFreeFallback = "free_fallback"
	imageTierRemoteHeavy  = "remote_heavy"
)

// imageModelCatalogMeta 单模型目录元数据（T0 最小字段）。
type imageModelCatalogMeta struct {
	Modes       []string // txt2img | img2img | edit | video | ref
	Tier        string
	UnitCost    string // "0" | "0.18 CNY/张" | "" = 未定价
	LicenseHint string
}

// imageModelCatalogByID 已知图像模型能力/档位/成本映射（键 = 小写 model id）。
// 说明：qwen-image-3.0-pro 的「同模型编辑」为 2026-08 阿里云文档口径，T3 接线时
// 以真机验证为准；此处仅目录标注，不改变任何 T0 能力可用性。
var imageModelCatalogByID = map[string]imageModelCatalogMeta{
	"krea2": {
		Modes:    []string{"txt2img", "img2img"},
		Tier:     imageTierLocalFree,
		UnitCost: "0",
	},
	"z-image-turbo": {
		Modes:    []string{"txt2img", "img2img"},
		Tier:     imageTierLocalFree,
		UnitCost: "0",
	},
	"flux": {
		Modes:    []string{"txt2img"},
		Tier:     imageTierLocalFree,
		UnitCost: "0",
	},
	"grok-imagine-image": {
		Modes:    []string{"txt2img"},
		Tier:     imageTierCloudPaid,
		UnitCost: "未定价",
	},
	"grok-imagine-image-quality": {
		Modes:    []string{"txt2img"},
		Tier:     imageTierCloudPaid,
		UnitCost: "未定价",
	},
	"cogview-4-250304": {
		Modes:    []string{"txt2img"},
		Tier:     imageTierCloudPaid,
		UnitCost: "未定价",
	},
	"glm-image": {
		Modes:    []string{"txt2img"},
		Tier:     imageTierCloudPaid,
		UnitCost: "0.1 CNY/张",
	},
	"qwen-image-3.0-pro": {
		Modes:    []string{"txt2img", "edit"}, // edit 待 T3 真机验证
		Tier:     imageTierCloudPaid,
		UnitCost: "0.18 CNY/张",
	},
}

// imageModelCatalogMetaFor 查已知模型目录；未知模型 ok=false（诚实留空）。
func imageModelCatalogMetaFor(model string) (imageModelCatalogMeta, bool) {
	m := strings.ToLower(strings.TrimSpace(model))
	meta, ok := imageModelCatalogByID[m]
	return meta, ok
}

// imageHubCostAndLicense 取模型成本/许可提示（供登记元数据；未知 = 空 = 诚实留空）。
func imageHubCostAndLicense(model string) (cost, license string) {
	meta, ok := imageModelCatalogMetaFor(model)
	if !ok {
		return "", ""
	}
	return meta.UnitCost, meta.LicenseHint
}
