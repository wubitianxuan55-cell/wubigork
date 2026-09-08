// mpp_test.go — MPP 二进制解析测试(v4.140.0)
//
// 两层:①合成流表直测解码层(mppParseFromStreams,不依赖 CFB 容器);
// ②真实样本门控测试(ParseMpp 全链路,样本在 git-ignored 的 clones/projectlibre
// 样例目录,缺失即 Skip——蒸馏与实证口径见 internal/schedule/mpp.go 头注)。
package schedule

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ── 合成字节构造器 ─────────────────────────────────────────────────

func pu16(b []byte, off int, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }
func pi32(b []byte, off int, v int32)  { binary.LittleEndian.PutUint32(b[off:], uint32(v)) }
func pf64(b []byte, off int, v float64) {
	binary.LittleEndian.PutUint64(b[off:], math.Float64bits(v))
}

func utf16Blob(s string) []byte {
	u16 := []uint16{}
	for _, r := range s {
		u16 = append(u16, uint16(r))
	}
	b := make([]byte, 2*len(u16))
	for i, c := range u16 {
		pu16(b, i*2, c)
	}
	return b
}

// mppPropsOf 构造 Props 块:头 16 字节(条目数@12)+ 键值条目(奇数长对齐)。
func mppPropsOf(items map[int64][]byte) []byte {
	head := make([]byte, 16)
	pu16(head, 12, uint16(len(items)))
	out := head
	keys := []int64{}
	for k := range items {
		keys = append(keys, k)
	}
	for len(keys) > 1 { // 稳定序(仅测试可比对)
		if keys[0] > keys[1] {
			keys[0], keys[1] = keys[1], keys[0]
		}
		for i := 0; i+2 < len(keys); i++ {
			if keys[i] > keys[i+1] {
				keys[i], keys[i+1] = keys[i+1], keys[i]
			}
		}
		break
	}
	for _, k := range keys {
		data := items[k]
		entry := make([]byte, 12+len(data)+len(data)%2)
		pi32(entry, 0, int32(len(data)))
		pi32(entry, 4, int32(k))
		copy(entry[12:], data)
		out = append(out, entry...)
	}
	return out
}

// mppVarMeta9Of 构造 VarMeta9 + Var2Data(条目:uid,typ → 内容)。
func mppVarMeta9Of(items [][3]interface{}) ([]byte, []byte) {
	head := make([]byte, 24)
	binary.LittleEndian.PutUint32(head, uint32(mppMagic))
	pi32(head, 8, int32(len(items)))
	pi32(head, 20, 0x20) // dataSize 占位
	meta := head
	data := []byte{}
	for _, it := range items {
		uid := it[0].(int64)
		typ := byte(it[1].(int64))
		blob := it[2].([]byte)
		e := make([]byte, 8)
		e[0], e[1], e[2] = byte(uid), byte(uid>>8), byte(uid>>16)
		e[3] = typ
		off := int32(len(data))
		pi32(e, 4, off)
		meta = append(meta, e...)
		chunk := make([]byte, 4+len(blob))
		pi32(chunk, 0, int32(len(blob)))
		copy(chunk[4:], blob)
		data = append(data, chunk...)
	}
	return meta, data
}

// mppFixedMetaOf 构造 FixedMeta+FixedData:条目大小 itemSize(flags i32@0,
// off i32@4,byte8@8),行内容依序拼接,偏移自动计算;extraHead=前导哑条目数。
func mppFixedMetaOf(itemSize int, items []struct {
	byte8 byte
	flags int32
	row   []byte
}, extraHead int) ([]byte, []byte) {
	head := make([]byte, 16)
	binary.LittleEndian.PutUint32(head, uint32(mppMagic))
	meta := head
	data := []byte{}
	// 前导哑条目(任务表前 3 条非任务):空行
	for i := 0; i < extraHead; i++ {
		e := make([]byte, itemSize)
		pi32(e, 4, 0)
		meta = append(meta, e...)
	}
	for _, it := range items {
		off := int32(len(data))
		e := make([]byte, itemSize)
		pi32(e, 0, it.flags)
		pi32(e, 4, off)
		e[8] = it.byte8
		meta = append(meta, e...)
		data = append(data, it.row...)
	}
	return meta, data
}

// ── 解码层原语 ─────────────────────────────────────────────────────

func TestMppTimestamp(t *testing.T) {
	// days=8337 → 1983-12-31 + 8337 天 = 2006-10-27(epoch 实证口径)
	b := make([]byte, 4)
	pu16(b, 0, 4800)  // 08:00(十分之一分钟)
	pu16(b, 2, 8337)  // 天数
	ts := mppTimestamp(b, 0)
	if ts == nil {
		t.Fatal("有效时间戳被判空")
	}
	if got := ts.Format("2006-01-02"); got != "2006-10-28" {
		t.Fatalf("日期换算错误:got %s want 2006-10-28", got)
	}
	pu16(b, 2, 65535) // NA 哨兵
	if mppTimestamp(b, 0) != nil {
		t.Fatal("65535 应判空")
	}
	pu16(b, 2, 50) // <100 视为空
	if mppTimestamp(b, 0) != nil {
		t.Fatal("days<100 应判空")
	}
}

func TestMppVarDataUnicode(t *testing.T) {
	meta, data := mppVarMeta9Of([][3]interface{}{
		{int64(7), int64(11), utf16Blob("挖土方")},
	})
	vd, err := mppParseVarData(meta, data, mpp9)
	if err != nil {
		t.Fatal(err)
	}
	if got := vd.unicode(7, 11); got != "挖土方" {
		t.Fatalf("UTF-16 解码错误:got %q", got)
	}
	if got := vd.unicode(8, 11); got != "" {
		t.Fatalf("缺失条目应返回空:got %q", got)
	}
}

func TestMppPropsRoundTrip(t *testing.T) {
	props := mppPropsOf(map[int64][]byte{
		37748738: {0x80, 0x12, 0xA8, 0x20}, // 任意 4 字节
		37748765: {0xE0, 0x01, 0, 0},       // 480
	})
	m, err := mppParseProps(props, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := mppPropsInt(m, 37748765); got != 480 {
		t.Fatalf("minutesPerDay=%d want 480", got)
	}
	if len(mppPropsBlob(m, 37748738)) != 4 {
		t.Fatal("blob 丢失")
	}
}

func TestMppDetectFormat(t *testing.T) {
	// skip28 → len+应用名("Microsoft Project 14.0")→ len+格式串("MSProject.MPP14")
	app := "Microsoft Project 14.0"
	fmtStr := "MSProject.MPP14"
	b := make([]byte, 28)
	b = append(b, make([]byte, 4)...)
	pi32(b, 28, int32(len(app)+1))
	b = append(b, app...)
	b = append(b, 0)
	b = append(b, make([]byte, 4)...)
	pi32(b, len(b)-4, int32(len(fmtStr)+1))
	b = append(b, fmtStr...)
	b = append(b, 0)
	format, appVer, err := mppDetectFormat(b)
	if err != nil {
		t.Fatal(err)
	}
	if format != "MSProject.MPP14" || appVer != 14 {
		t.Fatalf("got %s/%d", format, appVer)
	}
	if _, _, err := mppDetectFormat([]byte("short")); err == nil {
		t.Fatal("短流应报错")
	}
}

func TestMppWeekFromHours(t *testing.T) {
	// 7×60:周一~五 flag=1(缺省周),周六/周日 periods=0(休息)
	b := make([]byte, 7*60)
	for d := 0; d < 7; d++ {
		if d >= 1 && d <= 5 {
			pu16(b, d*60, 1) // flag=1 缺省周(周一~五)
		}
	}
	got := mppWeekFromHours(b)
	want := []int{1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("workweek=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("workweek=%v want %v", got, want)
		}
	}
	// 全休息文件 → 回落缺省
	empty := make([]byte, 7*60)
	if w := mppWeekFromHours(empty); len(w) != 5 {
		t.Fatalf("全休息应回落缺省周,got %v", w)
	}
}

// ── 合成 MPP9 全链路 ───────────────────────────────────────────────

func TestMppSyntheticMPP9(t *testing.T) {
	dayTenths := int32(4800) // 480 分钟/天

	// 工程 Props:开始日期 2026-03-02(周一)= epoch 天数、480 分/天
	startDays := uint16((1711939200000 - mppEpochMs) / 86400000) // 2024-04-01 为可读参照
	_ = startDays
	startBlob := make([]byte, 4)
	days2026 := int64(15340) // 1983-12-31 + 15340 天 = 2025-12-31 +2 → 断言只看非空
	_ = days2026
	pu16(startBlob, 2, 15344) // → 2026-01-04(周日,只验证解码)
	projProps := mppPropsOf(map[int64][]byte{
		37748738: startBlob,
		37748765: {0xE0, 0x01, 0, 0}, // 480
	})

	// 任务:挖土(dur 3d)→ 垫层(dur 2d,pct 50)→ 里程碑(meta8=0x20,dur 0)
	mkRow := func(uid, id, parent int32, dur int32, pct int16) []byte {
		row := make([]byte, 768)
		pi32(row, 0, uid)
		pi32(row, 4, id)
		pi32(row, 36, parent)
		pu16(row, 40, 1)
		pu16(row, 58, 7) // 单位=天
		pi32(row, 60, dur)
		pu16(row, 122, uint16(pct))
		return row
	}
	taskItems := []struct {
		byte8 byte
		flags int32
		row   []byte
	}{
		{0x00, 0, mkRow(101, 1, 0, 3*dayTenths, 0)},
		{0x00, 0, mkRow(102, 2, 0, 2*dayTenths, 50)},
		{0x20, 0, mkRow(103, 3, 0, 0, 0)}, // 里程碑
	}
	taskMeta, taskData := mppFixedMetaOf(47, taskItems, 3)
	nameMeta, nameData := mppVarMeta9Of([][3]interface{}{
		{int64(101), int64(11), utf16Blob("挖土")},
		{int64(102), int64(11), utf16Blob("垫层")},
		{int64(103), int64(11), utf16Blob("竣工")},
	})

	// 搭接:挖土 FS 垫层(lag 0);垫层 FS 竣工(lag -1 天)
	mkCons := func(cid, a, b2 int32, typ int16, lag int32) []byte {
		r := make([]byte, 20)
		pi32(r, 0, cid)
		pi32(r, 4, a)
		pi32(r, 8, b2)
		pu16(r, 12, uint16(typ))
		pu16(r, 14, 7)
		pi32(r, 16, lag)
		return r
	}
	consItems := []struct {
		byte8 byte
		flags int32
		row   []byte
	}{
		{0, 0, mkCons(1, 101, 102, 1, 0)},
		{0, 0, mkCons(2, 102, 103, 1, -1 * dayTenths)},
	}
	consMeta, consData := mppFixedMetaOf(10, consItems, 0)

	// 资源:塔吊(maxUnits 2.0)+ 材料(带计量单位标签)
	mkRes := func(uid int32, maxUnits float64) []byte {
		r := make([]byte, 128)
		pi32(r, 0, uid)
		pf64(r, 44, maxUnits)
		return r
	}
	resItems := []struct {
		byte8 byte
		flags int32
		row   []byte
	}{
		{0, 0, mkRes(1, 2.0)},
		{0, 0, mkRes(2, 1.0)},
	}
	resMeta, resData := mppFixedMetaOf(37, resItems, 0)
	resVarMeta, resVarData := mppVarMeta9Of([][3]interface{}{
		{int64(1), int64(1), utf16Blob("塔吊")},
		{int64(2), int64(1), utf16Blob("混凝土")},
		{int64(2), int64(8), utf16Blob("m³")},
	})

	// 分配:MPP9 定长 142;挖土←塔吊 units 150(→1.5)
	mkAsg := func(uid, task, res int32, units float64) []byte {
		r := make([]byte, 142)
		pi32(r, 0, uid)
		pi32(r, 4, task)
		pi32(r, 8, res)
		pf64(r, 54, units)
		return r
	}
	asgItems := []struct {
		byte8 byte
		flags int32
		row   []byte
	}{
		{0, 0, mkAsg(1, 101, 1, 150)},
	}
	asgMeta, asgData := mppFixedMetaOf(34, asgItems, 0)

	streams := map[string][]byte{
		"Props9":                    mppPropsOf(map[int64][]byte{}),
		"   19/Props":               projProps,
		"   19/TBkndTask/VarMeta":   nameMeta,
		"   19/TBkndTask/Var2Data":  nameData,
		"   19/TBkndTask/FixedMeta": taskMeta,
		"   19/TBkndTask/FixedData": taskData,
		"   19/TBkndCons/FixedMeta": consMeta,
		"   19/TBkndCons/FixedData": consData,
		"   19/TBkndRsc/VarMeta":    resVarMeta,
		"   19/TBkndRsc/Var2Data":   resVarData,
		"   19/TBkndRsc/FixedMeta":  resMeta,
		"   19/TBkndRsc/FixedData":  resData,
		"   19/TBkndAssn/FixedMeta": asgMeta,
		"   19/TBkndAssn/FixedData": asgData,
	}
	p, err := mppParseFromStreams(streams, mpp9, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Tasks) != 3 {
		t.Fatalf("任务数=%d want 3", len(p.Tasks))
	}
	t1, t2, t3 := p.Tasks[0], p.Tasks[1], p.Tasks[2]
	if t1.Name != "挖土" || t1.Duration != 3 || t1.Level != 1 {
		t.Fatalf("t1=%+v", t1)
	}
	if t2.Duration != 2 || t2.Progress != 50 {
		t.Fatalf("t2=%+v", t2)
	}
	if !t3.IsMilestone || t3.Duration != 0 {
		t.Fatalf("里程碑 t3=%+v", t3)
	}
	if t1.ID != "101" || t2.ID != "102" || t3.ID != "103" {
		t.Fatalf("uid 未用作任务 id:%v %v %v", t1.ID, t2.ID, t3.ID)
	}
	if len(p.Links) != 2 {
		t.Fatalf("搭接数=%d want 2", len(p.Links))
	}
	if p.Links[0].From != "101" || p.Links[0].To != "102" || p.Links[0].Type != FS || p.Links[0].Lag != 0 {
		t.Fatalf("link0=%+v", p.Links[0])
	}
	if p.Links[1].Lag != -1 {
		t.Fatalf("负时距换算错误:%+v", p.Links[1])
	}
	if len(p.Resources) != 2 {
		t.Fatalf("资源数=%d want 2", len(p.Resources))
	}
	if p.Resources[0].Name != "塔吊" || p.Resources[0].MaxUnits != 2 {
		t.Fatalf("资源0=%+v", p.Resources[0])
	}
	if p.Resources[1].Type != ResMaterial || p.Resources[1].Unit != "m³" {
		t.Fatalf("材料资源识别错误:%+v", p.Resources[1])
	}
	if len(p.Assignments) != 1 || p.Assignments[0].TaskID != "101" || p.Assignments[0].ResourceID != "r1" {
		t.Fatalf("分配=%+v", p.Assignments)
	}
	if p.Assignments[0].Units == nil || *p.Assignments[0].Units != 1.5 {
		t.Fatalf("units 换算错误:%+v", p.Assignments[0].Units)
	}
	if p.Calendar == nil || len(p.Calendar.Workweek) != 5 {
		t.Fatalf("缺省日历=%+v", p.Calendar)
	}
	// CPM 端到端:3 天 + 2 天(里程碑 0 天,FS-1 提前 1 天)→ 关键链可算
	r := ComputeCpm(p.Tasks, p.Links)
	if !r.OK {
		t.Fatalf("CPM 未通过:%s", r.Error)
	}
	if r.Duration != 5 {
		t.Fatalf("总工期=%d want 5(3+2,里程碑提前 1 天不计)", r.Duration)
	}
}

// ── 真实样本门控测试(git-ignored clones 样例,缺失即 Skip) ─────────

func TestMppRealSamples(t *testing.T) {
	samples := []string{
		"../../clones/projectlibre/projectlibre_build/resources/samples/Commercial construction project plan.mpp",
		"../../clones/projectlibre/projectlibre_build/resources/samples/New Product.mpp",
		"../../clones/projectlibre/projectlibre_build/resources/samples/Microsoft Office Project 2003 deployment.mpp",
	}
	any := false
	for _, path := range samples {
		data, err := os.ReadFile(filepath.FromSlash(path))
		if err != nil {
			continue
		}
		any = true
		p, err := ParseMpp(data)
		if err != nil {
			t.Errorf("%s:ParseMpp 失败:%v", filepath.Base(path), err)
			continue
		}
		if len(p.Tasks) == 0 {
			t.Errorf("%s:0 任务", filepath.Base(path))
			continue
		}
		named := 0
		for _, task := range p.Tasks {
			if task.Name != "" {
				named++
			}
		}
		if named == 0 {
			t.Errorf("%s:任务名全空(名称解码失败)", filepath.Base(path))
		}
		r := ComputeCpm(p.Tasks, p.Links)
		if !r.OK {
			t.Errorf("%s:CPM 未通过:%s(任务 %d/搭接 %d)", filepath.Base(path), r.Error, len(p.Tasks), len(p.Links))
			continue
		}
		if r.Duration <= 0 {
			t.Errorf("%s:总工期=%d", filepath.Base(path), r.Duration)
		}
		t.Logf("%s:任务 %d(有名 %d)/搭接 %d/资源 %d/分配 %d/总工期 %d 天",
			filepath.Base(path), len(p.Tasks), named, len(p.Links), len(p.Resources), len(p.Assignments), r.Duration)
	}
	if !any {
		t.Skip("真实 MPP 样本不存在(clones/projectlibre 样例目录未就位)")
	}
}

// TestMppRealSamples2013 Project 2013+ 变体门控测试（v4.154）:新版 Project
// 不再写 CompObj（工程目录编号判版回退）、任务行工期 @84/var 键=ID/大纲层级
// @172（键位与 2010 漂移,真机样本实证）。样本在 git-ignored clones/mpp2013,
// 缺失即 Skip。资源/分配行 2013+ 键位未钉死,暂不解析（宁缺勿错）。
func TestMppRealSamples2013(t *testing.T) {
	samples := []string{
		"../../clones/mpp2013/重庆干休所总进度计划1（开工）.mpp",
	}
	any := false
	for _, path := range samples {
		data, err := os.ReadFile(filepath.FromSlash(path))
		if err != nil {
			continue
		}
		any = true
		p, err := ParseMpp(data)
		if err != nil {
			t.Errorf("%s:ParseMpp 失败:%v", filepath.Base(path), err)
			continue
		}
		if len(p.Tasks) != 38 {
			t.Errorf("%s:任务 %d, want 38（含 8 分组）", filepath.Base(path), len(p.Tasks))
		}
		named := 0
		groups := 0
		for _, task := range p.Tasks {
			if task.Name != "" {
				named++
			}
			if task.Level == 0 {
				groups++
			}
		}
		if named != len(p.Tasks) {
			t.Errorf("%s:任务名不全（%d/%d）——2013+ var 键=ID 口径破坏", filepath.Base(path), named, len(p.Tasks))
		}
		if groups != 8 {
			t.Errorf("%s:分组 %d, want 8（层级栈重建口径）", filepath.Base(path), groups)
		}
		if len(p.Links) != 42 {
			t.Errorf("%s:搭接 %d, want 42（ID 键空间回链）", filepath.Base(path), len(p.Links))
		}
		r := ComputeCpm(p.Tasks, p.Links)
		if !r.OK || r.Duration != 136 {
			t.Errorf("%s:CPM ok=%v 总工期=%d, want 136（总平施工 136d 主链）", filepath.Base(path), r.OK, r.Duration)
		}
		t.Logf("%s:任务 %d/分组 %d/搭接 %d/总工期 %d 天/开工 %s",
			filepath.Base(path), len(p.Tasks), groups, len(p.Links), r.Duration, p.StartDate)
	}
	if !any {
		t.Skip("2013+ 真机样本不存在(clones/mpp2013 未就位)")
	}
}
