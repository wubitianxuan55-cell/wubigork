// mpp.go — 二进制 MS Project 工程(.mpp)导入解析器(v4.140.0 刀1)
//
// 支持 MPP9(Project 2000-2003)/ MPP12(2007)/ MPP14(2010-2021,含 2013+
// 链接偏移微调)的务实子集:任务(名称/大纲层级/工期/里程碑/进度)、搭接
// (TBkndCons 四类型+有符号时距)、资源(名称/最大单位/材料标签)、分配
// (任务↔资源+投入强度)、工程开始日期、分钟每天、默认日历周工作制。
// 约束日期/日历例外/费率暂不映射——导入后全部任务 auto 模式,排程交回
// CPM 按搭接重算(与 xlsx 导入同口径)。
//
// 格式蒸馏自 ProjectLibre 内嵌 MPXJ 读取器(clones/projectlibre 只读,零搬运):
// 布局=标准 OLE2/CFB 命名流——根 Props9/Props12/Props14;工程目录 "   19"/
// "   112"/"   114"(前导空格是名字的一部分);其下 TBkndTask/Rsc/Assn/Cons/Cal
// 存储,各含 VarMeta/Var2Data/FixedMeta/FixedData。无压缩;唯一变换是口令
// XOR(mask=0xFF-加密码)。MPP9 路径已用真实样本逐字节实证;MPP12/14 偏移
// 按 MPXJ 字段表蒸馏,失败一律 fail-closed 并明示版本。
//
// 关键编码:全部小端;时间戳=int16 时刻(自午夜十分之一分钟)+uint16 天数
// (自 1983-12-31,epoch 毫秒 441676800000;days<100 或 65535=空);工期=
// int32 十分之一分钟,按文件 minutesPerDay 换算工作日;字符串=UTF-16LE
// NUL 结尾;Summary 标志不取自文件而按"有子任务"重算(蒸馏口径)。
package schedule

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
)

type mppVersion int

const (
	mpp9  mppVersion = 9
	mpp12 mppVersion = 12
	mpp14 mppVersion = 14
)

const (
	mppMagic   = uint32(0xFADFADBA)  // VarMeta/FixedMeta 共用魔数
	mppEpochMs = int64(441676800000) // 1983-12-31T00:00:00Z

	// 根 Props 口令双轨键:打开口令标志位(bit0=1 即受保护)与口令加密码
	// (0xFF-code=XOR 掩码,作用于工程 Props 与各 FixedData)。
	mppPropsOpenPwdKey = int64(893386752)
	mppPropsCryptKey   = int64(893386759)

	// 工程 Props 键:分钟/天、工程开始日期、缺省日历周工时 blob。
	mppPropsMinPerDayKey = int64(37748765)
	mppPropsStartDateKey = int64(37748738)
	mppPropsCalWeekKey   = int64(37753736)

	// 资源/分配走 meta 定位时的行上限(MPXJ maxExpectedSize 口径)。
	mppSmallRowCap = 256
)

// ── 版本布局表(AP4-11):同一语义字段按 (major, appVer) 各占一位 ─────

// mppAsgMode 分配行的行定位模式。
type mppAsgMode int

const (
	mppAsgMetaLocated mppAsgMode = iota // meta 定位(MPP12:定长行被弃)
	mppAsgFixed                         // 纯定长(MPP14:110)
	mppAsgFixedOrMeta                   // 定长优先,行数与 meta 计数不符回退 meta 定位(MPP9:142,MPP9Reader 同口径)
)

// mppLayout 单版本二进制布局表:FixedMeta 条目/行布局常量 + 语义字段偏移与
// Var2Data 键位。选择键见 mppLayoutFor;偏移 -1 = 该版本无此字段。
type mppLayout struct {
	// ── 行布局(MPXJ FixedMeta/FixedData 口径) ──
	taskMetaItem int // 任务 FixedMeta 条目大小
	taskRowCap   int // 任务行上限(maxExpectedSize)
	taskMinRow   int // 任务完整行字节数(幻影行判定:实际字节数×100 ≤ minRow×75 → 跳过)
	resMetaItem  int // 资源 FixedMeta 条目大小
	asgMetaItem  int // 分配 FixedMeta 条目大小
	asgRowSize   int // 分配定长行字节数(mppAsgMetaLocated 时为 0)
	asgMode      mppAsgMode
	consMetaItem int // 搭接 FixedMeta 条目大小
	consRowSize  int // 搭接定长行字节数

	// ── 任务字段(行内偏移 / Var2Data 键) ──
	taskNameKey         int64 // 名称键(MPP9=11;MPP12/14=字段 ID 低 16 位)
	taskWbsKey          int64 // WBS 键
	taskNameFallbackKey int64 // 名称主键未命中时的兜底键(MPP14 键位漂移);0=无
	taskParentOff       int   // 父任务 UID i32;-1=无(2013+ 失效恒 0,层级栈重建)
	taskDurOff          int   // 工期 i32(十分之一分钟)
	taskPctOff          int   // 进度 i16
	taskOutlineOff      int   // 大纲层级 i16;-1=无(仅 2013+)
	taskUIDFromID       bool  // 2013+:行内 var 键整体换 ID 键空间,uid 统一取 ID 回链
	milestoneByMetaBit  bool  // true=meta[8]&0x20;false=零工期即里程碑(2013+ meta 无独立位)
	parentByStack       bool  // 2013+:父链由大纲层级栈重建(文件行序即 ID 升序)

	// ── 分配/搭接/日历字段 ──
	asgUnitsOff int   // 分配 Units f64 偏移(÷100,1.0=100%)
	consLagOff  int   // 搭接 lag i32 偏移(2013+ 由 @16 移到 @14)
	calVarKey   int64 // 缺省日历周工时 Var2Data 键(MPP9=3,其余=8)
	calHoursOff int   // 周工时 7×60 块在 blob 内的偏移

	// ── 能力开关 ──
	parseResAsg bool // false=该变体资源/分配键位未钉死,不解析(宁缺勿错)
}

var (
	// MPP9(Project 2000-2003):VarMeta 条目 8 字节;任务行上限 768、完整行
	// 264 字节(字段表最大偏移 BaselineFixedCost@256+8);分配定长 142。
	mppLayoutV9 = mppLayout{
		taskMetaItem: 47, taskRowCap: 768, taskMinRow: 264,
		resMetaItem: 37, asgMetaItem: 34,
		asgRowSize: 142, asgMode: mppAsgFixedOrMeta,
		consMetaItem: 10, consRowSize: 20, consLagOff: 16,
		taskNameKey: 11, taskWbsKey: 10,
		taskParentOff: 36, taskDurOff: 60, taskPctOff: 122, taskOutlineOff: -1,
		milestoneByMetaBit: true,
		asgUnitsOff:        54,
		calVarKey:          3, calHoursOff: 4,
		parseResAsg: true,
	}
	// MPP12(Project 2007):VarMeta 条目 12 字节、Var2Data 键=字段 ID 低 16 位;
	// 行布局与 MPP9 一致;分配改 meta 定位。
	mppLayoutV12 = mppLayout{
		taskMetaItem: 47, taskRowCap: 768, taskMinRow: 264,
		resMetaItem: 37, asgMetaItem: 34,
		asgRowSize: 0, asgMode: mppAsgMetaLocated,
		consMetaItem: 10, consRowSize: 20, consLagOff: 16,
		taskNameKey: 14, taskWbsKey: 16,
		taskParentOff: 36, taskDurOff: 60, taskPctOff: 122, taskOutlineOff: -1,
		milestoneByMetaBit: true,
		asgUnitsOff:        54,
		calVarKey:          8, calHoursOff: 0,
		parseResAsg: true,
	}
	// MPP14(Project 2010):工期移 @42、进度移 @90;任务行上限 1024、完整行
	// 206 字节;名称键 11 作兜底;分配定长 110、Units@46。
	mppLayoutV14 = mppLayout{
		taskMetaItem: 47, taskRowCap: 1024, taskMinRow: 206,
		resMetaItem: 37, asgMetaItem: 34,
		asgRowSize: 110, asgMode: mppAsgFixed,
		consMetaItem: 10, consRowSize: 20, consLagOff: 16,
		taskNameKey: 14, taskWbsKey: 16, taskNameFallbackKey: 11,
		taskParentOff: 36, taskDurOff: 42, taskPctOff: 90, taskOutlineOff: -1,
		milestoneByMetaBit: true,
		asgUnitsOff:        46,
		calVarKey:          8, calHoursOff: 0,
		parseResAsg: true,
	}
	// MPP14 × Project 2013+（appVer≥15）行内变体:工期 @84、进度 @90、大纲
	// 层级 @172、var 键=ID、里程碑=零工期、父链层级栈重建、lag @14;资源/
	// 分配键位未钉死不解析(宁缺勿错,下表分配字段为 2010 惰性拷贝)。
	mppLayoutV142013 = mppLayout{
		taskMetaItem: 47, taskRowCap: 1024, taskMinRow: 206,
		resMetaItem: 37, asgMetaItem: 34,
		asgRowSize: 110, asgMode: mppAsgFixed,
		consMetaItem: 10, consRowSize: 20, consLagOff: 14,
		taskNameKey: 14, taskWbsKey: 16, taskNameFallbackKey: 11,
		taskParentOff: -1, taskDurOff: 84, taskPctOff: 90, taskOutlineOff: 172,
		taskUIDFromID:      true,
		milestoneByMetaBit: false,
		parentByStack:      true,
		asgUnitsOff:        46,
		calVarKey:          8, calHoursOff: 0,
		parseResAsg: false,
	}
)

// mppLayoutFor 按 (major 版本, CompObj 应用主版本) 选布局表:2013+（appVer≥15）
// 是 MPP14 的行内变体,其余 major 一表一版(未知版本已在 ParseMpp 挡在门外)。
func mppLayoutFor(v mppVersion, appVer int) *mppLayout {
	if v == mpp14 && appVer >= 15 {
		return &mppLayoutV142013
	}
	switch v {
	case mpp9:
		return &mppLayoutV9
	case mpp12:
		return &mppLayoutV12
	default:
		return &mppLayoutV14
	}
}

// ── 入口 ───────────────────────────────────────────────────────────

// ParseMpp 解析 .mpp 二进制(OLE2/CFB)为进度计划模型。
func ParseMpp(data []byte) (Project, error) {
	streams, projDir, err := mppOpenStreams(data)
	if err != nil {
		return Project{}, err
	}
	// 版本探测：老文件读 CompObj 应用名；新版 Project（2013+）不再写 CompObj，
	// 回退按工程目录编号判版（"   19"=9/"   112"=12/"   114"=14）。无 CompObj
	// ⇒ 2013+（appVer≥15：TBkndCons lag 偏移 @14 口径）。
	var v mppVersion
	var appVer int
	if comp, ok := streams["\x01CompObj"]; ok {
		format, av, err := mppDetectFormat(comp)
		if err != nil {
			return Project{}, err
		}
		v = mppVersionOfFormat(format)
		if v == 0 {
			return Project{}, fmt.Errorf("不支持的 MPP 格式(%s):请另存为 MS Project XML 或 Project 2003 格式后再导入", format)
		}
		appVer = av
	} else {
		v = mppVersionOfDir(projDir)
		if v == 0 {
			return Project{}, fmt.Errorf("不是有效的 MPP 文件(缺 CompObj 且未识别工程目录):请另存为 MS Project XML 后再导入")
		}
		appVer = 15
	}
	return mppParseFromStreams(streams, v, appVer)
}

// mppVersionOfDir 工程目录编号 → MPP 版本（CompObj 缺失时的回退探测，MPXJ 同款启发式）。
func mppVersionOfDir(dir string) mppVersion {
	switch dir {
	case "   19":
		return mpp9
	case "   112":
		return mpp12
	case "   114":
		return mpp14
	}
	return 0
}

func mppVersionOfFormat(format string) mppVersion {
	switch format {
	case "MSProject.MPP9", "MSProject.MPT9", "MSProject.GLOBAL9":
		return mpp9
	case "MSProject.MPP12", "MSProject.MPT12", "MSProject.GLOBAL12":
		return mpp12
	case "MSProject.MPP14", "MSProject.MPT14", "MSProject.GLOBAL14":
		return mpp14
	}
	return 0
}

// mppParseFromStreams 从语义化流表解码计划(CFB 打开与版本探测之后的部分;
// 独立成函数以便合成流表单测)。
func mppParseFromStreams(streams map[string][]byte, v mppVersion, appVer int) (Project, error) {
	rootPropsName := map[mppVersion]string{mpp9: "Props9", mpp12: "Props12", mpp14: "Props14"}[v]
	projDir := map[mppVersion]string{mpp9: "   19", mpp12: "   112", mpp14: "   114"}[v]

	// 根 Props 块只承载口令位与加密掩码；新版 Project（2013+）不再写根 Props，
	// 缺流=未保护，按空表处理（工程属性一律读 projDir/Props）。
	var rootProps map[int64][]byte
	if len(streams[rootPropsName]) > 0 {
		var err error
		rootProps, err = mppParseProps(streams[rootPropsName], 0)
		if err != nil {
			return Project{}, fmt.Errorf("MPP 工程属性读取失败:%w", err)
		}
	}
	if mppPropsByte(rootProps, mppPropsOpenPwdKey)&0x01 != 0 {
		return Project{}, fmt.Errorf("MPP 文件受打开口令保护:请先在 Project 中取消口令并另存,再导入")
	}
	mask := byte(0)
	if code := mppPropsByte(rootProps, mppPropsCryptKey); code != 0 {
		mask = 0xFF - code // 口令变换:作用于工程 Props 与各 FixedData
	}

	projProps, err := mppParseProps(streams[projDir+"/Props"], mask)
	if err != nil || len(projProps) == 0 {
		return Project{}, fmt.Errorf("MPP 缺少工程属性流(%s/Props):文件可能不完整", projDir)
	}

	minPerDay := int64(480)
	if n := int64(mppPropsInt(projProps, mppPropsMinPerDayKey)); n > 0 {
		minPerDay = n
	}
	dayTenths := minPerDay * 10 // 工期十分之一分钟 → 工作日除数

	p := Project{}
	if ts := mppPropsTimestamp(projProps, mppPropsStartDateKey); ts != nil {
		p.StartDate = ts.Format("2006-01-02")
	}
	if p.StartDate == "" {
		p.StartDate = time.Now().Format("2006-01-02")
	}

	// 版本布局表:同一语义字段的偏移/键位一律查表(mppLayoutFor 按 major+appVer 选)。
	lay := mppLayoutFor(v, appVer)

	// ── 任务 ────────────────────────────────────────────────────────
	tasks, err := mppParseTasks(streams, projDir, v, lay, mask)
	if err != nil {
		return Project{}, err
	}

	// ── 搭接(TBkndCons:MPP9/12/14 唯一的链接来源,20 字节定长行)─────
	rawLinks, err := mppParseLinks(streams, projDir, lay, mask)
	if err != nil {
		return Project{}, fmt.Errorf("MPP 搭接表读取失败:%w", err)
	}

	// ── 资源(名称在 Var2Data 键 1,各版本一致)──────────────────────
	// Project 2013+（appVer≥15）资源/分配行的键位与偏移尚未钉死（真机样本
	// 实证与 2010 口径漂移）——宁缺勿错,不解析（诚实降级,任务/搭接/日历不受影响）。
	resources := []mppRawRes{}
	rawAsgs := []mppRawAsg{}
	if lay.parseResAsg {
		resVar, _ := mppParseVarData(streams[projDir+"/TBkndRsc/VarMeta"], streams[projDir+"/TBkndRsc/Var2Data"], v)
		resRows, _ := mppFixedRows(streams[projDir+"/TBkndRsc/FixedMeta"], streams[projDir+"/TBkndRsc/FixedData"], lay.resMetaItem, mppSmallRowCap, mask)
		for _, r := range resRows {
			if len(r.data) < 48 {
				continue
			}
			uid := int64(mppI16(r.data, 0)) // createResourceMap 口径:short@0
			resources = append(resources, mppRawRes{
				uid:      uid,
				name:     resVar.unicode(uid, 1),
				label:    resVar.unicode(uid, 8), // 材料计量单位(有=材料资源)
				maxUnits: mppF64(r.data, 44),
			})
		}

		// ── 分配(MPP9 定长 142/计数不符回退 meta;MPP12 meta 定位;MPP14 定长 110)──
		if asgMeta, ok := streams[projDir+"/TBkndAssn/FixedMeta"]; ok {
			asgRows, err2 := mppAssignRows(asgMeta, streams[projDir+"/TBkndAssn/FixedData"], lay, mask)
			if err2 != nil {
				return Project{}, fmt.Errorf("MPP 分配表读取失败:%w", err2)
			}
			unitsOff := lay.asgUnitsOff // MPP9/12:54、MPP14:46(÷100,1.0=100%)
			for _, r := range asgRows {
				if r.flags&0xFF != 0 || len(r.data) < unitsOff+8 {
					continue // meta 首字节非 0=已删除分配(MPXJ meta[0]!=0 口径)
				}
				rawAsgs = append(rawAsgs, mppRawAsg{
					taskUID: int64(mppI32(r.data, 4)),
					resUID:  int64(mppI32(r.data, 8)),
					units:   mppF64(r.data, unitsOff) / 100,
				})
			}
		}
	}

	// ── 默认日历周工作制(每周 7×60 字节;props 兜底;例外日不解析)────
	workweek := mppDefaultWeek()
	calVar, _ := mppParseVarData(streams[projDir+"/TBkndCal/VarMeta"], streams[projDir+"/TBkndCal/Var2Data"], v)
	if blob := calVar.blob(1, lay.calVarKey); len(blob) >= lay.calHoursOff+7*60 {
		workweek = mppWeekFromHours(blob[lay.calHoursOff:])
	} else if blob := mppPropsBlob(projProps, mppPropsCalWeekKey); len(blob) >= 4+7*60 {
		workweek = mppWeekFromHours(blob[4:])
	}

	return mppBuildProject(p, tasks, rawLinks, resources, rawAsgs, workweek, dayTenths, v)
}

// ── 中间结构与模型装配 ─────────────────────────────────────────────

type mppMetaRow struct {
	flags int32 // FixedMeta 条目 flags(bit0x02 任务/日历删除;分配非 0=删除)
	data  []byte
	byte8 byte // FixedMeta 行第 8 字节(里程碑位 metaData[8]&0x20)
}

type mppTask struct {
	uid, id        int64
	parentUID      int64
	name, wbs      string
	durationTenths int64
	pct            int
	milestone      bool
	deleted        bool
	outline        int // 2013+ 大纲层级（0=项目标题行，逐级 +1；父链由层级栈重建）
}

type mppRawLink struct {
	pred, succ, relType, lagTenths int64
}

type mppRawRes struct {
	uid         int64
	name, label string
	maxUnits    float64
}

type mppRawAsg struct {
	taskUID, resUID int64
	units           float64
}

// mppParseTasks 任务表:跳过前 3 条 meta;flags bit0x02=删除(uid 占位防误挂);
// 8 字节行=null 占位;<75% 满=幻影行。字段偏移与 Var2Data 键一律查 mppLayout:
// MPP9/12 同位(父@36/工期@60/进度@122,名称/WBS 键 11/10),MPP14 自成一套
// (父@36/工期@42/进度@90,键 14/16);Project 2013+（appVer≥15）MPP14 再变体:
// 工期 @84、进度 @90、var 数据键=ID（uid@0 变陈旧键,真机样本实证）、大纲层级
// @172、里程碑=零工期,父链由层级栈重建。
func mppParseTasks(streams map[string][]byte, projDir string, v mppVersion, lay *mppLayout, mask byte) ([]*mppTask, error) {
	taskVar, err := mppParseVarData(streams[projDir+"/TBkndTask/VarMeta"], streams[projDir+"/TBkndTask/Var2Data"], v)
	if err != nil {
		return nil, fmt.Errorf("MPP 任务数据读取失败:%w", err)
	}
	rows, err := mppFixedRows(streams[projDir+"/TBkndTask/FixedMeta"], streams[projDir+"/TBkndTask/FixedData"], lay.taskMetaItem, lay.taskRowCap, mask)
	if err != nil {
		return nil, fmt.Errorf("MPP 任务表读取失败:%w", err)
	}
	tasks := []*mppTask{}
	for i := 3; i < len(rows); i++ {
		row, meta := rows[i].data, rows[i]
		if row == nil {
			continue
		}
		if meta.flags&0x02 != 0 { // 已删除:uid 占位(子行父指针校验用)
			tasks = append(tasks, &mppTask{uid: int64(mppI16(row, 0)), deleted: true})
			continue
		}
		if len(row) == 8 { // null 占位任务
			continue
		}
		if len(row)*100 <= lay.taskMinRow*75 { // <75% 满视为幻影行(MPXJ 同口径)
			continue
		}
		t := &mppTask{uid: int64(mppI32(row, 0)), id: int64(mppI32(row, 4))}
		if lay.taskUIDFromID {
			// 2013+ 变体:行内 var 键整体换 ID 键空间（uid@0 是陈旧键,与名字
			// 表大面积错位——真机样本实证），统一取 ID 为回链键。
			t.uid = t.id
		}
		t.durationTenths = int64(mppI32(row, lay.taskDurOff))
		t.pct = mppPct(mppI16(row, lay.taskPctOff))
		if lay.milestoneByMetaBit {
			t.milestone = meta.byte8&0x20 != 0
		} else {
			t.milestone = t.durationTenths == 0 // 零工期=里程碑(2013+ meta 无独立位)
		}
		if lay.taskParentOff >= 0 {
			t.parentUID = int64(mppI32(row, lay.taskParentOff))
		}
		if lay.taskOutlineOff >= 0 {
			t.outline = int(mppI16(row, lay.taskOutlineOff))
		}
		t.name = taskVar.unicode(t.uid, lay.taskNameKey)
		if t.name == "" && lay.taskNameFallbackKey != 0 {
			t.name = taskVar.unicode(t.uid, lay.taskNameFallbackKey) // MPP14 键位兜底
		}
		t.wbs = taskVar.unicode(t.uid, lay.taskWbsKey)
		tasks = append(tasks, t)
	}
	if lay.parentByStack {
		// 父链重建:文件行序即 ID 升序（真机样本实证），层级栈配对——
		// 每行的父=其前最近的更浅层级行（parentUID@36 在 2013+ 失效恒 0）。
		stack := []*mppTask{}
		for _, t := range tasks {
			if t.deleted {
				continue
			}
			for len(stack) > 0 && stack[len(stack)-1].outline >= t.outline {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				t.parentUID = stack[len(stack)-1].uid
			}
			stack = append(stack, t)
		}
	}
	return tasks, nil
}

// mppParseLinks 搭接行:constraintID 必须严格递增(去重);类型 0=FF 1=FS
// 2=SF 3=SS;lag 有符号十分之一分钟(偏移查表:2013+ 由 @16 移到 @14)。
func mppParseLinks(streams map[string][]byte, projDir string, lay *mppLayout, mask byte) ([]mppRawLink, error) {
	rows, err := mppFixedRows(streams[projDir+"/TBkndCons/FixedMeta"], streams[projDir+"/TBkndCons/FixedData"], lay.consMetaItem, lay.consRowSize, mask)
	if err != nil {
		return nil, err
	}
	links := []mppRawLink{}
	lastCID := int64(-1)
	for _, r := range rows {
		if r.flags&0xFFFF != 0 || len(r.data) < lay.consRowSize {
			continue // meta short@0 非 0=已删除(ConstraintFactory 口径)
		}
		cid := int64(mppI32(r.data, 0))
		if cid <= lastCID {
			continue
		}
		lastCID = cid
		pred, succ := int64(mppI32(r.data, 4)), int64(mppI32(r.data, 8))
		if pred == succ {
			continue
		}
		links = append(links, mppRawLink{
			pred:      pred,
			succ:      succ,
			relType:   int64(mppI16(r.data, 12)),
			lagTenths: int64(mppI32(r.data, lay.consLagOff)),
		})
	}
	return links, nil
}

// mppBuildProject 装配模型:两级大纲(有活子=分组,否则子任务;更深层级压平)、
// 任务按文件行序(ID 升序;MPP14 顺序键在 Fixed2Data,此处按 ID 的口径对
// 2013+ 重排文件的次序可能有个别出入)、搭接/分配按 uid 回链(断链丢弃)。
func mppBuildProject(p Project, tasks []*mppTask, rawLinks []mppRawLink, resources []mppRawRes, assignments []mppRawAsg, workweek []int, dayTenths int64, v mppVersion) (Project, error) {
	byUID := map[int64]*mppTask{}
	alive := []*mppTask{}
	for _, t := range tasks {
		if t.deleted {
			continue
		}
		byUID[t.uid] = t
		alive = append(alive, t)
	}
	if len(alive) == 0 {
		return Project{}, fmt.Errorf("MPP 未解出任何任务:格式版本可能不受支持(v%d)", v)
	}
	hasName := false
	for _, t := range alive {
		if t.name != "" {
			hasName = true
			break
		}
	}
	if !hasName {
		return Project{}, fmt.Errorf("MPP 任务名称解码失败:文件变体可能不受支持(v%d)", v)
	}
	sort.SliceStable(alive, func(i, j int) bool { return alive[i].id < alive[j].id })

	childCount := map[int64]int{}
	for _, t := range alive {
		if pt := byUID[t.parentUID]; pt != nil && pt != t {
			childCount[pt.uid]++
		}
	}
	p.Tasks = nil
	for _, t := range alive {
		days := mppRoundDiv(t.durationTenths, dayTenths) // 四舍五入到工作日
		if days < 0 {
			days = 0
		}
		row := Task{ID: fmt.Sprintf("%d", t.uid), Name: t.name, Level: 1, Progress: t.pct}
		if t.milestone {
			row.IsMilestone = true
		} else {
			row.Duration = days
		}
		if childCount[t.uid] > 0 { // 分组行:汇总唯一口径为子孙求和,行自带值清零
			row.Level = 0
			row.Duration = 0
			row.Progress = 0
			row.IsMilestone = false
		}
		p.Tasks = append(p.Tasks, row)
	}
	relMap := map[int64]LinkType{0: FF, 1: FS, 2: SF, 3: SS}
	for _, l := range rawLinks {
		from, to := byUID[l.pred], byUID[l.succ]
		if from == nil || to == nil {
			continue
		}
		fromID, toID := fmt.Sprintf("%d", from.uid), fmt.Sprintf("%d", to.uid)
		lt, ok := relMap[l.relType]
		if !ok {
			lt = FS
		}
		lag := mppRoundDiv(l.lagTenths, dayTenths)
		p.Links = append(p.Links, Link{From: fromID, To: toID, Type: lt, Lag: lag})
	}
	resID := map[int64]string{}
	for _, r := range resources {
		if r.name == "" {
			continue
		}
		id := fmt.Sprintf("r%d", r.uid)
		resID[r.uid] = id
		res := Resource{ID: id, Name: r.name, Type: ResWork}
		if r.label != "" { // 有材料计量单位 → 材料资源(诚实启发式)
			res.Type = ResMaterial
			res.Unit = r.label
		}
		if r.maxUnits > 0 {
			res.MaxUnits = r.maxUnits
		}
		p.Resources = append(p.Resources, res)
	}
	for _, a := range assignments {
		t, okT := byUID[a.taskUID]
		rid, okR := resID[a.resUID]
		if !okT || !okR || childCount[t.uid] > 0 { // 分配只挂叶任务(fail-closed 口径)
			continue
		}
		asg := Assignment{TaskID: fmt.Sprintf("%d", t.uid), ResourceID: rid}
		if a.units > 0 && a.units != 1 {
			u := a.units
			asg.Units = &u
		}
		p.Assignments = append(p.Assignments, asg)
	}
	p.Calendar = &Calendar{Workweek: workweek, Holidays: nil}
	return p, nil
}

// mppDefaultWeek 缺省周工作制:周一~五(文件无日历数据时)。
func mppDefaultWeek() []int { return []int{1, 2, 3, 4, 5} }

func intInSlice(v int, s []int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// mppWeekFromHours 每周工时块(7×60 字节,Day1..7=周日..周六)→ 工作日集合
// (JS getDay 口径:周日=0..周六=6)。每日:int16 flag@0(flag==1=缺省周,即
// 周一~五工作);否则 int16 段数@2,段数 0=休息。
//
// 分工（IN3-02）：本函数是 MPP 二进制的**格式解析器**——只从文件字节推
// Workweek 集合，产出经 mppBuildProject 唯一消费（构造 Calendar.Workweek，
// 节假日例外 MPP 侧不解析恒 nil）；「某日是否工作日」的规则求值一律走
// calendar.go IsWorkingDate，本函数不做、也不得做。
func mppWeekFromHours(b []byte) []int {
	def := mppDefaultWeek()
	out := []int{}
	for i := 0; i < 7; i++ {
		base := i * 60
		if base+4 > len(b) {
			break
		}
		flag := mppI16(b, base)
		periods := mppI16(b, base+2)
		if flag == 1 {
			if intInSlice(i, def) {
				out = append(out, i)
			}
			continue
		}
		if periods > 0 {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		return def
	}
	sort.Ints(out)
	return out
}

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

// ── 版本探测(CompObj:skip 28 → i32 len → 应用名 → i32 len → 格式串)────

// mppDetectFormat 返回格式串(MSProject.MPP9 等)与应用主版本号(15/16=2013+)。
func mppDetectFormat(compObj []byte) (format string, appVer int, err error) {
	if len(compObj) < 40 {
		return "", 0, fmt.Errorf("不是有效的 MPP 文件(缺 CompObj)")
	}
	pos := 28
	n := int(mppI32(compObj, pos))
	pos += 4
	if n <= 0 || pos+n > len(compObj) {
		return "", 0, fmt.Errorf("CompObj 应用名异常")
	}
	app := string(compObj[pos : pos+n-1]) // 尾部 NUL 不含
	pos += n
	if pos+4 <= len(compObj) {
		m := int(mppI32(compObj, pos))
		pos += 4
		if m > 0 && pos+m <= len(compObj) {
			format = string(compObj[pos : pos+m-1])
		}
	}
	if strings.HasPrefix(app, "Microsoft Project") {
		if seg := strings.Index(app, "."); seg > 0 {
			start := seg
			for start > 0 && app[start-1] >= '0' && app[start-1] <= '9' {
				start--
			}
			if start < seg {
				num := 0
				for i := start; i < seg; i++ {
					num = num*10 + int(app[i]-'0')
				}
				if num > 0 {
					appVer = num
				}
			}
		}
	}
	if format == "" {
		return "", appVer, fmt.Errorf("不是有效的 MPP 工程文件")
	}
	return format, appVer, nil
}

// ── CFB 容器:mscfb 深度优先遍历 + 已知布局状态机重建语义键 ─────────

// mppOpenStreams 读出解析所需流,键为语义名:"\x01CompObj"、"Props9/12/14"、
// "<工程目录>/Props"、"<工程目录>/TBknd*/<流名>"。mscfb Name 不含前导 \x01。
func mppOpenStreams(data []byte) (map[string][]byte, string, error) {
	r, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("不是有效的 MPP 文件(OLE2 打开失败):%w", err)
	}
	out := map[string][]byte{}
	projDir := ""  // 当前工程/视图目录("   19"/"   112"/"   114"/"   29")
	foundDir := "" // 首个工程目录（版本回退探测用）
	tb := ""       // 当前 TBknd* 存储
	awaitProps := false
	for {
		entry, err := r.Next()
		if err != nil {
			break // io.EOF
		}
		name := entry.Name
		if entry.FileInfo().IsDir() {
			switch {
			case name == "   19" || name == "   112" || name == "   114":
				if foundDir == "" {
					foundDir = name
				}
				projDir, tb, awaitProps = name, "", true
			case name == "   29": // 视图目录:离开工程目录子树
				tb, awaitProps = "", false
			case strings.HasPrefix(name, "TBknd"):
				tb = name
			}
			continue
		}
		key := ""
		switch {
		case tb != "":
			key = projDir + "/" + tb + "/" + name
		case awaitProps && name == "Props":
			key = projDir + "/Props"
			awaitProps = false
		case name == "Props9" || name == "Props12" || name == "Props14":
			key = name
		case name == "CompObj":
			key = "\x01CompObj"
		}
		if key == "" {
			continue
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(entry); err != nil {
			return nil, "", fmt.Errorf("MPP 流 %s 读取失败:%w", name, err)
		}
		out[key] = buf.Bytes()
	}
	return out, foundDir, nil
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
