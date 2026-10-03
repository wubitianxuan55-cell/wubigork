package schedule

// mpp_layout.go — MPP 版本布局表（批 27 IN3-03 文件拆分，自 mpp.go 原位搬移，
// 零逻辑改动）：mppVersion/mppAsgMode 枚举 + mppLayout 结构 + MPP9/12/14
// (含 Project 2013+ 变体)的 (major, appVer) 字段偏移表 + mppLayoutFor 选表。
// 字节语义与流布局说明见 mpp.go 头注；低层块解析见 mpp_rows.go。
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
