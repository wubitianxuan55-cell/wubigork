// count.go — 通用符号统计与图层线长（算量内核的通用识别原语）。
//
// 通用性约定（用户拍板「按通用图纸识别方式考虑」——机制与专业无关，
// 不预设任何厂商块名前缀/图例版式）：
//   - 符号=块引用（INSERT）：任何 CAD 图纸的专业符号（灯具/门窗/设备/洁具…）
//     惯例上都以块引用布点——按块名聚合计数天然精确；
//   - 命名=邻近文字众数投票：块实例附近的标注文字（设备标注/图例表/引出
//     标注同构）取众数为该块的显示名，众数占比为置信度；无文字或众数不
//     达阈值的块保留原名，交用户在确认表命名（学习型映射后续持久化）；
//   - 图框归属=实例落点判定（DetectSheets 的 box），与文本清单同口径。
package cad

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"

	gocad "github.com/unitedrhino/go-cad"
)

// CountItem 一种块的统计行（通用符号算量表单元）。
type CountItem struct {
	Block      string         `json:"block"`      // 原始块名（未命名块为内部编码）
	Name       string         `json:"name"`       // 自动命名（邻近文字众数；空=待人工命名）
	Confidence float64        `json:"confidence"` // 命名置信度=众数实例占已配文字实例比 0~1
	Count      int            `json:"count"`
	PerSheet   map[string]int `json:"perSheet"` // 图框序号(1 基)→数量；面板按 open 的 sheets 映射名称
	Sample     [2]float64     `json:"sample"`   // 首实例坐标（定位/预览用）
}

// LayerLen 图层线长统计（管线/墙线等线性量的通用近似——几何长度≠施工量，
// 预留/损耗等系数归组价侧用户口径）。
type LayerLen struct {
	Layer  string  `json:"layer"`
	Length float64 `json:"length"` // 图纸单位（mm 图 1:1 即毫米）
	Lines  int     `json:"lines"`  // 计入的线/多段线实体数
}

// CountResult GaeaCadCount 载荷。
type CountResult struct {
	Items    []CountItem `json:"items"`
	Layers   []LayerLen  `json:"layers"`
	Warnings []string    `json:"warnings"`
}

// 邻近文字配对半径：文字自身高度的倍数（与图纸比例无关——标注与符号的间距
// 惯例上以字高为尺度）。同排优先（|dy|≤2.5 字高），无同排再收邻近。
const (
	nearbyRadiusX = 8 // 水平半径（字高倍数，含右侧图例行/左侧标注）
	nearbyBandY   = 2.5
	// nameMinVotes  众数最低票数（1 票不命名——防单个误配文字污染名称）。
	nameMinVotes = 2
	// nameMinConf   众数最低置信度。
	nameMinConf = 0.5
)

// Count 通用符号统计：全图纸块引用计数 + 邻近文字众数命名 + 图框归属 + 层线长。
func (d *Drawing) Count() *CountResult {
	res := &CountResult{Warnings: []string{}}
	if d.doc == nil {
		return res
	}
	names := blockNames(d.doc)
	texts := d.texts // 已解码转义，含坐标/字高/图框归属

	type agg struct {
		item    CountItem
		votes   map[string]int
		matched int // 配到文字的实例数
	}
	aggs := map[uint64]*agg{}

	for _, ent := range d.doc.ModelSpaceEntities() {
		rv := reflect.ValueOf(ent)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			continue
		}
		switch rv.Elem().Type().Name() {
		case "EntInsert":
			ev := rv.Elem()
			bh := ev.FieldByName("BlockHeader").Uint()
			pos := ev.FieldByName("Position")
			x, y := pos.FieldByName("X").Float(), pos.FieldByName("Y").Float()
			a := aggs[bh]
			if a == nil {
				a = &agg{item: CountItem{
					Block: names[bh], PerSheet: map[string]int{},
					Sample: [2]float64{x, y},
				}, votes: map[string]int{}}
				if a.item.Block == "" {
					a.item.Block = fmt.Sprintf("(块 0x%X)", bh)
				}
				aggs[bh] = a
			}
			a.item.Count++
			if si := d.sheetAt(x, y); si > 0 {
				a.item.PerSheet[fmt.Sprintf("%d", si)]++
			}
			// 邻近文字（同排最近；找不到再取半径内最近）
			if t, ok := nearestText(texts, x, y); ok {
				a.votes[t]++
				a.matched++
			}
		}
	}

	for _, a := range aggs {
		// 众数投票命名
		best, bestN := "", 0
		for k, v := range a.votes {
			if v > bestN || (v == bestN && k < best) {
				best, bestN = k, v
			}
		}
		if bestN >= nameMinVotes {
			a.item.Name = best
			if a.matched > 0 {
				a.item.Confidence = math.Round(float64(bestN)/float64(a.matched)*100) / 100
			}
			if a.item.Confidence < nameMinConf {
				a.item.Name = "" // 众数不稳（实例间文字分歧大）：不命名
				a.item.Confidence = 0
			}
		}
		res.Items = append(res.Items, a.item)
	}
	sort.Slice(res.Items, func(i, j int) bool {
		if res.Items[i].Count != res.Items[j].Count {
			return res.Items[i].Count > res.Items[j].Count
		}
		return res.Items[i].Block < res.Items[j].Block
	})
	if res.Items == nil {
		res.Items = []CountItem{}
	}

	res.Layers = d.LayerLengths()
	if len(res.Items) == 0 {
		res.Warnings = append(res.Warnings, "未发现块引用：该图纸符号可能全部为散线绘制（或天正自定义对象），符号统计不适用；文本清单与问图纸仍可用")
	}
	return res
}

// LayerLengths 图层线长：LINE 长度 + LWPOLYLINE 段长（弧段按弦长近似，
// 拱高忽略——通用近似的诚实口径）。
func (d *Drawing) LayerLengths() []LayerLen {
	if d.doc == nil {
		return []LayerLen{}
	}
	layerName := map[uint64]string{}
	for h, lc := range d.doc.LayerColors {
		if lc.Name != "" {
			layerName[h] = DecodeUnicodeEscape(lc.Name)
		}
	}
	byLayer := map[uint64]*LayerLen{}
	get := func(h uint64) *LayerLen {
		l := byLayer[h]
		if l == nil {
			nm := layerName[h]
			if nm == "" {
				nm = fmt.Sprintf("图层0x%X", h)
			}
			l = &LayerLen{Layer: nm}
			byLayer[h] = l
		}
		return l
	}
	for _, ent := range d.doc.ModelSpaceEntities() {
		rv := reflect.ValueOf(ent)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			continue
		}
		ev := rv.Elem()
		lh := ev.FieldByName("BaseEntity").FieldByName("Layer").Uint()
		switch ev.Type().Name() {
		case "EntLine":
			s, e := ev.FieldByName("Start"), ev.FieldByName("End")
			dx, dy := e.FieldByName("X").Float()-s.FieldByName("X").Float(), e.FieldByName("Y").Float()-s.FieldByName("Y").Float()
			if v := math.Hypot(dx, dy); v > 0 {
				l := get(lh)
				l.Length += v
				l.Lines++
			}
		case "EntLwPolyline":
			verts := ev.FieldByName("Vertices")
			if verts.Kind() == reflect.Slice && verts.Len() >= 2 {
				var total float64
				for i := 0; i+1 < verts.Len(); i++ {
					x1, y1 := verts.Index(i).FieldByName("X").Float(), verts.Index(i).FieldByName("Y").Float()
					x2, y2 := verts.Index(i+1).FieldByName("X").Float(), verts.Index(i+1).FieldByName("Y").Float()
					total += math.Hypot(x2-x1, y2-y1)
				}
				if ev.FieldByName("Flags").Uint()&1 != 0 && verts.Len() >= 3 { // 闭合位 flag&1
					x1, y1 := verts.Index(verts.Len()-1).FieldByName("X").Float(), verts.Index(verts.Len()-1).FieldByName("Y").Float()
					total += math.Hypot(verts.Index(0).FieldByName("X").Float()-x1, verts.Index(0).FieldByName("Y").Float()-y1)
				}
				if total > 0 {
					l := get(lh)
					l.Length += total
					l.Lines++
				}
			}
		}
	}
	out := make([]LayerLen, 0, len(byLayer))
	for _, l := range byLayer {
		out = append(out, *l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Length > out[j].Length })
	return out
}

// sheetAt 点落在哪个图框（1 基序号；0=不在任何图框）。
func (d *Drawing) sheetAt(x, y float64) int {
	for i, s := range d.sheets {
		if x >= s.Box[0] && x <= s.Box[2] && y >= s.Box[1] && y <= s.Box[3] {
			return i + 1
		}
	}
	return 0
}

// nearestText 块实例的邻近文字：同排（|dy|≤2.5 字高）水平半径 8 字高内取最近；
// 无同排命中再放宽取半径内最近（覆盖上/下方标注版式）。
// 名称候选排除标注噪声（isNameNoise）：比例/纯数字类文字（图签区的
// 「A2 1:50」「1:100」、轴号尺寸等）在任何专业的图纸上都存在，作名称只会污染。
func nearestText(texts []Text, x, y float64) (string, bool) {
	bestSame, bestSameD := "", math.Inf(1)
	bestAny, bestAnyD := "", math.Inf(1)
	for _, t := range texts {
		s := strings.TrimSpace(t.Text)
		if s == "" || t.Height <= 0 || isNameNoise(s) {
			continue
		}
		dx, dy := math.Abs(t.X-x), math.Abs(t.Y-y)
		r := t.Height * nearbyRadiusX
		if dx > r || dy > r {
			continue
		}
		dist := math.Hypot(dx, dy)
		if dist < bestAnyD {
			bestAny, bestAnyD = s, dist
		}
		if dy <= t.Height*nearbyBandY && dx < bestSameD {
			bestSame, bestSameD = s, dx
		}
	}
	if bestSame != "" {
		return bestSame, true
	}
	return bestAny, bestAny != ""
}

// reNameNoise 标注噪声模式：N:M 比例（含全角冒号/空格）、纯数字（含小数/
// 负号/百分号——尺寸/数量标注）、单字母（轴号/图幅代号）。
var reNameNoise = regexp.MustCompile(`^[-−]?\d+([.,]\d+)?([：:]\d+([.,]\d+)?)?\s*%?$|^\d+\s*[：:]\s*\d+.*$|^[A-Za-z]$|^[-−]?[0-9]+([.,][0-9]+)?[a-zA-Z°%]*$`)

// isNameNoise 该文字是否不适合做块名称（标注噪声）。
func isNameNoise(s string) bool {
	if reNameNoise.MatchString(s) {
		return true
	}
	// 含比例段（如「A2 1:50」「1:50 立面」）整串视为比例标注。
	return reRatioSeg.MatchString(s)
}

var reRatioSeg = regexp.MustCompile(`\d+\s*[：:]\s*\d+`)

// blockNames BLOCK_HEADER handle → 块名（经内部对象反射读取——object 包为
// go-cad internal，仅消费返回值不解包其类型；DXF/JSON 来源同样经此表）。
func blockNames(doc *gocad.Document) map[uint64]string {
	out := map[uint64]string{}
	for h, o := range doc.InternalObjects() {
		rv := reflect.ValueOf(o)
		if rv.IsNil() || rv.Kind() != reflect.Pointer || rv.Elem().Type().Name() != "ObjGeneric" {
			continue
		}
		ev := rv.Elem()
		if ev.FieldByName("Name").String() != "BLOCK_HEADER" {
			continue
		}
		fields := ev.FieldByName("Fields")
		for i := 0; i < fields.Len(); i++ {
			f := fields.Index(i)
			if f.FieldByName("Key").String() != "name" {
				continue
			}
			v := f.FieldByName("Val")
			if v.Kind() == reflect.Interface && !v.IsNil() {
				v = v.Elem()
			}
			if s := DecodeUnicodeEscape(fmt.Sprintf("%v", v.Interface())); s != "" {
				out[h] = s
			}
		}
	}
	return out
}
