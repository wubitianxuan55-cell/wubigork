// mpp_layout_test.go — mppLayout 版本布局表测试（AP4-11）。
//
// 复用 mpp_test.go 的合成构造器（同包）；补齐既有测试未覆盖的版本轨：
// ①mppLayoutFor 按 (major, appVer) 选表断言；②MPP12/MPP14(2010)/MPP14×2013+
// 三条合成流表全链路（VarMeta 12 字节条目、字段偏移查表路由、2013+ 层级栈
// 重建与 lag@14）。真实样本门控仍由 mpp_test.go 承担，本文件不改动其一行。
package schedule

import (
	"encoding/binary"
	"testing"
)

// mppVarMeta12Of 构造 VarMeta12/14 + Var2Data（条目:uid i32@0, off i32@4,
// typ i16@8, unknown i16@10）。
func mppVarMeta12Of(items [][3]interface{}) ([]byte, []byte) {
	head := make([]byte, 24)
	binary.LittleEndian.PutUint32(head, uint32(mppMagic))
	pi32(head, 8, int32(len(items)))
	pi32(head, 20, 0x20)
	meta := head
	data := []byte{}
	for _, it := range items {
		uid := it[0].(int64)
		typ := it[1].(int64)
		blob := it[2].([]byte)
		e := make([]byte, 12)
		pi32(e, 0, int32(uid))
		pi32(e, 4, int32(len(data)))
		pu16(e, 8, uint16(typ))
		meta = append(meta, e...)
		chunk := make([]byte, 4+len(blob))
		pi32(chunk, 0, int32(len(blob)))
		copy(chunk[4:], blob)
		data = append(data, chunk...)
	}
	return meta, data
}

type mppFixedItem = struct {
	byte8 byte
	flags int32
	row   []byte
}

func TestMppLayoutFor(t *testing.T) {
	l9 := mppLayoutFor(mpp9, 9)
	if l9.taskDurOff != 60 || l9.taskPctOff != 122 || l9.taskParentOff != 36 ||
		l9.taskNameKey != 11 || l9.taskWbsKey != 10 || l9.taskRowCap != 768 ||
		l9.taskMinRow != 264 || l9.asgRowSize != 142 || l9.asgMode != mppAsgFixedOrMeta ||
		l9.asgUnitsOff != 54 || l9.consLagOff != 16 || l9.calVarKey != 3 ||
		l9.calHoursOff != 4 || !l9.parseResAsg || !l9.milestoneByMetaBit {
		t.Fatalf("MPP9 布局: %+v", l9)
	}
	l12 := mppLayoutFor(mpp12, 12)
	if l12.taskDurOff != 60 || l12.taskNameKey != 14 || l12.taskWbsKey != 16 ||
		l12.asgMode != mppAsgMetaLocated || l12.asgRowSize != 0 ||
		l12.calVarKey != 8 || l12.calHoursOff != 0 || !l12.parseResAsg {
		t.Fatalf("MPP12 布局: %+v", l12)
	}
	l14 := mppLayoutFor(mpp14, 14)
	if l14.taskDurOff != 42 || l14.taskPctOff != 90 || l14.taskRowCap != 1024 ||
		l14.taskMinRow != 206 || l14.asgRowSize != 110 || l14.asgUnitsOff != 46 ||
		l14.taskNameFallbackKey != 11 || l14.consLagOff != 16 || !l14.parseResAsg {
		t.Fatalf("MPP14 布局: %+v", l14)
	}
	l2013 := mppLayoutFor(mpp14, 15)
	if l2013.taskDurOff != 84 || l2013.taskPctOff != 90 || l2013.taskOutlineOff != 172 ||
		l2013.taskParentOff != -1 || l2013.consLagOff != 14 || !l2013.taskUIDFromID ||
		!l2013.parentByStack || l2013.milestoneByMetaBit || l2013.parseResAsg {
		t.Fatalf("MPP14×2013 布局: %+v", l2013)
	}
	// 选择键:appVer≥15 都是 2013 变体（无 CompObj 回退即 appVer=15）
	if *mppLayoutFor(mpp14, 16) != *l2013 || *mppLayoutFor(mpp14, 99) != *l2013 {
		t.Fatal("appVer≥15 应全落 2013 变体")
	}
	if *mppLayoutFor(mpp14, 14) != *mppLayoutFor(mpp14, 0) {
		t.Fatal("appVer<15 应落 2010 口径")
	}
}

// ── MPP12:键=字段 ID 低 16 位、工期@60、分配 meta 定位 ─────────────

func TestMppSyntheticMPP12(t *testing.T) {
	dayTenths := int32(4800)
	projProps := mppPropsOf(map[int64][]byte{
		37748765: {0xE0, 0x01, 0, 0}, // 480 分/天
	})
	mkRow := func(uid, id, parent int32, dur int32, pct int16) []byte {
		row := make([]byte, 768)
		pi32(row, 0, uid)
		pi32(row, 4, id)
		pi32(row, 36, parent)
		pi32(row, 60, dur)
		pu16(row, 122, uint16(pct))
		return row
	}
	taskMeta, taskData := mppFixedMetaOf(47, []mppFixedItem{
		{0x00, 0, mkRow(301, 1, 0, 3*dayTenths, 0)},
		{0x00, 0, mkRow(302, 2, 301, 2*dayTenths, 25)},
	}, 3)
	nameMeta, nameData := mppVarMeta12Of([][3]interface{}{
		{int64(301), int64(14), utf16Blob("场地平整")},
		{int64(302), int64(14), utf16Blob("土方开挖")},
		{int64(301), int64(16), utf16Blob("1")},
		{int64(302), int64(16), utf16Blob("1.1")},
	})
	// 分配:meta 定位(MPP12 无定长行),Units double@54
	mkAsg := func(uid, task, res int32, units float64) []byte {
		r := make([]byte, 96)
		pi32(r, 0, uid)
		pi32(r, 4, task)
		pi32(r, 8, res)
		pf64(r, 54, units)
		return r
	}
	asgMeta, asgData := mppFixedMetaOf(34, []mppFixedItem{
		{0, 0, mkAsg(1, 302, 7, 150)},
	}, 0)
	// 搭接:301 FS 302（lag@16,2013 前各版本一致）
	mkCons := func(cid, a, b int32, typ int16, lag int32) []byte {
		r := make([]byte, 20)
		pi32(r, 0, cid)
		pi32(r, 4, a)
		pi32(r, 8, b)
		pu16(r, 12, uint16(typ))
		pi32(r, 16, lag)
		return r
	}
	consMeta, consData := mppFixedMetaOf(10, []mppFixedItem{
		{0, 0, mkCons(1, 301, 302, 1, 0)},
	}, 0)
	// 资源（分配回链必需）:uid short@0、名称 Var2Data 键 1、maxUnits@44
	mkRes := func(uid int32, maxUnits float64) []byte {
		r := make([]byte, 128)
		pi32(r, 0, uid)
		pf64(r, 44, maxUnits)
		return r
	}
	resMeta, resData := mppFixedMetaOf(37, []mppFixedItem{
		{0, 0, mkRes(7, 1.0)},
	}, 0)
	resVarMeta, resVarData := mppVarMeta12Of([][3]interface{}{
		{int64(7), int64(1), utf16Blob("挖机")},
	})
	streams := map[string][]byte{
		"Props12":                    mppPropsOf(map[int64][]byte{}),
		"   112/Props":               projProps,
		"   112/TBkndTask/VarMeta":   nameMeta,
		"   112/TBkndTask/Var2Data":  nameData,
		"   112/TBkndTask/FixedMeta": taskMeta,
		"   112/TBkndTask/FixedData": taskData,
		"   112/TBkndCons/FixedMeta": consMeta,
		"   112/TBkndCons/FixedData": consData,
		"   112/TBkndRsc/VarMeta":    resVarMeta,
		"   112/TBkndRsc/Var2Data":   resVarData,
		"   112/TBkndRsc/FixedMeta":  resMeta,
		"   112/TBkndRsc/FixedData":  resData,
		"   112/TBkndAssn/FixedMeta": asgMeta,
		"   112/TBkndAssn/FixedData": asgData,
	}
	p, err := mppParseFromStreams(streams, mpp12, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Tasks) != 2 {
		t.Fatalf("任务数=%d want 2", len(p.Tasks))
	}
	if p.Tasks[0].Name != "场地平整" || p.Tasks[0].Level != 0 { // 有子 → 分组行
		t.Fatalf("t1=%+v", p.Tasks[0])
	}
	if p.Tasks[1].Name != "土方开挖" || p.Tasks[1].Duration != 2 || p.Tasks[1].Progress != 25 {
		t.Fatalf("t2=%+v（键 14/16 路由破坏）", p.Tasks[1])
	}
	if len(p.Links) != 1 || p.Links[0].From != "301" || p.Links[0].To != "302" {
		t.Fatalf("搭接=%+v", p.Links)
	}
	if len(p.Assignments) != 1 || p.Assignments[0].TaskID != "302" ||
		p.Assignments[0].ResourceID != "r7" || p.Assignments[0].Units == nil || *p.Assignments[0].Units != 1.5 {
		t.Fatalf("分配=%+v（meta 定位路由破坏）", p.Assignments)
	}
}

// ── MPP14(2010):工期@42/进度@90、分配定长 110、lag@16 ───────────────

func TestMppSyntheticMPP14_2010(t *testing.T) {
	dayTenths := int32(4800)
	projProps := mppPropsOf(map[int64][]byte{
		37748765: {0xE0, 0x01, 0, 0},
	})
	mkRow := func(uid, id, parent int32, dur int32, pct int16) []byte {
		row := make([]byte, 1024)
		pi32(row, 0, uid)
		pi32(row, 4, id)
		pi32(row, 36, parent)
		pi32(row, 42, dur)
		pu16(row, 90, uint16(pct))
		return row
	}
	taskMeta, taskData := mppFixedMetaOf(47, []mppFixedItem{
		{0x00, 0, mkRow(201, 1, 0, 3*dayTenths, 0)},
		{0x00, 0, mkRow(202, 2, 201, 2*dayTenths, 50)},
		{0x20, 0, mkRow(203, 3, 0, 0, 0)}, // 里程碑:meta[8]&0x20
	}, 3)
	nameMeta, nameData := mppVarMeta12Of([][3]interface{}{
		{int64(201), int64(14), utf16Blob("挖土")},
		{int64(202), int64(14), utf16Blob("垫层")},
		{int64(203), int64(14), utf16Blob("竣工")},
		{int64(201), int64(16), utf16Blob("1")},
		{int64(202), int64(16), utf16Blob("1.1")},
		{int64(203), int64(16), utf16Blob("2")},
	})
	mkCons := func(cid, a, b int32, typ int16, lag int32) []byte {
		r := make([]byte, 20)
		pi32(r, 0, cid)
		pi32(r, 4, a)
		pi32(r, 8, b)
		pu16(r, 12, uint16(typ))
		pi32(r, 16, lag) // 2010:lag@16
		return r
	}
	consMeta, consData := mppFixedMetaOf(10, []mppFixedItem{
		{0, 0, mkCons(1, 201, 202, 1, 0)},
		{0, 0, mkCons(2, 202, 203, 1, -1*dayTenths)},
	}, 0)
	mkRes := func(uid int32, maxUnits float64) []byte {
		r := make([]byte, 128)
		pi32(r, 0, uid)
		pf64(r, 44, maxUnits)
		return r
	}
	resMeta, resData := mppFixedMetaOf(37, []mppFixedItem{
		{0, 0, mkRes(9, 2.0)},
	}, 0)
	resVarMeta, resVarData := mppVarMeta12Of([][3]interface{}{
		{int64(9), int64(1), utf16Blob("塔吊")},
	})
	// 分配:MPP14 定长 110,Units@46
	mkAsg := func(uid, task, res int32, units float64) []byte {
		r := make([]byte, 110)
		pi32(r, 0, uid)
		pi32(r, 4, task)
		pi32(r, 8, res)
		pf64(r, 46, units)
		return r
	}
	asgMeta, asgData := mppFixedMetaOf(34, []mppFixedItem{
		{0, 0, mkAsg(1, 202, 9, 150)},
	}, 0)
	streams := map[string][]byte{
		"Props14":                    mppPropsOf(map[int64][]byte{}),
		"   114/Props":               projProps,
		"   114/TBkndTask/VarMeta":   nameMeta,
		"   114/TBkndTask/Var2Data":  nameData,
		"   114/TBkndTask/FixedMeta": taskMeta,
		"   114/TBkndTask/FixedData": taskData,
		"   114/TBkndCons/FixedMeta": consMeta,
		"   114/TBkndCons/FixedData": consData,
		"   114/TBkndRsc/VarMeta":    resVarMeta,
		"   114/TBkndRsc/Var2Data":   resVarData,
		"   114/TBkndRsc/FixedMeta":  resMeta,
		"   114/TBkndRsc/FixedData":  resData,
		"   114/TBkndAssn/FixedMeta": asgMeta,
		"   114/TBkndAssn/FixedData": asgData,
	}
	p, err := mppParseFromStreams(streams, mpp14, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Tasks) != 3 {
		t.Fatalf("任务数=%d want 3", len(p.Tasks))
	}
	if p.Tasks[0].Name != "挖土" || p.Tasks[0].Level != 0 {
		t.Fatalf("t1=%+v", p.Tasks[0])
	}
	if p.Tasks[1].Duration != 2 || p.Tasks[1].Progress != 50 {
		t.Fatalf("t2=%+v（工期@42/进度@90 路由破坏）", p.Tasks[1])
	}
	if !p.Tasks[2].IsMilestone {
		t.Fatalf("meta[8]&0x20 里程碑位失效:t3=%+v", p.Tasks[2])
	}
	if len(p.Links) != 2 || p.Links[1].Lag != -1 {
		t.Fatalf("搭接=%+v（lag@16 路由破坏）", p.Links)
	}
	if len(p.Resources) != 1 || p.Resources[0].Name != "塔吊" {
		t.Fatalf("资源=%+v", p.Resources)
	}
	if len(p.Assignments) != 1 || p.Assignments[0].Units == nil || *p.Assignments[0].Units != 1.5 {
		t.Fatalf("分配=%+v（定长 110/Units@46 路由破坏）", p.Assignments)
	}
}

// ── MPP14×2013+:工期@84、大纲层级@172、键=ID、层级栈重建、lag@14 ────

func TestMppSyntheticMPP14_2013(t *testing.T) {
	dayTenths := int32(4800)
	projProps := mppPropsOf(map[int64][]byte{
		37748765: {0xE0, 0x01, 0, 0},
	})
	// 2013+ 行:ID@4（uid@0 陈旧键）、工期@84、进度@90、大纲@172;父@36 恒 0 失效
	mkRow := func(id int32, dur int32, pct int16, outline int16) []byte {
		row := make([]byte, 1024)
		pi32(row, 0, 999999) // 陈旧 uid:若误用会与名字表错位 → 名字空
		pi32(row, 4, id)
		pi32(row, 36, 0)
		pi32(row, 84, dur)
		pu16(row, 90, uint16(pct))
		pu16(row, 172, uint16(outline))
		return row
	}
	taskMeta, taskData := mppFixedMetaOf(47, []mppFixedItem{
		{0x00, 0, mkRow(1, 10*dayTenths, 0, 0)}, // 分组
		{0x00, 0, mkRow(2, 3*dayTenths, 40, 1)},
		{0x00, 0, mkRow(3, 0, 0, 1)},           // 零工期=里程碑(2013+ 无 meta 位)
		{0x00, 0, mkRow(4, 2*dayTenths, 0, 0)}, // 根级任务
	}, 3)
	nameMeta, nameData := mppVarMeta12Of([][3]interface{}{
		{int64(1), int64(14), utf16Blob("总平施工")}, // 键=ID（uidFromID 口径）
		{int64(2), int64(14), utf16Blob("土方")},
		{int64(3), int64(14), utf16Blob("竣工")},
		{int64(4), int64(14), utf16Blob("围墙")},
		{int64(1), int64(16), utf16Blob("1")},
		{int64(2), int64(16), utf16Blob("1.1")},
		{int64(3), int64(16), utf16Blob("1.2")},
		{int64(4), int64(16), utf16Blob("2")},
	})
	mkCons := func(cid, a, b int32, typ int16, lag int32) []byte {
		r := make([]byte, 20)
		pi32(r, 0, cid)
		pi32(r, 4, a)
		pi32(r, 8, b)
		pu16(r, 12, uint16(typ))
		pi32(r, 14, lag) // 2013+:lag@14
		return r
	}
	consMeta, consData := mppFixedMetaOf(10, []mppFixedItem{
		{0, 0, mkCons(1, 2, 3, 1, 0)},
		{0, 0, mkCons(2, 1, 4, 1, -1*dayTenths)},
	}, 0)
	streams := map[string][]byte{
		"Props14":                    mppPropsOf(map[int64][]byte{}),
		"   114/Props":               projProps,
		"   114/TBkndTask/VarMeta":   nameMeta,
		"   114/TBkndTask/Var2Data":  nameData,
		"   114/TBkndTask/FixedMeta": taskMeta,
		"   114/TBkndTask/FixedData": taskData,
		"   114/TBkndCons/FixedMeta": consMeta,
		"   114/TBkndCons/FixedData": consData,
		// 2013+ 无 TBkndRsc/TBkndAssn（parseResAsg=false,诚实降级）
	}
	p, err := mppParseFromStreams(streams, mpp14, 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Tasks) != 4 {
		t.Fatalf("任务数=%d want 4", len(p.Tasks))
	}
	if p.Tasks[0].Name != "总平施工" || p.Tasks[0].Level != 0 {
		t.Fatalf("t1=%+v（键=ID 或层级栈重建破坏）", p.Tasks[0])
	}
	if p.Tasks[1].Name != "土方" || p.Tasks[1].Duration != 3 || p.Tasks[1].Progress != 40 || p.Tasks[1].Level != 1 {
		t.Fatalf("t2=%+v（工期@84/进度@90/大纲@172 路由破坏）", p.Tasks[1])
	}
	if !p.Tasks[2].IsMilestone || p.Tasks[2].Name != "竣工" {
		t.Fatalf("t3=%+v（零工期里程碑口径破坏）", p.Tasks[2])
	}
	if p.Tasks[3].Name != "围墙" || p.Tasks[3].Level != 1 || p.Tasks[3].Duration != 2 {
		t.Fatalf("t4=%+v", p.Tasks[3])
	}
	if len(p.Resources) != 0 || len(p.Assignments) != 0 {
		t.Fatalf("2013+ 应不解析资源/分配: %+v %+v", p.Resources, p.Assignments)
	}
	if len(p.Links) != 2 || p.Links[1].Lag != -1 {
		t.Fatalf("搭接=%+v（lag@14 路由破坏）", p.Links)
	}
	r := ComputeCpm(p.Tasks, p.Links)
	// 分组行自带工期清零（汇总不在 CPM 内做）→ 关键链=土方(3d)→竣工(里程碑 0),
	// 1→4 经分组行 0d 且 FS-1d 提前,不构成关键链。
	if !r.OK || r.Duration != 3 {
		t.Fatalf("CPM ok=%v 总工期=%d, want 3（土方 3d→竣工 0）", r.OK, r.Duration)
	}
}
