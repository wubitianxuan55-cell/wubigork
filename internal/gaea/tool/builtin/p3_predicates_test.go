package builtin

// P3 四类验收谓词 harness（2026-11-10 众神殿 pan_1794234593182 裁决 action-6）：
// 非文档工具按 tool.PredicateClasses 四类定义统一验收口径——每类金标例（通过）
// + 已知缺陷例（正确失败）。类缺 harness = TestP3PredicateClassesCovered 红；
// 类内代表工具缺 RegistryMap 行 = TestP3PredicateClassToolsRegistered 红
// （与 P0 注册表地图对齐）。
//
// 办公四类写（office_generate/docx_apply/docx_accept/pptx_apply/xlsx_apply）的
// 引用级验收已由 app 包 office/verify 测试族覆盖（本文件不重复），此处钉
// 非文档代表场景：造价/知识/记忆（读取检索）、进度计划（领域计算+交付写）、
// 待办（任务编排）；dag_plan/task 六闸刀断言见 internal/gaea/dag 与
// internal/app 的 gaea_dag_test（写路径/只读作用域/节点数闸）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// p3Harness 类 id → harness。PredicateClasses 新增类而本表缺项 = 守卫红。
var p3Harness = map[string]func(t *testing.T){
	"read-retrieval":   p3ReadRetrieval,
	"domain-compute":   p3DomainCompute,
	"deliver-write":    p3DeliverWrite,
	"task-orchestrate": p3TaskOrchestrate,
}

func TestP3PredicateClassesCovered(t *testing.T) {
	for id, cls := range tool.PredicateClasses {
		fn, ok := p3Harness[id]
		if !ok {
			t.Fatalf("P3 harness 缺类 %s（%s）——新增谓词类须同步 harness", id, cls.Name)
		}
		t.Run(cls.Name, fn)
	}
}

func TestP3PredicateClassToolsRegistered(t *testing.T) {
	for id, cls := range tool.PredicateClasses {
		for _, name := range cls.Tools {
			if _, ok := tool.RegistryMap[name]; !ok {
				t.Fatalf("类 %s 代表工具 %s 不在 RegistryMap（P0/P3 对齐漂移）", id, name)
			}
		}
	}
}

func p3Args(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// p3ReadRetrieval 读取检索类：知识/记忆/造价三库——命中与库实况一致；未命中
// 如实报零不编造（谓词=检索精度与诚实缺省）。
func p3ReadRetrieval(t *testing.T) {
	// ── 金标例①：知识库沉淀→检索命中 ──
	kst, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(kst)
	t.Cleanup(knowledge.ResetForTest)
	ka := knowledgeAdd{}
	if out, err := ka.Execute(context.Background(), p3Args(t, map[string]any{
		"title": "桩基施工经验", "category": "经验总结",
		"body": "HP300 液压振动锤在砂层进尺 8 米/台班。",
	})); err != nil || !strings.Contains(out, "已保存") {
		t.Fatalf("知识沉淀: %q %v", out, err)
	}
	ks := knowledgeSearch{}
	if out, err := ks.Execute(context.Background(), p3Args(t, map[string]any{"query": "桩基施工经验"})); err != nil || !strings.Contains(out, "桩基施工经验") {
		t.Fatalf("知识检索应命中库实况: %q %v", out, err)
	}
	// ── 缺陷例①：非法沉淀正确失败（fail-closed；检索为模糊匹配语义，
	// 「未命中如实」以报错为否面、不报错为诚实缺省——不冒充精确率）──
	if _, err := ka.Execute(context.Background(), p3Args(t, map[string]any{
		"title": "", "category": "经验总结", "body": "无标题条目",
	})); err == nil {
		t.Fatal("空标题沉淀应正确失败")
	}

	// ── 金标例②：记忆库沉淀→检索命中（temp UserDir，不碰真实用户库；
	// Name 走 ASCII slug——记忆 Save 校验不接受纯 CJK 名）──
	opts := memory.Options{UserDir: t.TempDir(), CWD: t.TempDir()}
	set := memory.Load(opts)
	idxMem := memory.Memory{Name: "client-xlsx-preference", Type: memory.TypeProject, Kind: memory.KindSemantic,
		Description: "客户偏好", Body: "报表交付偏好 XLSX 格式"}
	if _, err := set.Store.Save(idxMem); err != nil {
		t.Fatal(err)
	}
	idx := memory.Load(opts).Search
	if idx == nil {
		t.Fatal("索引为空")
	}
	ctx := memory.WithSpace(context.Background(), spaces.SpaceWork)
	ctx = memory.WithSessionSaver(ctx, stubIndexSource{idx: idx, space: spaces.SpaceWork})
	if out, err := (memorySearch{}).Execute(ctx, p3Args(t, map[string]any{"query": "客户偏好"})); err != nil || !strings.Contains(out, "客户偏好") {
		t.Fatalf("记忆检索应命中（名称+描述形态）: %q %v", out, err)
	}
	// ── 缺陷例②：缺 query 正确失败（检索为模糊匹配语义——小语料下
	// 「无关查询零命中」不成立，诚实缺省以参数闸 fail-closed 为否面）──
	if _, err := (memorySearch{}).Execute(ctx, json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("缺 query 应正确失败: %v", err)
	}

	// ── 金标例③：造价库沉淀→检索闭环（temp db，不碰真实用户库）──
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	SetCostStoreForTest(cost.Open(gdb))
	t.Cleanup(func() { SetCostStoreForTest(nil) })
	cs := costSave{}
	if out, err := cs.Execute(context.Background(), p3Args(t, map[string]any{
		"title": "HP300 高频液压振动锤", "category": "机械", "unit": "台班",
		"price": 3200, "spec": "300kW", "source": "市场询价",
	})); err != nil || !strings.Contains(out, "3200") {
		t.Fatalf("造价沉淀: %q %v", out, err)
	}
	if out, err := (costSearch{}).Execute(context.Background(), p3Args(t, map[string]any{"query": "HP300 振动锤"})); err != nil || !strings.Contains(out, "3200") {
		t.Fatalf("造价检索应命中库实况: %q %v", out, err)
	}
	// ── 缺陷例③：无效参数 JSON 正确失败（模糊检索语义同上，参数闸兜底）──
	if _, err := (costSearch{}).Execute(context.Background(), json.RawMessage(`{bad json`)); err == nil || !strings.Contains(err.Error(), "参数无效") {
		t.Fatalf("无效参数应正确失败: %v", err)
	}
}

// p3DomainCompute 领域计算类：进度计划分析可复算；约束违反（环）fail-closed。
func p3DomainCompute(t *testing.T) {
	dir := t.TempDir()
	// ── 金标例：合法项目 apply→analyze 复算一致（工期=关键路径 10）──
	schedApply(t, dir, `{"project":{"name":"办公楼","startDate":"2026-09-01",
		"calendar":{"workweek":[1,2,3,4,5],"holidays":[]},
		"tasks":[{"id":"a","name":"场地","level":1,"duration":10,"progress":0},
		         {"id":"m","name":"竣工","level":1,"duration":0,"progress":0,"isMilestone":true}],
		"links":[{"from":"a","to":"m","type":"FS","lag":0}]}}`)
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !strings.Contains(ana, `"duration":10`) || !strings.Contains(ana, `"critical":["场地","竣工"]`) {
		t.Fatalf("分析结果应可复算（关键路径工期 10、关键链=场地→竣工）: %s", ana)
	}
	// ── 缺陷例：环依赖 → fail-closed 正确失败 ──
	tool := scheduleApply{workDir: dir, roots: []string{dir}}
	_, err = tool.Execute(context.Background(), json.RawMessage(`{"ops":[{"type":"set_links","toId":"a","links":[{"from":"m"}]}]}`))
	if err == nil || !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("环应 fail-closed: %v", err)
	}
}

// p3DeliverWrite 交付写类：声明路径=实际落点；越界正确失败（办公四类写引用
// 级验收在 app 包 office/verify 测试族）。
func p3DeliverWrite(t *testing.T) {
	dir := t.TempDir()
	// ── 金标例：write_file 声明路径落点逐字节一致 ──
	wf := writeFile{workDir: dir, roots: []string{dir}}
	if out, err := wf.Execute(context.Background(), p3Args(t, map[string]any{
		"path": "交付/说明.txt", "content": "正文第一行。",
	})); err != nil {
		t.Fatalf("write_file: %v %q", err, out)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "交付", "说明.txt"))
	if err != nil || string(raw) != "正文第一行。" {
		t.Fatalf("声明路径落点应逐字节一致: %q %v", string(raw), err)
	}
	// ── 缺陷例：越界写正确失败且零落盘 ──
	if _, err := wf.Execute(context.Background(), p3Args(t, map[string]any{
		"path": "../逃逸.txt", "content": "x",
	})); err == nil {
		t.Fatal("越界写应拒绝")
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "逃逸.txt")); !os.IsNotExist(err) {
		t.Fatalf("越界文件不得落盘: %v", err)
	}
}

// p3TaskOrchestrate 任务编排类：todo 编排进度落盘可恢复；非法输入正确失败
// （dag_plan/task 六闸刀断言见 dag 与 app 包测试）。
func p3TaskOrchestrate(t *testing.T) {
	// todo_write 经 findGaeaDir(os.Getwd()) 落 .gaea/todos.md——chdir 到带
	// .gaea 的临时目录（进程 cwd 固定回还原，参照 addbuiltins_test 先例）。
	old, _ := os.Getwd()
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".gaea"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(ws); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	// ── 金标例：合法编排 → 计数正确 + .gaea/todos.md 落盘（可恢复）──
	tw := todoWrite{}
	out, err := tw.Execute(context.Background(), p3Args(t, map[string]any{
		"todos": []map[string]any{
			{"content": "汇总成本数据", "status": "completed", "level": 0},
			{"content": "生成报告", "status": "in_progress", "level": 0},
		},
	}))
	if err != nil {
		t.Fatalf("todo_write: %v", err)
	}
	if !strings.Contains(out, "2 total") || !strings.Contains(out, "1 completed") || !strings.Contains(out, "1 in progress") {
		t.Fatalf("编排计数应正确: %q", out)
	}
	md, err := os.ReadFile(filepath.Join(ws, ".gaea", "todos.md"))
	if err != nil || !strings.Contains(string(md), "生成报告") {
		t.Fatalf("进度应落盘 .gaea/todos.md（断窗恢复原料）: %q %v", string(md), err)
	}
	// ── 缺陷例：非法状态 → 正确失败 ──
	if _, err := tw.Execute(context.Background(), p3Args(t, map[string]any{
		"todos": []map[string]any{{"content": "x", "status": "done"}},
	})); err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Fatalf("非法状态应正确失败: %v", err)
	}
}
