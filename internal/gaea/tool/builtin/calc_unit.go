package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(calcUnit{}) }

// calcUnit converts between units across categories: length, weight, temperature, pressure.
type calcUnit struct{}

func (calcUnit) Name() string { return "calc_unit" }

func (calcUnit) Description() string {
	return "单位转换：长度(m/km/mile/ft/inch)、重量(kg/g/lb/oz)、温度(C/F/K)、压力(Pa/kPa/atm/bar/psi)。"
}

func (calcUnit) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "value":{"type":"number","description":"要转换的数值"},
  "from_unit":{"type":"string","description":"源单位，如 m, km, mile, ft, inch, kg, g, lb, oz, C, F, K, Pa, kPa, atm, bar, psi"},
  "to_unit":{"type":"string","description":"目标单位"}
},
"required":["value","from_unit","to_unit"]
}`)
}

func (calcUnit) ReadOnly() bool { return true }

func (calcUnit) CompactDescription() string     { return compactDesc["calc_unit"] }
func (calcUnit) CompactSchema() json.RawMessage { return compactSchema["calc_unit"] }

// unitCategory groups units of the same kind.
type unitCategory int

const (
	catLength unitCategory = iota
	catWeight
	catTemperature
	catPressure
)

// unitInfo holds information about a known unit.
type unitInfo struct {
	category unitCategory
	// ToSI converts a value in this unit to the SI base unit.
	// For temperature, this returns Kelvin.
	toSI func(v float64) float64
	// FromSI converts a value from the SI base unit to this unit.
	fromSI func(v float64) float64
}

var knownUnits = map[string]unitInfo{
	// ---- Length (SI base: meter) ----
	"m":          {catLength, id, id},
	"meter":      {catLength, id, id},
	"meters":     {catLength, id, id},
	"km":         {catLength, func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }},
	"kilometer":  {catLength, func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }},
	"mile":       {catLength, func(v float64) float64 { return v * 1609.344 }, func(v float64) float64 { return v / 1609.344 }},
	"ft":         {catLength, func(v float64) float64 { return v * 0.3048 }, func(v float64) float64 { return v / 0.3048 }},
	"foot":       {catLength, func(v float64) float64 { return v * 0.3048 }, func(v float64) float64 { return v / 0.3048 }},
	"feet":       {catLength, func(v float64) float64 { return v * 0.3048 }, func(v float64) float64 { return v / 0.3048 }},
	"inch":       {catLength, func(v float64) float64 { return v * 0.0254 }, func(v float64) float64 { return v / 0.0254 }},
	"in":         {catLength, func(v float64) float64 { return v * 0.0254 }, func(v float64) float64 { return v / 0.0254 }},
	"cm":         {catLength, func(v float64) float64 { return v * 0.01 }, func(v float64) float64 { return v / 0.01 }},
	"centimeter": {catLength, func(v float64) float64 { return v * 0.01 }, func(v float64) float64 { return v / 0.01 }},
	"mm":         {catLength, func(v float64) float64 { return v * 0.001 }, func(v float64) float64 { return v / 0.001 }},
	"yard":       {catLength, func(v float64) float64 { return v * 0.9144 }, func(v float64) float64 { return v / 0.9144 }},

	// ---- Weight / Mass (SI base: kilogram) ----
	"kg":       {catWeight, id, id},
	"kilogram": {catWeight, id, id},
	"g":        {catWeight, func(v float64) float64 { return v * 0.001 }, func(v float64) float64 { return v / 0.001 }},
	"gram":     {catWeight, func(v float64) float64 { return v * 0.001 }, func(v float64) float64 { return v / 0.001 }},
	"lb":       {catWeight, func(v float64) float64 { return v * 0.45359237 }, func(v float64) float64 { return v / 0.45359237 }},
	"pound":    {catWeight, func(v float64) float64 { return v * 0.45359237 }, func(v float64) float64 { return v / 0.45359237 }},
	"oz":       {catWeight, func(v float64) float64 { return v * 0.028349523125 }, func(v float64) float64 { return v / 0.028349523125 }},
	"ounce":    {catWeight, func(v float64) float64 { return v * 0.028349523125 }, func(v float64) float64 { return v / 0.028349523125 }},
	"ton":      {catWeight, func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }},
	"tonne":    {catWeight, func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }},
	"mg":       {catWeight, func(v float64) float64 { return v * 1e-6 }, func(v float64) float64 { return v / 1e-6 }},
	"t":        {catWeight, func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }},

	// ---- Temperature ----
	// SI base is Kelvin; toSI converts to K, fromSI converts from K.
	"C": {catTemperature,
		func(v float64) float64 { return v + 273.15 },
		func(v float64) float64 { return v - 273.15 },
	},
	"celsius": {catTemperature,
		func(v float64) float64 { return v + 273.15 },
		func(v float64) float64 { return v - 273.15 },
	},
	"F": {catTemperature,
		func(v float64) float64 { return (v-32)*5/9 + 273.15 },
		func(v float64) float64 { return (v-273.15)*9/5 + 32 },
	},
	"fahrenheit": {catTemperature,
		func(v float64) float64 { return (v-32)*5/9 + 273.15 },
		func(v float64) float64 { return (v-273.15)*9/5 + 32 },
	},
	"K":      {catTemperature, id, id},
	"kelvin": {catTemperature, id, id},

	// ---- Pressure (SI base: pascal) ----
	"Pa":         {catPressure, id, id},
	"pascal":     {catPressure, id, id},
	"kPa":        {catPressure, func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }},
	"MPa":        {catPressure, func(v float64) float64 { return v * 1e6 }, func(v float64) float64 { return v / 1e6 }},
	"atm":        {catPressure, func(v float64) float64 { return v * 101325 }, func(v float64) float64 { return v / 101325 }},
	"atmosphere": {catPressure, func(v float64) float64 { return v * 101325 }, func(v float64) float64 { return v / 101325 }},
	"bar":        {catPressure, func(v float64) float64 { return v * 100000 }, func(v float64) float64 { return v / 100000 }},
	"psi":        {catPressure, func(v float64) float64 { return v * 6894.7572931783 }, func(v float64) float64 { return v / 6894.7572931783 }},
	"Torr":       {catPressure, func(v float64) float64 { return v * 133.322 }, func(v float64) float64 { return v / 133.322 }},
	"mmHg":       {catPressure, func(v float64) float64 { return v * 133.322 }, func(v float64) float64 { return v / 133.322 }},
}

func id(v float64) float64 { return v }

func (calcUnit) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Value    float64 `json:"value"`
		FromUnit string  `json:"from_unit"`
		ToUnit   string  `json:"to_unit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	from := strings.TrimSpace(strings.ToLower(p.FromUnit))
	to := strings.TrimSpace(strings.ToLower(p.ToUnit))

	fromInfo, ok := knownUnits[from]
	if !ok {
		return "", fmt.Errorf("unknown unit: %q (supported: m, km, mile, ft, inch, cm, mm, yard, kg, g, lb, oz, ton, mg, C, F, K, Pa, kPa, MPa, atm, bar, psi)", p.FromUnit)
	}
	toInfo, ok := knownUnits[to]
	if !ok {
		return "", fmt.Errorf("unknown unit: %q", p.ToUnit)
	}
	if fromInfo.category != toInfo.category {
		catNames := map[unitCategory]string{catLength: "长度", catWeight: "重量", catTemperature: "温度", catPressure: "压力"}
		return "", fmt.Errorf("无法转换：%s → %s（类别不同）", catNames[fromInfo.category], catNames[toInfo.category])
	}

	// Convert: value -> SI -> target unit
	si := fromInfo.toSI(p.Value)
	result := toInfo.fromSI(si)

	return fmt.Sprintf("%g %s = %g %s", p.Value, p.FromUnit, result, p.ToUnit), nil
}
