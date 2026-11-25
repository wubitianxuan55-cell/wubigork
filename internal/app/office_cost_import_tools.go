package app

// office_cost_import_tools.go — 模型读表导入的工具面（cost_library 的 import_* 动作）。
//
// 2026-11-25 用户拍板：造价资料格式不固定，导入由模型读资料、整理后整批提交；五表
// 机械解析（import_project）只留作标准模版的快速通道。暂存、校验、勾稽、提交的核心
// 在 workcost/importstage.go，这里只管会话保管、参数与回执。
//
// 会话只在内存里（两小时没动静自动丢弃）：暂存本来就没落库，丢了模型重新
// import_add 即可。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// clSrcSchema import_* 各行的来源位置（回读核对的锚点）在说明里的写法。
const clSrcSchema = `{"type":"object","description":"这一行在资料里的位置（回读核对的锚点，强烈建议每行都带）：sheet=工作表名（csv 不给），row=行号（从 1 起，与 Excel 一致）","properties":{"sheet":{"type":"string"},"row":{"type":"integer"}}}`

// ── 会话保管（内存） ─────────────────────────────────────────────────

const costImportTTL = 2 * time.Hour

var costImportSessions = struct {
	sync.Mutex
	m map[string]*costImportEntry
}{m: map[string]*costImportEntry{}}

type costImportEntry struct {
	imp     *workcost.ImportSession
	touched time.Time
}

func costImportPut(imp *workcost.ImportSession) {
	costImportSessions.Lock()
	defer costImportSessions.Unlock()
	now := time.Now()
	for id, e := range costImportSessions.m {
		if now.Sub(e.touched) > costImportTTL {
			e.imp.Close()
			delete(costImportSessions.m, id)
		}
	}
	costImportSessions.m[imp.ID] = &costImportEntry{imp: imp, touched: now}
}

// costImportGet 取会话；不存在或已过期返回 false。
func costImportGet(id string) (*workcost.ImportSession, bool) {
	costImportSessions.Lock()
	defer costImportSessions.Unlock()
	e, ok := costImportSessions.m[id]
	if !ok {
		return nil, false
	}
	if time.Since(e.touched) > costImportTTL {
		e.imp.Close()
		delete(costImportSessions.m, id)
		return nil, false
	}
	e.touched = time.Now()
	return e.imp, true
}

func costImportDrop(id string) {
	costImportSessions.Lock()
	defer costImportSessions.Unlock()
	if e, ok := costImportSessions.m[id]; ok {
		e.imp.Close()
		delete(costImportSessions.m, id)
	}
}

// init 给确认卡登记「提交导入会写什么」的取数口（暂存区在本包的会话里，控制层拿不到）。
func init() {
	control.SetCostImportSummaryHook(func(importID string) (string, int, int, int, bool) {
		imp, ok := costImportGet(importID)
		if !ok {
			return "", 0, 0, 0, false
		}
		r, q, b := imp.Staged()
		return imp.Project.Name, r, q, b, true
	})
}

// ── 动作 ─────────────────────────────────────────────────────────────

var clImportKindLabel = map[string]string{"resource": "工料机", "quota": "定额", "bill": "清单"}

// importOpen 开一次导入：登记资料路径、项目口径与控制数。口径不过规矩当场拒。
func (t costLibraryTool) importOpen(s *workcost.Store, p costLibraryArgs) (string, error) {
	project := workcost.ImportProject{}
	if p.Project != nil {
		project = *p.Project
	}
	controls := workcost.ImportControls{}
	if p.Controls != nil {
		controls = *p.Controls
	}
	abs := ""
	if src := strings.TrimSpace(p.Source); src != "" {
		got, _ := resolvePreviewPath(src)
		if got == "" {
			return "", fmt.Errorf("读不到资料 %s（不存在，或在工作区之外）", src)
		}
		if info, err := os.Stat(got); err != nil || info.IsDir() {
			return "", fmt.Errorf("读不到资料 %s", src)
		}
		abs = got
	}
	imp, err := s.OpenImport(abs, project, controls)
	if err != nil {
		return "", err
	}
	costImportPut(imp)
	var out strings.Builder
	out.WriteString("✅ 已开导入：" + imp.ID + "\n")
	fmt.Fprintf(&out, "**项目**: %s", clDim(project.Name, "（未给项目——只能提交工料机与定额）"))
	fmt.Fprintf(&out, "｜专业 %s｜地区 %s｜计价期 %s\n", clDim(imp.Specialty, "未标注"), clDim(imp.Region, "未标注"), clDim(imp.Period, "未标注"))
	if abs != "" {
		fmt.Fprintf(&out, "**资料**: %s\n", filepath.ToSlash(abs))
		switch strings.ToLower(filepath.Ext(abs)) {
		case ".xlsx", ".xlsm", ".csv":
			out.WriteString("回读核对可用：行里带 src（工作表名 + 行号）的，程序会回到资料里逐个核对你提交的数与文字。\n")
		default:
			out.WriteString("这种格式不支持回读核对——行里带了 src 也按「未核对」计，数字没有第二道把关，转述时向用户说明。\n")
		}
	} else {
		out.WriteString("没登记资料路径：无法回读核对，行里不必带 src。\n")
	}
	if p.Controls == nil {
		out.WriteString("**控制数**: 未登记（资料上有行数、合计的话，重新 import_open 带上 controls，勾稽时能对出漏行漏项）。\n")
	}
	out.WriteString("\n下一步：读资料、整理成工料机 / 定额 / 清单行，action=import_add 分批提交（先工料机、再定额、后清单；每批 ≤200 行，每行带 src）。")
	// 行字段契约不常驻 schema（省每轮上下文），在这个流程必经节点随身发放。
	out.WriteString("\n\n" + clImportRowsContract)
	return out.String(), nil
}

// importAdd 分批入暂存：合规的进暂存区，不合规的退回并说明原因。库里一行不动。
func (t costLibraryTool) importAdd(s *workcost.Store, p costLibraryArgs) (string, error) {
	imp, err := t.importSession(p.ImportID)
	if err != nil {
		return "", err
	}
	if len(p.Resources)+len(p.Quotas)+len(p.Bills) == 0 {
		return "", fmt.Errorf("import_add 需要 resources / quotas / bills 至少一组（行字段契约见 import_open 回执，或 action=help）")
	}
	var results []workcost.ImportAddResult
	add := func(kind string, n int, run func() (workcost.ImportAddResult, error)) error {
		if n == 0 {
			return nil
		}
		if n > workcost.ImportMaxPerBatch {
			return fmt.Errorf("%s 一批最多 %d 行（本次 %d），请分批提交", clImportKindLabel[kind], workcost.ImportMaxPerBatch, n)
		}
		r, err := run()
		if err != nil {
			return err
		}
		results = append(results, r)
		return nil
	}
	if err := add("resource", len(p.Resources), func() (workcost.ImportAddResult, error) { return imp.AddResources(s, p.Resources) }); err != nil {
		return "", err
	}
	if err := add("quota", len(p.Quotas), func() (workcost.ImportAddResult, error) { return imp.AddQuotas(s, p.Quotas) }); err != nil {
		return "", err
	}
	if err := add("bill", len(p.Bills), func() (workcost.ImportAddResult, error) { return imp.AddBills(s, p.Bills) }); err != nil {
		return "", err
	}

	rejected := 0
	var out strings.Builder
	fmt.Fprintf(&out, "🔍 暂存（未写库）：导入 %s\n", imp.ID)
	for _, r := range results {
		rejected += len(r.Rejected)
		fmt.Fprintf(&out, "**本批%s**: 提交 %d 行 → 入暂存 %d（其中覆盖 %d）｜退回 %d\n",
			clImportKindLabel[r.Kind], r.Accepted+r.Replaced+len(r.Rejected), r.Accepted+r.Replaced, r.Replaced, len(r.Rejected))
	}
	for _, r := range results {
		if len(r.Rejected) == 0 {
			continue
		}
		fmt.Fprintf(&out, "**退回的%s %d 行**（改好只重发这几行，其余已在暂存区；列前 %d 行）:\n",
			clImportKindLabel[r.Kind], len(r.Rejected), clCap(len(r.Rejected)))
		for i, rj := range r.Rejected {
			if i >= clCap(len(r.Rejected)) {
				break
			}
			name := rj.Code
			if name == "" {
				name = rj.Title
			}
			fmt.Fprintf(&out, "- 第 %d 行 %s：%s\n", rj.Index, name, strings.Join(rj.Reasons, "；"))
		}
	}
	r, q, b := imp.Staged()
	fmt.Fprintf(&out, "**暂存合计**: 工料机 %d｜定额 %d｜清单 %d\n", r, q, b)
	if rejected > 0 {
		out.WriteString("库里一行未动。退回的行改好后重新 import_add（同码覆盖）；全部整理完 action=import_check 勾稽。")
	} else {
		out.WriteString("库里一行未动。资料整理完后 action=import_check 勾稽。")
	}
	return out.String(), nil
}

// importSession 取导入会话。
func (t costLibraryTool) importSession(id string) (*workcost.ImportSession, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("需要 import_id（先 action=import_open 开一次导入）")
	}
	imp, ok := costImportGet(id)
	if !ok {
		return nil, fmt.Errorf("找不到导入会话 %s（已过期或没开过）——重新 import_open 后再 import_add", id)
	}
	return imp, nil
}

// importCheck 勾稽（只读）。
func (t costLibraryTool) importCheck(s *workcost.Store, p costLibraryArgs) (string, error) {
	imp, err := t.importSession(p.ImportID)
	if err != nil {
		return "", err
	}
	report, err := clImportCheckReceipt(s, imp)
	if err != nil {
		return "", err
	}
	return report, nil
}

// clImportCheckReceipt 勾稽报告（import_check 与 import_commit 的演练共用）。
func clImportCheckReceipt(s *workcost.Store, imp *workcost.ImportSession) (string, error) {
	rep, err := imp.Check(s)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	fmt.Fprintf(&out, "🔍 勾稽（只读）：导入 %s\n", imp.ID)
	r, q, b := imp.Staged()
	fmt.Fprintf(&out, "**暂存**: 工料机 %d｜定额 %d｜清单 %d\n", r, q, b)
	if len(rep.Controls) == 0 {
		out.WriteString("**控制数**: 未登记（资料上的行数与合计——没有它对不出漏行漏项）\n")
	} else {
		out.WriteString("**控制数**（资料 / 暂存）:\n")
		for _, c := range rep.Controls {
			mark := "✓"
			if !c.OK {
				mark = "✗"
			}
			fmt.Fprintf(&out, "- %s：%s / %s %s\n", c.Label, c.Declared, c.Staged, mark)
		}
	}
	if rep.NoSource {
		fmt.Fprintf(&out, "**回读核对**: 这次导入没有可回读的资料（没登记路径或格式不支持）——%d 行全部未核对，数字没有第二道把关。\n", rep.UnverifiedRows)
	} else {
		fmt.Fprintf(&out, "**回读核对**: 已核对 %d 行 %d 个字段｜不符 %d 处｜未带来源位置的 %d 行｜模型换算 %d 个字段\n",
			rep.ReadbackRows, rep.ReadbackFields, len(rep.ReadbackIssues), rep.UnverifiedRows, rep.DerivedFields)
	}
	for i, is := range rep.ReadbackIssues {
		if i >= clCap(len(rep.ReadbackIssues)) {
			fmt.Fprintf(&out, "- …共 %d 处\n", len(rep.ReadbackIssues))
			break
		}
		gave := is.Gave
		if gave == "" {
			gave = "（无）"
		}
		fmt.Fprintf(&out, "- %s（%s）%s：提交 %s——%s\n", is.Row, is.Where, is.Field, gave, is.Found)
	}
	if len(rep.Dangling) > 0 {
		fmt.Fprintf(&out, "**引用悬空 %d 处**:\n", len(rep.Dangling))
		for i, d := range rep.Dangling {
			if i >= clCap(len(rep.Dangling)) {
				break
			}
			fmt.Fprintf(&out, "- %s\n", d)
		}
	}
	fmt.Fprintf(&out, "**单价对照**: 可比 %d 条｜偏差超 1%% 的 %d 条\n", rep.PriceChecked, len(rep.PriceIssues))
	for i, pc := range rep.PriceIssues {
		if i >= clCap(len(rep.PriceIssues)) {
			fmt.Fprintf(&out, "- …共 %d 条\n", len(rep.PriceIssues))
			break
		}
		fmt.Fprintf(&out, "- [%s] %s：原表 %s / 现算 %s（%+.1f%%）\n", pc.Code, pc.Title, clMoney(pc.ListPrice), clMoney(pc.Computed), pc.DeviationPct)
	}
	switch {
	case rep.Problems > 0:
		fmt.Fprintf(&out, "**结论**: 有 %d 处要处理——回到资料逐条核对，改完用 import_add 覆盖对应行，再 import_check。如实转述给用户，不要带着问题提交。\n", rep.Problems)
	case rep.NoSource:
		fmt.Fprintf(&out, "**结论**: 没有要处理的问题；但这次导入没有可回读的资料，%d 行的数字都没有第二道把关——向用户说明后再提交。\n", rep.UnverifiedRows)
	case rep.UnverifiedRows > 0:
		fmt.Fprintf(&out, "**结论**: 已核对的部分全部对上；还有 %d 行没带来源位置，这些行的数字没有第二道把关——建议补上 src 重发，或向用户说明后再提交。\n", rep.UnverifiedRows)
	default:
		out.WriteString("**结论**: 全部对上，可以提交（action=import_commit，去掉 dry_run 正式提交会弹确认）。\n")
	}
	return out.String(), nil
}

// importCommit 提交：演练只出报告；正式写库（写前再过一遍规矩，有一行不合规就整批不写）。
func (t costLibraryTool) importCommit(s *workcost.Store, p costLibraryArgs) (string, error) {
	imp, err := t.importSession(p.ImportID)
	if err != nil {
		return "", err
	}
	report, err := clImportCheckReceipt(s, imp)
	if err != nil {
		return "", err
	}
	r, q, b := imp.Staged()
	if p.DryRun {
		var out strings.Builder
		fmt.Fprintf(&out, "🔍 演练（未写库）：提交导入 %s\n", imp.ID)
		fmt.Fprintf(&out, "**将写入**: %s｜工料机 %d｜定额 %d｜清单 %d\n", clDim(imp.Project.Name, "（无项目）"), r, q, b)
		out.WriteString(report)
		return out.String(), nil
	}
	res, err := imp.Commit(s)
	if err != nil {
		return "", err
	}
	costImportDrop(imp.ID)
	var out strings.Builder
	fmt.Fprintf(&out, "✅ 已完成：提交导入 %s\n", imp.ID)
	if res.ProjectName != "" {
		fmt.Fprintf(&out, "**项目**: #%d %s（专业 %s｜地区 %s｜计价期 %s）\n", res.ProjectID, res.ProjectName,
			clDim(imp.Project.Specialty, "未标注"), clDim(imp.Project.Region, "未标注"), clDim(imp.Period, "未标注"))
	}
	fmt.Fprintf(&out, "**写入**: 工料机 新建 %d / 更新 %d｜定额 新建 %d / 更新 %d｜清单 %d 行｜价格事实 %d 条\n",
		res.ResourcesNew, res.ResourcesUpd, res.QuotasNew, res.QuotasUpd, res.Bills, res.PriceFacts)
	if len(res.RegionMerged) > 0 {
		// 铜陵反馈 #18：四要素同、仅地区缺失/不同的既有行并入原行（更新，不消歧）。
		fmt.Fprintf(&out, "**地区并入**: %d 条工料机按身份四要素并入既有行（更新原行地区，未新建）\n", len(res.RegionMerged))
		for i, m := range res.RegionMerged {
			if i >= clCap(len(res.RegionMerged)) {
				fmt.Fprintf(&out, "- …共 %d 条\n", len(res.RegionMerged))
				break
			}
			fmt.Fprintf(&out, "- %s\n", m)
		}
	}
	if n := len(res.Errors); n > 0 {
		fmt.Fprintf(&out, "**未写成 %d 行**（其余已写；修掉后重新 import_open → import_add 这几行 → import_commit，已写的行按更新处理不会重复）:\n", n)
		for i, e := range res.Errors {
			if i >= clCap(n) {
				break
			}
			fmt.Fprintf(&out, "- %s\n", e)
		}
	}
	if res.ProjectID > 0 {
		fmt.Fprintf(&out, "\n下一步：cost_audit project_id=%d 跑机检；造价数据库板块切回即见。", res.ProjectID)
	} else {
		out.WriteString("\n下一步：造价数据库板块切回即见。")
	}
	return out.String(), nil
}

// importDiscard 丢弃一次导入（暂存区本来就没落库）。
func (t costLibraryTool) importDiscard(_ *workcost.Store, p costLibraryArgs) (string, error) {
	imp, err := t.importSession(p.ImportID)
	if err != nil {
		return "", err
	}
	r, q, b := imp.Staged()
	costImportDrop(imp.ID)
	return fmt.Sprintf("✅ 已完成：丢弃导入 %s（暂存的 工料机 %d｜定额 %d｜清单 %d 一并丢弃；库里本来就没写过）", imp.ID, r, q, b), nil
}
