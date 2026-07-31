package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(calcStats{}) }

// calcStats performs basic statistical analysis on a list of numeric values.
type calcStats struct{}

func (calcStats) Name() string { return "calc_stats" }

func (calcStats) Description() string {
	return "基础统计分析：对数值数组计算均值、中位数、标准差、最小值、最大值、计数。"
}

func (calcStats) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "values":{"type":"array","items":{"type":"number"},"description":"数值数组"}
},
"required":["values"]
}`)
}

func (calcStats) ReadOnly() bool { return true }

func (calcStats) CompactDescription() string     { return compactDesc["calc_stats"] }
func (calcStats) CompactSchema() json.RawMessage { return compactSchema["calc_stats"] }

type statsResult struct {
	Count  int     `json:"count"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"std_dev"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Median float64 `json:"median"`
	Q1     float64 `json:"q1,omitempty"`
	Q3     float64 `json:"q3,omitempty"`
	Sum    float64 `json:"sum"`
}

func (calcStats) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Values []float64 `json:"values"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if len(p.Values) == 0 {
		return "", fmt.Errorf("values array must not be empty")
	}

	n := len(p.Values)
	sorted := make([]float64, n)
	copy(sorted, p.Values)
	sort.Float64s(sorted)

	min := sorted[0]
	max := sorted[n-1]
	sum := 0.0
	for _, v := range p.Values {
		sum += v
	}
	mean := sum / float64(n)

	// Standard deviation (population)
	var sqSum float64
	for _, v := range p.Values {
		d := v - mean
		sqSum += d * d
	}
	stdDev := math.Sqrt(sqSum / float64(n))

	// Median
	var median float64
	if n%2 == 0 {
		median = (sorted[n/2-1] + sorted[n/2]) / 2.0
	} else {
		median = sorted[n/2]
	}

	// Quartiles
	var q1, q3 float64
	q1 = percentile(sorted, 0.25)
	q3 = percentile(sorted, 0.75)

	res := statsResult{
		Count:  n,
		Mean:   mean,
		StdDev: stdDev,
		Min:    min,
		Max:    max,
		Median: median,
		Q1:     q1,
		Q3:     q3,
		Sum:    sum,
	}

	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

// percentile computes the p-quantile (0 ≤ p ≤ 1) of sorted data using linear interpolation.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	pos := p * float64(n-1)
	idx := int(pos)
	frac := pos - float64(idx)
	if idx >= n-1 {
		return sorted[n-1]
	}
	return sorted[idx] + frac*(sorted[idx+1]-sorted[idx])
}
