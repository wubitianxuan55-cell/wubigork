package workcost

// ── 存量资源自动归类用例 ────────────────────────────────────────────
//
// 分类名取自**真实用户库**（1590 条成本条目的实际分布），因此这些用例同时是
// 分类口径的回归锚点：改动关键词表若让主流分类归错类，这里必红。

import (
	"strings"
	"testing"
)

func TestClassifyKindRealLibraryCategories(t *testing.T) {
	cases := []struct {
		path  string
		title string
		want  string
		note  string
	}{
		// 材料路径（真实库 TOP 分类，均应为材料）
		{"材料/安装材料/管材管件", "PE给水管 DN110", KindMaterial, "273 条，库内最大类"},
		{"材料/安装材料/电线电缆", "YJV电缆 4×25", KindMaterial, "226 条"},
		{"材料/市政绿化材料", "香樟 胸径12cm", KindMaterial, "184 条"},
		{"材料/土建材料/水泥及水泥制品", "P.O 42.5 水泥", KindMaterial, "94 条"},
		{"材料/安装材料/阀门", "闸阀 DN100", KindMaterial, "89 条"},
		{"材料/土建材料/钢材", "HRB400 螺纹钢 Φ20", KindMaterial, "76 条"},
		{"材料/土建材料/防水材料", "SBS防水卷材 4mm", KindMaterial, "45 条"},

		// 机械路径
		{"机械/土方机械", "挖掘机 1.0m³", KindMachine, ""},
		{"机械", "自卸汽车 12t", KindMachine, ""},
		// 材料路径下的机械配件：机械关键词必须命中才归机械
		{"材料/辅助材料/机械配件", "破碎锤钎杆", KindMachine, "材料路径但明确是机械件"},
		{"材料/辅助材料/耗材", "焊条 J422", KindMaterial, "材料路径、无机械词 → 材料"},

		// 人工 / 劳务
		{"人工", "综合工日", KindLabor, ""},
		{"土方", "人工清底 工日", KindLabor, ""},
		// 外委 → 独立行类（实测最新模版：水泥窑处置/外购土/监测化验标行类=外委，
		// 汇总行并入材料桶）
		{"其他/处置", "残余废水委外处置（含运）", KindOutsourced, "委外"},
		{"其他/处置", "残余一般固废委外处置（含运输）", KindOutsourced, "委外"},
		{"其他/处置", "水泥窑协同处置（污染土壤）", KindOutsourced, "外委"},
		{"其他/处置", "外购清洁土", KindOutsourced, "外购"},
		// 现场发生的技术服务仍是人工（实测产物：修复方案编制、专项设计论证）
		{"监测检测", "大气环境监测外委", KindOutsourced, "外委"},
		{"检测/实体检测", "地下水效果评估监测", KindLabor, "现场技术服务=人工"},
		{"人工", "修复方案编制及评审", KindLabor, "编制=人工"},
		{"技术服务", "深基坑专项设计及专家论证", KindLabor, "设计论证=人工"},

		// 兜底：无任何信号 → 材料
		{"", "普通硅酸盐水泥", KindMaterial, "无分类无关键词"},
		{"综合单价/土方", "机械挖一般土方", KindMachine, "综合单价类但机械词命中"},
	}
	for _, c := range cases {
		if got := ClassifyKind(c.path, c.title); got != c.want {
			t.Errorf("ClassifyKind(%q, %q) = %q, want %q（%s）", c.path, c.title, got, c.want, c.note)
		}
	}
}

// TestClassifyKindMachineWinsOnStrongSignal 反向验证：材料路径 + 明确机械信号
// 必须归机械，而非被路径前缀一票定成材料。
func TestClassifyKindMachineWinsOnStrongSignal(t *testing.T) {
	if got := ClassifyKind("材料/辅助材料/机械配件", "挖掘机斗齿"); got != KindMachine {
		t.Fatalf("材料路径下的明确机械件应归机械，得到 %q", got)
	}
}

// TestClassifyKindTopPathAuthoritative 顶层路径优先级：机械路径即使名称含
// 「材料」字样也归机械（路径是人工维护的强意图）。
func TestClassifyKindTopPathAuthoritative(t *testing.T) {
	if got := ClassifyKind("机械/土方机械", "斗齿材料"); got != KindMachine {
		t.Fatalf("机械路径应权威，得到 %q", got)
	}
}

func TestTopSegment(t *testing.T) {
	cases := map[string]string{
		"材料/安装材料/管材管件": "材料",
		"材料":           "材料",
		"  机械 / 土方 ":   "机械",
		"":             "",
	}
	for in, want := range cases {
		if got := topSegment(in); got != want {
			t.Errorf("topSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCountKinds(t *testing.T) {
	list := []SeedCandidate{
		{Kind: KindLabor}, {Kind: KindMaterial}, {Kind: KindMaterial},
		{Kind: KindMachine}, {Kind: KindOutsourced}, {},
	}
	c := CountKinds(list)
	if c.Labor != 1 || c.Material != 3 || c.Machine != 1 || c.Outsourced != 1 {
		t.Fatalf("归类统计错误: %+v", c)
	}
	if c.Total() != 6 {
		t.Fatalf("合计应为 6，得到 %d", c.Total())
	}
}

// TestComposeOutsourcedRollsIntoMaterial 外委口径钉子：外委是独立行类，但三费
// 归集并入材料桶——对齐实测最新模版汇总行「材料 = SUMIF(材料) + SUMIF(外委)」。
func TestComposeOutsourcedRollsIntoMaterial(t *testing.T) {
	lines := []ComposeLine{
		{Kind: KindLabor, Title: "普通工", Unit: "工日", Quantity: 1, Price: 300},
		{Kind: KindMaterial, Title: "C20商品混凝土", Unit: "m³", Quantity: 1, Price: 345},
		{Kind: KindOutsourced, Title: "水泥窑协同处置", Unit: "t", Quantity: 1, Price: 190},
	}
	c := ComposeUnitPrice(lines)
	closeTo(t, "人工费", c.LaborFee, 300)
	closeTo(t, "材料费（含外委 190）", c.MaterialFee, 535)
	closeTo(t, "外委小计（拆分展示，不重复计入）", c.OutsourcedFee, 190)
	closeTo(t, "综合单价", c.CompositePrice, 835)
	if len(c.Warnings) != 0 {
		t.Errorf("外委是已知类别，不应告警：%v", c.Warnings)
	}
	// 反向：外委若未并入材料桶，材料费会是 345 —— 必须能区分这两种口径。
	if c.MaterialFee == 345 {
		t.Fatal("外委未并入材料桶——与实测模版汇总行口径不符")
	}
}

// TestSeedCandidateAsResource 候选转资源：基准价与现行价都取条目价，
// 身份字段清洗空格。
func TestSeedCandidateAsResource(t *testing.T) {
	c := SeedCandidate{
		SourceName: "旺平-道渣-元-m3", Title: " 道渣 ", Spec: " 级配碎石 ", Unit: " m³ ",
		Price: 78, CategoryPath: "材料/土建材料/砖瓦灰砂石", Source: "宝兴县旺平矿业地块成本测算表最终版",
	}
	r := c.AsResource()
	if r.Kind != KindMaterial {
		t.Errorf("Kind = %q, want 材料", r.Kind)
	}
	if r.Title != "道渣" || r.Spec != "级配碎石" || r.Unit != "m³" {
		t.Errorf("身份字段应去空格，得到 %q/%q/%q", r.Title, r.Spec, r.Unit)
	}
	if r.BasePrice != 78 || r.CurrentPrice != 78 {
		t.Errorf("基准价与现行价都应为 78，得到 %v/%v", r.BasePrice, r.CurrentPrice)
	}
	if r.EffectivePrice() != 78 {
		t.Errorf("核算取用价应为 78，得到 %v", r.EffectivePrice())
	}
	if r.Status != "现行" {
		t.Errorf("Status = %q, want 现行", r.Status)
	}
}

// TestMnemonicOf 助记码提取（对齐实测产物 EXC/ALU/C20/HDPE 风格）。
func TestMnemonicOf(t *testing.T) {
	cases := map[string]string{
		"C20商品混凝土":     "C20",
		"HDPE防渗膜1.5mm": "HDPE",
		"EXC挖掘机":       "EXC",
		"YJV电缆 4×25":   "YJV",
		"PE给水管 DN110":  "PE",
		"P.O 42.5 水泥":  "PO42",
		"普通硅酸盐水泥":      "", // 纯中文提取不到
		"12 号槽钢":       "", // 首位纯数字片段不足两字符 → 空 → 走类别前缀
		"a":            "", // 单字符不足
	}
	for in, want := range cases {
		if got := mnemonicOf(in, KindMaterial); got != want {
			t.Errorf("mnemonicOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNextResourceCodeMnemonicAndCollision 编码分配：助记码优先、撞码加后缀、
// 纯中文回退类别前缀 + 序号。
func TestNextResourceCodeMnemonicAndCollision(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	a, err := s.SaveResource(Resource{Kind: KindMachine, Title: "挖掘机 1.0m³", Unit: "台班", CurrentPrice: 1200})
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if a.Code == "" {
		t.Fatalf("编码不应为空")
	}
	t.Logf("挖掘机编码 = %q", a.Code)

	// 同名不同单位 → 新资源，编码不得与 a 相撞。
	b, err := s.SaveResource(Resource{Kind: KindMachine, Title: "挖掘机 1.0m³", Unit: "台时", CurrentPrice: 150})
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if b.Code == a.Code {
		t.Fatalf("编码撞码：%q 出现两次", b.Code)
	}

	// 纯中文名回退类别前缀。
	c, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "普通硅酸盐水泥", Unit: "t", CurrentPrice: 420})
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if c.Code == "P001" {
		t.Logf("纯中文名编码 = %q（前缀+三位序号）", c.Code)
	} else if c.Code == "" || len(c.Code) < 2 {
		t.Fatalf("纯中文名须分配到可读编码，得到 %q", c.Code)
	}
	t.Logf("水泥编码 = %q", c.Code)
}

// ── 资源化规则（ToSeedCandidate / PreviewSeed / ApplySeed）────────────

// TestToSeedCandidateSkipsComposite 综合单价条目默认不降级为资源。
func TestToSeedCandidateSkipsComposite(t *testing.T) {
	in := SeedInput{Name: "n", Title: "机械挖一般土方", Price: 12.5, CategoryPath: "综合单价/土方"}
	if _, ok, reason := ToSeedCandidate(in, DefaultSeedOptions()); ok {
		t.Fatal("综合单价条目默认应被跳过，不降级为资源")
	} else if reason == "" {
		t.Fatal("跳过必须给出原因")
	}
	// 显式开启后纳入。
	if _, ok, _ := ToSeedCandidate(in, SeedOptions{IncludeComposite: true}); !ok {
		t.Fatal("IncludeComposite=true 时应纳入")
	}
}

// TestToSeedCandidateLedgerSpecMovedToNote 实测库里 spec 被用作来源台账，
// 应识别为口径说明并入 Note，而非当成技术规格污染资源身份。
func TestToSeedCandidateLedgerSpecMovedToNote(t *testing.T) {
	in := SeedInput{
		Name:         "旺平-道渣-元-m3",
		Title:        "旺平-道渣-元-m3",
		Spec:         "宝兴县旺平矿业地块成本测算表最终版",
		Unit:         "m³",
		Price:        78,
		CategoryPath: "材料/土建材料/砖瓦灰砂石",
		Body:         "便道基层改级配碎石",
	}
	c, ok, _ := ToSeedCandidate(in, DefaultSeedOptions())
	if !ok {
		t.Fatal("应纳入")
	}
	if c.Spec != "" {
		t.Errorf("台账串不应留在 Spec，得到 %q", c.Spec)
	}
	if c.Note == "" || !strings.Contains(c.Note, "宝兴县旺平矿业地块") {
		t.Errorf("台账串应并入 Note，得到 %q", c.Note)
	}
	if !strings.Contains(c.Note, "便道基层改级配碎石") {
		t.Errorf("原 Body 不应丢失，得到 %q", c.Note)
	}
}

// TestToSeedCandidateRealSpecKept 真正的技术规格必须保留（不能误判为台账）。
func TestToSeedCandidateRealSpecKept(t *testing.T) {
	cases := []string{"DN110", "4×25", "Φ20", "厚5cm", "1.5mm双光面", "400g/m²"}
	for _, spec := range cases {
		c, ok, _ := ToSeedCandidate(SeedInput{
			Name: "x", Title: "某材料", Spec: spec, Price: 10, CategoryPath: "材料/安装材料/管材管件",
		}, DefaultSeedOptions())
		if !ok {
			t.Fatalf("spec=%q 应纳入", spec)
		}
		if c.Spec != spec {
			t.Errorf("技术规格应保留：in=%q out=%q", spec, c.Spec)
		}
	}
}

// TestToSeedCandidateSkipReasons 跳过原因分类正确且可统计。
// 注意：Title 为空时按设计回退 Name（成本库实测 name/title 相同），
// 故「无名称」用例必须 Title 与 Name 都空才触发跳过。
func TestToSeedCandidateSkipReasons(t *testing.T) {
	inputs := []SeedInput{
		{Name: "a", Title: "正常材料", Price: 10, CategoryPath: "材料/土建材料/钢材"},
		{Name: "b", Title: "零价材料", Price: 0, CategoryPath: "材料/土建材料/钢材"},
		{Name: "c", Title: "", Price: 10, CategoryPath: "材料/土建材料/钢材"}, // Title 空 → 回退 Name "c"
		{Name: "", Title: "", Price: 10, CategoryPath: "材料/土建材料/钢材"},  // 两者都空 → 跳过
		{Name: "d", Title: "综合单价项", Price: 10, CategoryPath: "综合单价/土方"},
	}
	p := PreviewSeed(inputs, DefaultSeedOptions())
	if p.Total != 5 {
		t.Errorf("Total = %d, want 5", p.Total)
	}
	if len(p.Candidates) != 2 {
		t.Errorf("应纳入 2 条（正常材料 + Name 回退），得到 %d", len(p.Candidates))
	}
	if p.Skipped["价格非正"] != 1 || p.Skipped["无名称"] != 1 {
		t.Errorf("跳过原因统计错误: %v", p.Skipped)
	}
	if p.Counts.Total() != 2 || p.Counts.Material != 2 {
		t.Errorf("归类统计错误: %+v", p.Counts)
	}
}

// TestApplySeedIdempotent 资源化入库幂等：重复应用不制造重复资源，
// 第二次全部计入 Updated。
func TestApplySeedIdempotent(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	cands := []SeedCandidate{
		{Title: "PE给水管 DN110", Spec: "DN110", Unit: "m", Price: 45, Kind: KindMaterial, CategoryPath: "材料/安装材料/管材管件"},
		{Title: "挖掘机 1.0m³", Spec: "1.0m³", Unit: "台班", Price: 1200, Kind: KindMachine, CategoryPath: "机械/土方机械"},
		{Title: "综合工日", Unit: "工日", Price: 300, Kind: KindLabor, CategoryPath: "人工"},
	}
	first := s.ApplySeed(cands)
	if first.Created != 3 || first.Updated != 0 || len(first.Errors) != 0 {
		t.Fatalf("首次入库应新建 3 条：%+v", first)
	}
	if first.Counts.Labor != 1 || first.Counts.Material != 1 || first.Counts.Machine != 1 {
		t.Errorf("归类统计错误: %+v", first.Counts)
	}

	second := s.ApplySeed(cands)
	if second.Created != 0 || second.Updated != 3 {
		t.Fatalf("重复应用应全部更新、零新建：%+v", second)
	}
	list, err := s.ListResources("", "")
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("重复应用后资源数应为 3，得到 %d", len(list))
	}
}

// TestApplySeedPartialFailureRecordsError 单条脏数据不中断整批。
func TestApplySeedPartialFailureRecordsError(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	cands := []SeedCandidate{
		{Title: "", Unit: "m", Price: 10, Kind: KindMaterial}, // 无名称 → 失败
		{Title: "有效材料", Unit: "m", Price: 10, Kind: KindMaterial},
	}
	res := s.ApplySeed(cands)
	if len(res.Errors) != 1 {
		t.Fatalf("应记录 1 条错误，得到 %v", res.Errors)
	}
	if res.Created != 1 {
		t.Fatalf("有效条目应入库，created=%d", res.Created)
	}
}
