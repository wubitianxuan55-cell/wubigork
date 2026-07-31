package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(materialQuery{}) }

// materialQuery queries built-in engineering material properties.
type materialQuery struct{}

func (materialQuery) Name() string { return "material_query" }

func (materialQuery) Description() string {
	return "查询常见工程材料属性（密度、弹性模量、屈服强度、热导率等）。内置钢、铝、铜、混凝土、木材等。"
}

func (materialQuery) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "material":{"type":"string","description":"材料名称，如 steel, aluminum, copper, concrete, wood, glass, titanium, brass, bronze, cast_iron, stainless_steel, carbon_fiber, PVC, acrylic"},
  "property":{"type":"string","description":"属性名称（可选）。不填则返回全部属性。可用属性: density, elastic_modulus, yield_strength, tensile_strength, thermal_conductivity, specific_heat, poisson_ratio, cte(热膨胀系数)"}
},
"required":["material"]
}`)
}

func (materialQuery) ReadOnly() bool { return true }

func (materialQuery) CompactDescription() string     { return compactDesc["material_query"] }
func (materialQuery) CompactSchema() json.RawMessage { return compactSchema["material_query"] }

// materialProps holds key engineering properties for a material.
type materialProps struct {
	Name                string  `json:"name"`
	Category            string  `json:"category"`
	Density             float64 `json:"density_kg_m3,omitempty"`             // kg/m³
	ElasticModulus      float64 `json:"elastic_modulus_GPa,omitempty"`       // GPa
	YieldStrength       float64 `json:"yield_strength_MPa,omitempty"`        // MPa
	TensileStrength     float64 `json:"tensile_strength_MPa,omitempty"`      // MPa
	ThermalConductivity float64 `json:"thermal_conductivity_W_mK,omitempty"` // W/(m·K)
	SpecificHeat        float64 `json:"specific_heat_J_kgK,omitempty"`       // J/(kg·K)
	PoissonRatio        float64 `json:"poisson_ratio,omitempty"`             // dimensionless
	CTE                 float64 `json:"cte_um_mK,omitempty"`                 // μm/(m·K) — coefficient of thermal expansion
	Hardness            float64 `json:"hardness_HB,omitempty"`               // Brinell
}

var materialDB = map[string]materialProps{
	// ---- Metals ----
	"steel": {
		Name: "碳钢 (Carbon Steel)", Category: "metal",
		Density: 7850, ElasticModulus: 200, YieldStrength: 250, TensileStrength: 400,
		ThermalConductivity: 50, SpecificHeat: 490, PoissonRatio: 0.3, CTE: 11.7, Hardness: 130,
	},
	"stainless_steel": {
		Name: "不锈钢 (Stainless Steel 304)", Category: "metal",
		Density: 8000, ElasticModulus: 193, YieldStrength: 215, TensileStrength: 505,
		ThermalConductivity: 16.2, SpecificHeat: 500, PoissonRatio: 0.29, CTE: 17.3, Hardness: 200,
	},
	"aluminum": {
		Name: "铝合金 (Aluminum 6061)", Category: "metal",
		Density: 2700, ElasticModulus: 69, YieldStrength: 276, TensileStrength: 310,
		ThermalConductivity: 167, SpecificHeat: 896, PoissonRatio: 0.33, CTE: 23.6, Hardness: 95,
	},
	"copper": {
		Name: "铜 (Copper)", Category: "metal",
		Density: 8960, ElasticModulus: 110, YieldStrength: 70, TensileStrength: 220,
		ThermalConductivity: 401, SpecificHeat: 385, PoissonRatio: 0.35, CTE: 16.5, Hardness: 50,
	},
	"brass": {
		Name: "黄铜 (Brass)", Category: "metal",
		Density: 8500, ElasticModulus: 100, YieldStrength: 200, TensileStrength: 380,
		ThermalConductivity: 120, SpecificHeat: 380, PoissonRatio: 0.34, CTE: 19, Hardness: 100,
	},
	"bronze": {
		Name: "青铜 (Bronze)", Category: "metal",
		Density: 8800, ElasticModulus: 110, YieldStrength: 180, TensileStrength: 350,
		ThermalConductivity: 71, SpecificHeat: 370, PoissonRatio: 0.34, CTE: 18, Hardness: 90,
	},
	"cast_iron": {
		Name: "铸铁 (Cast Iron)", Category: "metal",
		Density: 7200, ElasticModulus: 120, YieldStrength: 200, TensileStrength: 250,
		ThermalConductivity: 55, SpecificHeat: 460, PoissonRatio: 0.26, CTE: 10.5, Hardness: 180,
	},
	"titanium": {
		Name: "钛合金 (Ti-6Al-4V)", Category: "metal",
		Density: 4430, ElasticModulus: 114, YieldStrength: 880, TensileStrength: 950,
		ThermalConductivity: 6.7, SpecificHeat: 560, PoissonRatio: 0.34, CTE: 9, Hardness: 330,
	},

	// ---- Concrete & Masonry ----
	"concrete": {
		Name: "混凝土 (Concrete)", Category: "construction",
		Density: 2400, ElasticModulus: 30, YieldStrength: 0, TensileStrength: 3,
		ThermalConductivity: 1.7, SpecificHeat: 880, PoissonRatio: 0.2, CTE: 10, Hardness: 0,
	},
	"reinforced_concrete": {
		Name: "钢筋混凝土 (Reinforced Concrete)", Category: "construction",
		Density: 2500, ElasticModulus: 35, YieldStrength: 0, TensileStrength: 3,
		ThermalConductivity: 1.8, SpecificHeat: 880, PoissonRatio: 0.2, CTE: 10, Hardness: 0,
	},
	"brick": {
		Name: "砖 (Brick)", Category: "construction",
		Density: 1800, ElasticModulus: 15, YieldStrength: 0, TensileStrength: 2,
		ThermalConductivity: 0.72, SpecificHeat: 840, PoissonRatio: 0.15, CTE: 6, Hardness: 0,
	},

	// ---- Wood ----
	"wood": {
		Name: "木材（松木, Pine）", Category: "wood",
		Density: 500, ElasticModulus: 10, YieldStrength: 0, TensileStrength: 40,
		ThermalConductivity: 0.14, SpecificHeat: 2300, PoissonRatio: 0.4, CTE: 5, Hardness: 0,
	},
	"oak": {
		Name: "橡木 (Oak)", Category: "wood",
		Density: 750, ElasticModulus: 12, YieldStrength: 0, TensileStrength: 60,
		ThermalConductivity: 0.17, SpecificHeat: 2000, PoissonRatio: 0.4, CTE: 5, Hardness: 0,
	},

	// ---- Polymers ----
	"pvc": {
		Name: "聚氯乙烯 (PVC)", Category: "polymer",
		Density: 1400, ElasticModulus: 3, YieldStrength: 45, TensileStrength: 50,
		ThermalConductivity: 0.19, SpecificHeat: 900, PoissonRatio: 0.38, CTE: 70, Hardness: 80,
	},
	"acrylic": {
		Name: "亚克力 (Acrylic / PMMA)", Category: "polymer",
		Density: 1190, ElasticModulus: 3, YieldStrength: 55, TensileStrength: 70,
		ThermalConductivity: 0.19, SpecificHeat: 1470, PoissonRatio: 0.35, CTE: 75, Hardness: 100,
	},
	"nylon": {
		Name: "尼龙 (Nylon 6/6)", Category: "polymer",
		Density: 1140, ElasticModulus: 2.5, YieldStrength: 55, TensileStrength: 75,
		ThermalConductivity: 0.25, SpecificHeat: 1700, PoissonRatio: 0.39, CTE: 80, Hardness: 100,
	},

	// ---- Composites ----
	"carbon_fiber": {
		Name: "碳纤维复合材料 (Carbon Fiber / Epoxy)", Category: "composite",
		Density: 1600, ElasticModulus: 230, YieldStrength: 0, TensileStrength: 3500,
		ThermalConductivity: 5, SpecificHeat: 800, PoissonRatio: 0.3, CTE: 1.5, Hardness: 0,
	},
	"fiberglass": {
		Name: "玻璃纤维 (Fiberglass / GFRP)", Category: "composite",
		Density: 1800, ElasticModulus: 45, YieldStrength: 0, TensileStrength: 500,
		ThermalConductivity: 0.3, SpecificHeat: 1000, PoissonRatio: 0.25, CTE: 10, Hardness: 0,
	},

	// ---- Glass ----
	"glass": {
		Name: "玻璃 (Soda-Lime Glass)", Category: "ceramic",
		Density: 2500, ElasticModulus: 70, YieldStrength: 0, TensileStrength: 50,
		ThermalConductivity: 0.8, SpecificHeat: 840, PoissonRatio: 0.22, CTE: 8.5, Hardness: 500,
	},
}

// propertyAliases maps various naming conventions to canonical property keys.
var propertyAliases = map[string]string{
	"density":              "density",
	"密度":                   "density",
	"elastic_modulus":      "elastic_modulus",
	"modulus":              "elastic_modulus",
	"young's modulus":      "elastic_modulus",
	"young_modulus":        "elastic_modulus",
	"弹性模量":                 "elastic_modulus",
	"yield_strength":       "yield_strength",
	"yield":                "yield_strength",
	"屈服强度":                 "yield_strength",
	"tensile_strength":     "tensile_strength",
	"tensile":              "tensile_strength",
	"抗拉强度":                 "tensile_strength",
	"thermal_conductivity": "thermal_conductivity",
	"conductivity":         "thermal_conductivity",
	"热导率":                  "thermal_conductivity",
	"specific_heat":        "specific_heat",
	"比热容":                  "specific_heat",
	"poisson_ratio":        "poisson_ratio",
	"泊松比":                  "poisson_ratio",
	"cte":                  "cte",
	"thermal_expansion":    "cte",
	"热膨胀系数":                "cte",
	"hardness":             "hardness",
	"硬度":                   "hardness",
}

// propertyDisplayNames maps canonical keys to display labels.
var propertyDisplayNames = map[string]string{
	"density":              "密度 (Density)",
	"elastic_modulus":      "弹性模量 (Elastic Modulus)",
	"yield_strength":       "屈服强度 (Yield Strength)",
	"tensile_strength":     "抗拉强度 (Tensile Strength)",
	"thermal_conductivity": "热导率 (Thermal Conductivity)",
	"specific_heat":        "比热容 (Specific Heat)",
	"poisson_ratio":        "泊松比 (Poisson's Ratio)",
	"cte":                  "热膨胀系数 (CTE)",
	"hardness":             "硬度 (Hardness)",
}

func (materialQuery) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Material string `json:"material"`
		Property string `json:"property,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	key := strings.TrimSpace(strings.ToLower(p.Material))
	mat, ok := materialDB[key]
	if !ok {
		// List available materials
		var names []string
		for k := range materialDB {
			names = append(names, k)
		}
		return "", fmt.Errorf("未知材料 %q。可用材料: %s", p.Material, strings.Join(names, ", "))
	}

	if p.Property == "" {
		// Return all properties
		out, err := json.MarshalIndent(mat, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal: %w", err)
		}
		return string(out), nil
	}

	// Look up a specific property
	propKey := strings.TrimSpace(strings.ToLower(p.Property))
	if canonical, ok := propertyAliases[propKey]; ok {
		propKey = canonical
	}

	display, hasDisplay := propertyDisplayNames[propKey]
	if !hasDisplay {
		display = propKey
	}

	val, err := getProperty(mat, propKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s — %s: %v", mat.Name, display, val), nil
}

func getProperty(m materialProps, prop string) (interface{}, error) {
	switch prop {
	case "density":
		return m.Density, nil
	case "elastic_modulus":
		return m.ElasticModulus, nil
	case "yield_strength":
		return m.YieldStrength, nil
	case "tensile_strength":
		return m.TensileStrength, nil
	case "thermal_conductivity":
		return m.ThermalConductivity, nil
	case "specific_heat":
		return m.SpecificHeat, nil
	case "poisson_ratio":
		return m.PoissonRatio, nil
	case "cte":
		return m.CTE, nil
	case "hardness":
		return m.Hardness, nil
	default:
		return nil, fmt.Errorf("未知属性 %q。可用属性: density, elastic_modulus, yield_strength, tensile_strength, thermal_conductivity, specific_heat, poisson_ratio, cte, hardness", prop)
	}
}
