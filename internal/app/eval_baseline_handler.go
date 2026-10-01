package app

// 评测基线（长篇刀7，七刀收官）：确定性指标聚合快照 + 基线落盘可比 + 逐指标 Δ。
// 规格 进度计划/gaea-novel-eval-baseline-20260930.md（依据长篇规格 §4 裁剪）。
//
// 可比性纪律：快照 body 不含时钟（时间戳只在文件名），全部稳定序聚合——同一
// 状态两次产出逐字节相同；promptSetHash（prompts/*.json 内容 sha256）任一变更
// 即 stale，禁止直接对比。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"encoding/json"

	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
)

// evalSnapshotBody 快照正文（不含任何时钟——确定性验收的载体）。
type evalSnapshotBody struct {
	Chapters      int    `json:"chapters"`
	Chars         int    `json:"chars"`
	PromptSetHash string `json:"prompt_set_hash"`

	Taste struct {
		Mean  float64 `json:"mean"`
		Max   float64 `json:"max"`
		P90   float64 `json:"p90"`
		Worst int     `json:"worst_chapter"` // 最高分（最差）章号；零章=0
	} `json:"taste"`

	Quality struct {
		S1 int `json:"s1"`
		S2 int `json:"s2"`
		S3 int `json:"s3"`
	} `json:"quality"`

	Foreshadow struct {
		Items    int     `json:"items"`
		Planted  int     `json:"planted"`
		Hinted   int     `json:"hinted"`
		Revealed int     `json:"revealed"`
		Recall   float64 `json:"recall"` // revealed/items（0 items=0）
		Findings int     `json:"findings"`
	} `json:"foreshadow"`

	Story struct {
		Findings int            `json:"findings"`
		ByCode   map[string]int `json:"by_code"`
	} `json:"story"`

	StyleDelta *float64 `json:"style_delta"` // nil=无参考档（skipped）

	// v4.445 张力聚合：分析 V2 情感弧线强度（1-10）全书分布——零 LLM 纯聚合。
	// Covered=0 即无分析数据（强度域 1-10，均值 0 不会与真实值混淆）。
	Tension struct {
		Mean    float64 `json:"mean"`
		P90     float64 `json:"p90"`
		Swing   float64 `json:"swing"`   // P90-P10：全书张力波动（变平=节奏塌）
		Covered int     `json:"covered"` // 有强度数据的章数
	} `json:"tension"`

	ContextTotalRunes int `json:"context_total_runes"` // 刀6 清单合计
}

// promptSetHash prompts/*.json 内容 sha256（文件名稳定序拼接）。
func promptSetHash(promptsDir string) string {
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return "unknown"
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(promptsDir, n))
		if err != nil {
			continue
		}
		sum := sha256.Sum256(data)
		_, _ = fmt.Fprintf(h, "%s:%s\n", n, hex.EncodeToString(sum[:]))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// evalChapterList 章号列表（稳定升序，磁盘最大章号口径）。
func evalChapterList(pm *project.Manager) []int {
	max, err := pm.MaxChapterNum()
	if err != nil || max <= 0 {
		return nil
	}
	out := make([]int, 0, max)
	for i := 1; i <= max; i++ {
		out = append(out, i)
	}
	return out
}

// buildEvalSnapshot 聚合快照正文（零 LLM；任一分量失败降级计零不中断——快照
// 的价值在「同口径可比」，缺数据如实为 0/skipped 比整体失败更诚实）。
func (a *writingState) buildEvalSnapshot(pm *project.Manager) *evalSnapshotBody {
	body := &evalSnapshotBody{PromptSetHash: promptSetHash(filepath.Join("..", "..", "prompts"))}
	body.Story.ByCode = map[string]int{}

	chs := evalChapterList(pm)
	body.Chapters = len(chs)

	scores := make([]float64, 0, len(chs))
	for _, n := range chs {
		text, err := pm.ReadChapterAsStitch(n)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		body.Chars += countNonSpace(text)
		if ts, err := novelstyle.ScoreTextNoRef(text); err == nil && ts != nil {
			scores = append(scores, float64(ts.Score))
			if float64(ts.Score) > body.Taste.Max {
				body.Taste.Max = float64(ts.Score)
				body.Taste.Worst = n
			}
		}
		for _, is := range chapterQualityIssuesOf(text) {
			switch is.Severity {
			case "S1":
				body.Quality.S1++
			case "S2":
				body.Quality.S2++
			default:
				body.Quality.S3++
			}
		}
	}
	if n := len(scores); n > 0 {
		sum := 0.0
		for _, s := range scores {
			sum += s
		}
		body.Taste.Mean = round2(sum / float64(n))
		sorted := append([]float64(nil), scores...)
		sort.Float64s(sorted)
		body.Taste.P90 = round2(sorted[(n*9)/10])
		if body.Taste.Max > 0 && body.Taste.Worst == 0 {
			body.Taste.Worst = chs[len(chs)-1]
		}
	}

	if rep, err := a.LintForeshadows(); err == nil {
		body.Foreshadow.Items = rep.Items
		body.Foreshadow.Planted = rep.Planted
		body.Foreshadow.Hinted = rep.Hinted
		body.Foreshadow.Revealed = rep.Revealed
		body.Foreshadow.Findings = len(rep.Findings)
		if rep.Items > 0 {
			body.Foreshadow.Recall = round2(float64(rep.Revealed) / float64(rep.Items))
		}
	}
	if rep, err := a.NovelStoryHealth(); err == nil {
		body.Story.Findings = len(rep.Findings)
		for _, f := range rep.Findings {
			body.Story.ByCode[f.Code]++
		}
	}
	// 文风 Delta：有参考档时全书指纹对照（无档 skipped=nil）
	if sf, err := pm.ReadStyleFingerprint(); err == nil && sf != nil && len(sf.Fingerprint) > 0 {
		if ref, err := novelstyle.LoadFingerprint(sf.Fingerprint); err == nil && ref != nil && body.Chars > 0 {
			samples := make([]string, 0, len(chs))
			for _, n := range chs {
				if t, err := pm.ReadChapterAsStitch(n); err == nil && strings.TrimSpace(t) != "" {
					samples = append(samples, t)
				}
			}
			if obs, err := novelstyle.ComputeFingerprint(samples); err == nil && obs != nil {
				d := round2(novelstyle.Delta(obs, ref))
				body.StyleDelta = &d
			}
		}
	}
	if items, err := a.NovelContextInventory(1); err == nil {
		for _, it := range items {
			if n, _ := it["name"].(string); strings.HasPrefix(n, "合计") {
				if r, ok := it["runes"].(int); ok {
					body.ContextTotalRunes = r
				}
			}
		}
	}
	// v4.445 张力聚合：ReadAnalysisV2File 保持章号升序（Upsert 纪律）→ 稳定序。
	if af, err := pm.ReadAnalysisV2File(); err == nil && af != nil {
		intensities := make([]int, 0, len(af.Items))
		for i := range af.Items {
			if v := af.Items[i].Result.EmotionalArc.Intensity; v > 0 {
				intensities = append(intensities, v)
			}
		}
		body.Tension.Covered = len(intensities)
		if n := len(intensities); n > 0 {
			sum := 0
			for _, v := range intensities {
				sum += v
			}
			body.Tension.Mean = round2(float64(sum) / float64(n))
			sorted := append([]int(nil), intensities...)
			sort.Ints(sorted)
			body.Tension.P90 = round2(float64(sorted[(n*9)/10]))
			body.Tension.Swing = round2(float64(sorted[n-1] - sorted[(n*1)/10]))
		}
	}
	return body
}

func chapterQualityIssuesOf(text string) []novelgate.Issue {
	return novelgate.ChapterQualityIssues(text)
}

// countNonSpaceRunes 非空白 rune 数。
func countNonSpace(text string) int {
	n := 0
	for _, r := range text {
		if r != ' ' && r != '\n' && r != '\r' && r != '\t' {
			n++
		}
	}
	return n
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

// NovelEvalSnapshot 产出指标快照。persist=true 时落 eval/snapshots/<ts>.json
// （时间戳只在文件名，body 确定性）；返回快照正文与文件名（未 persist 为空）。
func (a *writingState) NovelEvalSnapshot(persist bool) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	body := a.buildEvalSnapshot(pm)
	out := map[string]interface{}{
		"chapters": body.Chapters, "chars": body.Chars, "promptSetHash": body.PromptSetHash,
		"taste":   map[string]interface{}{"mean": body.Taste.Mean, "max": body.Taste.Max, "p90": body.Taste.P90, "worstChapter": body.Taste.Worst},
		"quality": map[string]interface{}{"s1": body.Quality.S1, "s2": body.Quality.S2, "s3": body.Quality.S3},
		"foreshadow": map[string]interface{}{
			"items": body.Foreshadow.Items, "planted": body.Foreshadow.Planted, "hinted": body.Foreshadow.Hinted,
			"revealed": body.Foreshadow.Revealed, "recall": body.Foreshadow.Recall, "findings": body.Foreshadow.Findings,
		},
		"story":        map[string]interface{}{"findings": body.Story.Findings, "byCode": body.Story.ByCode},
		"tension": map[string]interface{}{
			"mean": body.Tension.Mean, "p90": body.Tension.P90,
			"swing": body.Tension.Swing, "covered": body.Tension.Covered,
		},
		"contextRunes": body.ContextTotalRunes,
	}
	if body.StyleDelta != nil {
		out["styleDelta"] = *body.StyleDelta
	} else {
		out["styleDelta"] = nil
	}
	savedAs := ""
	if persist {
		dir := filepath.Join(pm.Dir, "eval", "snapshots")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("建快照目录失败: %w", err)
		}
		name := time.Now().UTC().Format("20060102T150405Z") + ".json"
		if err := writeEvalJSON(filepath.Join(dir, name), body); err != nil {
			return nil, fmt.Errorf("写快照失败: %w", err)
		}
		savedAs = name
	}
	out["savedAs"] = savedAs
	return out, nil
}

// NovelEvalBaselineSet 把当前状态的快照设为基线（eval/baseline.json）。
func (a *writingState) NovelEvalBaselineSet() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	body := a.buildEvalSnapshot(pm)
	dir := filepath.Join(pm.Dir, "eval")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("建 eval 目录失败: %w", err)
	}
	if err := writeEvalJSON(filepath.Join(dir, "baseline.json"), body); err != nil {
		return nil, fmt.Errorf("写基线失败: %w", err)
	}
	return map[string]interface{}{"ok": true, "promptSetHash": body.PromptSetHash}, nil
}

// NovelEvalCompare 最近快照 vs 基线的逐指标 Δ。promptSetHash 不一致 → stale=true
// （如实拒绝对比语义，§4「哈希变更禁止直接对比」）；无基线/无快照如实报错。
func (a *writingState) NovelEvalCompare() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	baseRaw, err := os.ReadFile(filepath.Join(pm.Dir, "eval", "baseline.json"))
	if err != nil {
		return nil, fmt.Errorf("尚无基线：先「设为基线」再做对比")
	}
	var base evalSnapshotBody
	if err := json.Unmarshal(baseRaw, &base); err != nil {
		return nil, fmt.Errorf("基线解析失败: %w", err)
	}
	cur := a.buildEvalSnapshot(pm)

	stale := base.PromptSetHash != cur.PromptSetHash
	type pair struct {
		name          string
		from, to      float64
		lowerIsBetter bool
	}
	mk := []pair{
		{"AI 味均值", base.Taste.Mean, cur.Taste.Mean, true},
		{"AI 味 P90", base.Taste.P90, cur.Taste.P90, true},
		{"S1 信号", float64(base.Quality.S1), float64(cur.Quality.S1), true},
		{"S2 信号", float64(base.Quality.S2), float64(cur.Quality.S2), true},
		{"S3 提示", float64(base.Quality.S3), float64(cur.Quality.S3), true},
		{"伏笔回收率", base.Foreshadow.Recall, cur.Foreshadow.Recall, false},
		{"未回收伏笔 findings", float64(base.Foreshadow.Findings), float64(cur.Foreshadow.Findings), true},
		{"结构体检 findings", float64(base.Story.Findings), float64(cur.Story.Findings), true},
		{"上下文合计 rune", float64(base.ContextTotalRunes), float64(cur.ContextTotalRunes), true},
	}
	if base.StyleDelta != nil && cur.StyleDelta != nil {
		mk = append(mk, pair{"文风 Delta", *base.StyleDelta, *cur.StyleDelta, true})
	}
	// v4.445 张力两向：均值下降/波动收窄=节奏塌（worse）——与体检「中段塌陷」
	// 同族的节奏启发式；两侧都有数据才比（任一侧无分析数据=口径不齐，跳过）。
	if base.Tension.Covered > 0 && cur.Tension.Covered > 0 {
		mk = append(mk,
			pair{"张力均值", base.Tension.Mean, cur.Tension.Mean, false},
			pair{"张力波动", base.Tension.Swing, cur.Tension.Swing, false},
		)
	}
	items := make([]map[string]interface{}, 0, len(mk))
	for _, p := range mk {
		d := round2(p.to - p.from)
		dir := "flat"
		if d != 0 {
			better := (d < 0) == p.lowerIsBetter
			if better {
				dir = "better"
			} else {
				dir = "worse"
			}
		}
		items = append(items, map[string]interface{}{"metric": p.name, "from": p.from, "to": p.to, "delta": d, "dir": dir})
	}
	return map[string]interface{}{"stale": stale, "items": items}, nil
}

// writeEvalJSON 稳定序列化落盘（json.Marshal 已按字段序；无 map 无序问题——
// Story.ByCode map 除外，Marshal 会按 key 排序，仍确定性）。
func writeEvalJSON(path string, body *evalSnapshotBody) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
