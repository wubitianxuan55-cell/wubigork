package schedule

// mpp_rows.go — MPP 低层块解析与字节原语（批 27 IN3-03 文件拆分，自 mpp.go
// 原位搬移，零逻辑改动）：Props 块 / VarMeta+Var2Data / FixedMeta+FixedData
// 行读取（含 2013+ 变体 mppAssignRows/mppFixedRowsEvenly）+ 全小端字节原语
// （mppI16/mppU16/mppI32/mppI24/mppF64/mppTimestamp/mppRoundDiv/mppPct）。
// 入口/容器见 mpp.go，版本布局表见 mpp_layout.go。

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"
	"unicode/utf16"
)

// ── 基础块解析:Props / VarMeta / Var2Data / FixedMeta+FixedData ────

// mppParseProps Props 块(9/12/14 同构):头 16 字节(uint16 条目数@12),条目 =
// size(i32) key(i32) attr(i32) data(size),奇数长补 1 字节对齐。mask 非 0 表示
// 整块先按字节异或(口令变换)。
func mppParseProps(raw []byte, mask byte) (map[int64][]byte, error) {
	if len(raw) < 16 {
		return nil, fmt.Errorf("Props 块过短(%d)", len(raw))
	}
	buf := raw
	if mask != 0 {
		buf = mppXor(raw, mask)
	}
	out := map[int64][]byte{}
	count := int(mppU16(buf, 12))
	pos := 16
	for i := 0; i < count && pos+12 <= len(buf); i++ {
		size := int(mppI32(buf, pos))
		key := int64(mppI32(buf, pos+4))
		pos += 12
		if size < 1 || pos+size > len(buf) {
			break
		}
		data := make([]byte, size)
		copy(data, buf[pos:pos+size])
		out[key] = data
		pos += size
		if size%2 != 0 {
			pos++
		}
	}
	return out, nil
}

func mppPropsBlob(props map[int64][]byte, key int64) []byte { return props[key] }

func mppPropsByte(props map[int64][]byte, key int64) byte {
	if b := props[key]; len(b) > 0 {
		return b[0]
	}
	return 0
}

func mppPropsInt(props map[int64][]byte, key int64) int32 {
	if b := props[key]; len(b) >= 4 {
		return mppI32(b, 0)
	}
	return 0
}

// mppPropsTimestamp Props 键 → 日期(仅取天数部分,UTC)。
func mppPropsTimestamp(props map[int64][]byte, key int64) *time.Time {
	if b := props[key]; len(b) >= 4 {
		return mppTimestamp(b, 0)
	}
	return nil
}

// mppParseVarData VarMeta+Var2Data:MPP9 条目 8 字节(uid 3 字节+type 1+off 4),
// MPP12/14 条目 12 字节(uid 4+off 4+type 2+unknown 2;magic 允许 0)。
// Var2Data 每条目 = int32 size + 数据;越界/负 size 跳过(容错)。
func mppParseVarData(metaRaw, dataRaw []byte, v mppVersion) (*mppVarData, error) {
	vd := &mppVarData{table: map[[2]int64]int64{}}
	if len(metaRaw) < 24 {
		return vd, nil // 空表不算错
	}
	magic := uint32(mppI32(metaRaw, 0))
	if magic != mppMagic && magic != 0 {
		return nil, fmt.Errorf("VarMeta 魔数错误(0x%X)", magic)
	}
	entrySize := 8
	if v != mpp9 {
		entrySize = 12
	}
	count := int(mppI32(metaRaw, 8))
	pos := 24
	for i := 0; i < count && pos+entrySize <= len(metaRaw); i++ {
		var uid, typ, off int64
		if v == mpp9 {
			uid, typ, off = int64(mppI24(metaRaw, pos)), int64(metaRaw[pos+3]), int64(mppI32(metaRaw, pos+4))
		} else {
			uid, off, typ = int64(mppI32(metaRaw, pos)), int64(mppI32(metaRaw, pos+4)), int64(mppI16(metaRaw, pos+8))
		}
		vd.table[[2]int64{uid, typ}] = off
		vd.offsets = append(vd.offsets, off)
		pos += entrySize
	}
	sort.Slice(vd.offsets, func(i, j int) bool { return vd.offsets[i] < vd.offsets[j] })
	vd.blobs = map[int64][]byte{}
	for _, off := range vd.offsets {
		if off < 0 || off+4 > int64(len(dataRaw)) {
			continue
		}
		size := int(mppI32(dataRaw, int(off)))
		if size < 0 || off+4+int64(size) > int64(len(dataRaw)) {
			continue
		}
		blob := make([]byte, size)
		copy(blob, dataRaw[off+4:off+4+int64(size)])
		vd.blobs[off] = blob
	}
	return vd, nil
}

type mppVarData struct {
	table   map[[2]int64]int64
	offsets []int64
	blobs   map[int64][]byte
}

func (v *mppVarData) blob(uid, typ int64) []byte {
	off, ok := v.table[[2]int64{uid, typ}]
	if !ok {
		return nil
	}
	return v.blobs[off]
}

// unicode 取 (uid,typ) 条目按 UTF-16LE(NUL 结尾)解码。
func (v *mppVarData) unicode(uid, typ int64) string {
	b := v.blob(uid, typ)
	if len(b) == 0 {
		return ""
	}
	u16 := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		c := binary.LittleEndian.Uint16(b[i:])
		if c == 0 {
			break
		}
		u16 = append(u16, c)
	}
	return string(utf16.Decode(u16))
}

// mppFixedRows FixedMeta+FixedData 通用行定位:meta 条目 itemSize 字节
// (flags i32@0,offset i32@4),行数=(len-16)/itemSize(头内计数不可信);
// 行 i = data 流 [meta[i].off, 下一 meta off),cap 截断。mask 用于口令变换。
func mppFixedRows(metaRaw, dataRaw []byte, itemSize, capSize int, mask byte) ([]mppMetaRow, error) {
	if len(metaRaw) < 16 {
		return nil, fmt.Errorf("FixedMeta 过短(%d)", len(metaRaw))
	}
	if mppMagic != uint32(mppI32(metaRaw, 0)) {
		return nil, fmt.Errorf("FixedMeta 魔数错误(0x%X)", mppI32(metaRaw, 0))
	}
	buf := dataRaw
	if mask != 0 && len(buf) > 0 {
		buf = mppXor(buf, mask)
	}
	count := (len(metaRaw) - 16) / itemSize
	offs := make([]int64, count)
	flags := make([]int32, count)
	for i := 0; i < count; i++ {
		base := 16 + i*itemSize
		flags[i] = mppI32(metaRaw, base)
		offs[i] = int64(mppI32(metaRaw, base+4))
	}
	out := make([]mppMetaRow, 0, count)
	for i := 0; i < count; i++ {
		off := offs[i]
		if off < 0 || off >= int64(len(buf)) {
			continue
		}
		end := int64(len(buf))
		if i+1 < count && offs[i+1] > off && offs[i+1] <= end {
			end = offs[i+1]
		}
		size := end - off
		if size > int64(capSize) {
			size = int64(capSize)
		}
		if size <= 0 {
			continue
		}
		row := make([]byte, size)
		copy(row, buf[off:off+size])
		b8 := byte(0)
		if base := 16 + i*itemSize + 8; base < len(metaRaw) {
			b8 = metaRaw[base]
		}
		out = append(out, mppMetaRow{flags: flags[i], data: row, byte8: b8})
	}
	return out, nil
}

// mppAssignRows 分配行:模式查 mppLayout.asgMode——MPP9=定长 142(行数与 meta
// 计数不符回退 meta 定位,MPP9Reader 同口径);MPP12=meta 定位;MPP14=定长 110。
func mppAssignRows(metaRaw, dataRaw []byte, lay *mppLayout, mask byte) ([]mppMetaRow, error) {
	switch lay.asgMode {
	case mppAsgFixed:
		return mppFixedRowsEvenly(metaRaw, dataRaw, lay.asgMetaItem, lay.asgRowSize, mask)
	case mppAsgFixedOrMeta:
		metaCount := 0
		if len(metaRaw) >= 16 && mppMagic == uint32(mppI32(metaRaw, 0)) {
			metaCount = (len(metaRaw) - 16) / lay.asgMetaItem
		}
		if metaCount == 0 || len(dataRaw)/lay.asgRowSize == metaCount {
			return mppFixedRowsEvenly(metaRaw, dataRaw, lay.asgMetaItem, lay.asgRowSize, mask)
		}
	}
	return mppFixedRows(metaRaw, dataRaw, lay.asgMetaItem, mppSmallRowCap, mask)
}

// mppFixedRowsEvenly 定长切块(meta 仅取 flags;行 i = 数据[i*rs:(i+1)*rs])。
func mppFixedRowsEvenly(metaRaw, dataRaw []byte, itemSize, rowSize int, mask byte) ([]mppMetaRow, error) {
	if len(metaRaw) < 16 || mppMagic != uint32(mppI32(metaRaw, 0)) {
		return nil, fmt.Errorf("FixedMeta 魔数错误")
	}
	buf := dataRaw
	if mask != 0 && len(buf) > 0 {
		buf = mppXor(buf, mask)
	}
	count := len(buf) / rowSize
	if m := (len(metaRaw) - 16) / itemSize; count > m {
		count = m
	}
	out := make([]mppMetaRow, 0, count)
	for i := 0; i < count; i++ {
		row := make([]byte, rowSize)
		copy(row, buf[i*rowSize:(i+1)*rowSize])
		var flags int32
		if base := 16 + i*itemSize; base+4 <= len(metaRaw) {
			flags = mppI32(metaRaw, base)
		}
		out = append(out, mppMetaRow{flags: flags, data: row})
	}
	return out, nil
}

// ── 字节原语(全小端;时间戳/百分比口径蒸馏自 MPPUtility) ────────────

func mppI16(b []byte, off int) int16  { return int16(binary.LittleEndian.Uint16(b[off:])) }
func mppU16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func mppI32(b []byte, off int) int32  { return int32(binary.LittleEndian.Uint32(b[off:])) }

// mppI24 三字节小端整数(VarMeta9 的 uniqueID,高位补 0)。
func mppI24(b []byte, off int) int32 {
	return int32(uint32(b[off]) | uint32(b[off+1])<<8 | uint32(b[off+2])<<16)
}

func mppF64(b []byte, off int) float64 {
	if off+8 > len(b) {
		return 0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))
}

// mppTimestamp 8 字节(int16 时刻@0 + uint16 天数@2)→ 日期;days<100 或
// 65535=空(时刻部分我们只用日期,忽略)。
func mppTimestamp(b []byte, off int) *time.Time {
	if off+4 > len(b) {
		return nil
	}
	days := int64(mppU16(b, off+2))
	if days < 100 || days == 65535 {
		return nil
	}
	t := time.UnixMilli(mppEpochMs + days*86400000).UTC()
	return &t
}

// mppRoundDiv 四舍五入整除(负数对称:符号 * ((|x|+d/2)/d))。
func mppRoundDiv(x, d int64) int {
	if x < 0 {
		return -int((-x + d/2) / d)
	}
	return int((x + d/2) / d)
}

// mppPct 百分比:0..100 之外视为无效返回 0。
func mppPct(v int16) int {
	if v < 0 || v > 100 {
		return 0
	}
	return int(v)
}

// mppXor 口令变换:逐字节与 mask 异或。
func mppXor(b []byte, mask byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		out[i] = c ^ mask
	}
	return out
}
