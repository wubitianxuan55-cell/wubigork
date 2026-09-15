package app

// 阶段七 7.3-1 任务收件箱 App 层测试（线 B，规格 §2.4）：状态文件往返与
// 容错、Save 新建/编辑/来源不可变/校验拒绝/超上限、SetStatus 状态机集成面、
// space 过滤与排序、execSaveTask（voice/weixin 双例 + 空间取法/回退）、
// GaeaRouteIntent dry-run 预览零落盘与执行落盘。隔离工作区手法抄
// gaea_skill_distill_test（isolateWorkspaceTo + GAEA_DATA_ROOT）。

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/taskinbox"
)

// taskInboxFixture 隔离工作区 + 独立数据根：收件箱状态文件落 t.TempDir，
// 不污染真实用户数据（零常驻判据外的测试纪律）。
func taskInboxFixture(t *testing.T) (*App, string) {
	t.Helper()
	isolateWorkspaceTo(t, t.TempDir())
	dataRoot := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)
	return &App{}, dataRoot
}

// taskInboxReq Save 请求 → reqJSON。
func taskInboxReq(t *testing.T, in taskInboxSaveInput) string {
	t.Helper()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// taskInboxSetTestSpace 直接改全局 gaea 配置的生效空间（isolateWorkspaceTo
// 已装临时 cfg；测试内再切 play/off 验证 execSaveTask 的取法与回退）。
func taskInboxSetTestSpace(t *testing.T, space, mode string) {
	t.Helper()
	ga.mu.Lock()
	defer ga.mu.Unlock()
	ga.cfg.Session.Space = space
	ga.cfg.Space.Mode = mode
}

// reTaskInboxID 任务 ID 形状（"ti-"+12 小写十六进制）。
var reTaskInboxID = regexp.MustCompile(`^ti-[0-9a-f]{12}$`)

// TestTaskInboxRoundTrip 新建→列表→状态迁移→删除全往返 + 编辑只改
// title/note（来源字段不可变）+ camelCase 文件形状钉死。
func TestTaskInboxRoundTrip(t *testing.T) {
	a, dataRoot := taskInboxFixture(t)

	// 新建：title trim、status=pending、ID 形状、CreatedAt==UpdatedAt
	v, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{
		Title: "  写周报  ", Space: "work", Source: "voice",
		Action: "save_task", Target: "写周报", Session: "sess-1", Note: "周五前",
	}))
	if err != nil {
		t.Fatalf("新建任务: %v", err)
	}
	if !reTaskInboxID.MatchString(v.ID) {
		t.Errorf("ID 形状不符: %q", v.ID)
	}
	if v.Title != "写周报" || v.Space != "work" || v.Status != taskinbox.Status("pending") || v.Source != "voice" {
		t.Errorf("新建字段不符: %+v", v)
	}
	if v.Action != "save_task" || v.Target != "写周报" || v.Session != "sess-1" || v.Note != "周五前" {
		t.Errorf("新建审计字段不符: %+v", v)
	}
	if v.CreatedAt <= 0 || v.UpdatedAt != v.CreatedAt {
		t.Errorf("新建时间戳不符: created=%d updated=%d", v.CreatedAt, v.UpdatedAt)
	}

	// 文件形状：camelCase 键 + version 1（与前端/状态文件契约钉死）
	b, err := os.ReadFile(taskInboxPath(dataRoot))
	if err != nil {
		t.Fatalf("读状态文件: %v", err)
	}
	raw := string(b)
	for _, want := range []string{
		`"version": 1`, `"tasks"`, `"id"`, `"title"`, `"space"`,
		`"status": "pending"`, `"source"`, `"action"`, `"target"`,
		`"session"`, `"note"`, `"createdAt"`, `"updatedAt"`,
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("状态文件缺少 %q:\n%s", want, raw)
		}
	}
	for _, banned := range []string{"created_at", "updated_at", "CreatedAt", "UpdatedAt"} {
		if strings.Contains(raw, banned) {
			t.Errorf("状态文件不应含非 camelCase 键 %q:\n%s", banned, raw)
		}
	}

	// 列表：work 命中、play 空、空=全部
	if list := a.GaeaTaskInboxList("work"); len(list) != 1 || list[0].ID != v.ID {
		t.Fatalf("List(work) 应 1 条: %+v", list)
	}
	if list := a.GaeaTaskInboxList("play"); len(list) != 0 || list == nil {
		t.Fatalf("List(play) 应空表（非 nil）: %+v", list)
	}
	if list := a.GaeaTaskInboxList(""); len(list) != 1 {
		t.Fatalf("List(空) 应全部 1 条: %+v", list)
	}

	// 状态迁移：pending→doing→done 合法链
	if _, err := a.GaeaTaskInboxSetStatus(v.ID, "doing"); err != nil {
		t.Fatalf("pending→doing: %v", err)
	}
	if _, err := a.GaeaTaskInboxSetStatus(v.ID, "done"); err != nil {
		t.Fatalf("doing→done: %v", err)
	}
	// 终态拒绝任何迁移；同值短路成功（无变化不落盘——文件字节不变）
	if _, err := a.GaeaTaskInboxSetStatus(v.ID, "doing"); err == nil {
		t.Error("终态 done→doing 应拒绝")
	}
	before, _ := os.ReadFile(taskInboxPath(dataRoot))
	same, err := a.GaeaTaskInboxSetStatus(v.ID, "done")
	if err != nil {
		t.Fatalf("同值迁移应短路成功: %v", err)
	}
	if same.Status != taskinbox.Status("done") {
		t.Errorf("同值短路应返回原状态: %+v", same)
	}
	after, _ := os.ReadFile(taskInboxPath(dataRoot))
	if string(before) != string(after) {
		t.Error("同值短路不应落盘（文件字节应不变）")
	}

	// 编辑：只改 title/note；Space/Source/Action/Target/Session/CreatedAt
	// 不可变（来源审计链——入参带歪数据也不许洗白）
	edited, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{
		ID: v.ID, Title: "写月报", Space: "play", Source: "ctrlk",
		Action: "navigate", Target: "别的", Session: "sess-9", Note: "改备注",
	}))
	if err != nil {
		t.Fatalf("编辑任务: %v", err)
	}
	if edited.Title != "写月报" || edited.Note != "改备注" {
		t.Errorf("编辑应只改 title/note: %+v", edited)
	}
	if edited.Space != "work" || edited.Source != "voice" || edited.Action != "save_task" ||
		edited.Target != "写周报" || edited.Session != "sess-1" {
		t.Errorf("来源字段不可变被违反: %+v", edited)
	}
	if edited.CreatedAt != v.CreatedAt || edited.UpdatedAt < v.UpdatedAt {
		t.Errorf("CreatedAt 不可变且 UpdatedAt 应推进: %+v", edited)
	}

	// 删除：删后空表；重复删如实报错
	if err := a.GaeaTaskInboxDelete(v.ID); err != nil {
		t.Fatalf("删除任务: %v", err)
	}
	if list := a.GaeaTaskInboxList(""); len(list) != 0 {
		t.Fatalf("删除后应空表: %+v", list)
	}
	if err := a.GaeaTaskInboxDelete(v.ID); err == nil || !strings.Contains(err.Error(), "任务不存在") {
		t.Fatalf("重复删除应报任务不存在: %v", err)
	}
}

// TestTaskInboxTolerantLoad 损坏/缺字段文件容错回空表（收件箱丢得起）。
func TestTaskInboxTolerantLoad(t *testing.T) {
	a, dataRoot := taskInboxFixture(t)
	if err := os.WriteFile(taskInboxPath(dataRoot), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if tasks := loadTaskInbox(dataRoot); len(tasks) != 0 || tasks == nil {
		t.Fatalf("损坏文件应回空表（非 nil）: %+v", tasks)
	}
	if list := a.GaeaTaskInboxList("work"); len(list) != 0 || list == nil {
		t.Fatalf("损坏文件 List 应空表: %+v", list)
	}
	// tasks 缺省（version-only 文件）同样回空
	if err := os.WriteFile(taskInboxPath(dataRoot), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if tasks := loadTaskInbox(dataRoot); len(tasks) != 0 {
		t.Fatalf("version-only 应回空表: %+v", tasks)
	}
}

// TestTaskInboxSaveValidation 非法 space/source/空标题/坏 JSON/坏 ID/未知
// ID 一律触盘前拦截；source 缺省兜底 inbox（手动新建第五来源）。
func TestTaskInboxSaveValidation(t *testing.T) {
	a, dataRoot := taskInboxFixture(t)

	if _, err := a.GaeaTaskInboxSave("{bad-json"); err == nil {
		t.Error("坏 JSON 应报错")
	}
	if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "x", Space: "home"})); err == nil {
		t.Error("非法 space 应拒绝")
	}
	if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "x", Space: "work", Source: "web"})); err == nil {
		t.Error("非法 source 应拒绝")
	}
	if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "   ", Space: "work"})); err == nil {
		t.Error("空标题应拒绝")
	}
	// 坏 ID 形状编辑拒绝（残渣防御）
	for _, badID := range []string{"ti", "ti-0123456", "ti-012345678901234", "TI-012345678901", "jd-01234567"} {
		if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{ID: badID, Title: "x"})); err == nil {
			t.Errorf("坏 ID 编辑应拒绝: %q", badID)
		}
		if err := a.GaeaTaskInboxDelete(badID); err == nil {
			t.Errorf("坏 ID 删除应拒绝: %q", badID)
		}
		if _, err := a.GaeaTaskInboxSetStatus(badID, "doing"); err == nil {
			t.Errorf("坏 ID 迁移应拒绝: %q", badID)
		}
	}
	// 未知 ID 编辑如实报错
	if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{ID: "ti-000000000000", Title: "x"})); err == nil || !strings.Contains(err.Error(), "任务不存在") {
		t.Errorf("未知 ID 编辑应报任务不存在: %v", err)
	}
	// 校验失败不应产生状态文件
	if _, err := os.Stat(taskInboxPath(dataRoot)); !os.IsNotExist(err) {
		t.Error("校验失败不应产生状态文件")
	}

	// source 缺省 → inbox（兜底）；合法五来源放行
	v, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "买咖啡", Space: "play"}))
	if err != nil || v.Source != "inbox" {
		t.Fatalf("source 缺省应兜底 inbox: %+v err=%v", v, err)
	}
	for _, src := range []string{"ctrlk", "palette", "voice", "weixin", "inbox"} {
		if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "t-" + src, Space: "work", Source: src})); err != nil {
			t.Errorf("合法 source %q 应放行: %v", src, err)
		}
	}
}

// TestTaskInboxSaveMaxTasks 超上限拒绝新建（防膨胀）；编辑与删除不受限，
// 删一条腾位后可再建。
func TestTaskInboxSaveMaxTasks(t *testing.T) {
	a, dataRoot := taskInboxFixture(t)
	full := make([]taskinbox.Task, 0, taskinbox.MaxTasks)
	for i := 0; i < taskinbox.MaxTasks; i++ {
		full = append(full, taskinbox.Task{
			ID: fmt.Sprintf("ti-%012x", i), Title: fmt.Sprintf("任务 %d", i),
			Space: "work", Status: taskinbox.Status("pending"), Source: "inbox",
			CreatedAt: int64(i), UpdatedAt: int64(i),
		})
	}
	if err := saveTaskInbox(dataRoot, full); err != nil {
		t.Fatalf("铺满状态文件: %v", err)
	}
	if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "再多一条", Space: "work"})); err == nil || !strings.Contains(err.Error(), "上限") {
		t.Fatalf("超上限应拒绝新建: %v", err)
	}
	// 编辑既有任务不受上限影响
	if v, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{ID: full[0].ID, Title: "改标题"})); err != nil || v.Title != "改标题" {
		t.Fatalf("满员编辑应放行: %+v err=%v", v, err)
	}
	// 删一条腾位 → 可再建
	if err := a.GaeaTaskInboxDelete(full[taskinbox.MaxTasks-1].ID); err != nil {
		t.Fatalf("满员删除: %v", err)
	}
	if _, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "腾位新建", Space: "work", Source: "inbox"})); err != nil {
		t.Fatalf("腾位后新建应放行: %v", err)
	}
}

// TestTaskInboxSetStatusGuards 状态机集成面：非法状态串拒绝、pending→done
// 跨步拒绝、pending→abandoned 放行、未知任务报错。
func TestTaskInboxSetStatusGuards(t *testing.T) {
	a, _ := taskInboxFixture(t)
	v, err := a.GaeaTaskInboxSave(taskInboxReq(t, taskInboxSaveInput{Title: "交房租", Space: "work", Source: "voice"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaTaskInboxSetStatus(v.ID, "paused"); err == nil {
		t.Error("未知状态串应拒绝")
	}
	if _, err := a.GaeaTaskInboxSetStatus(v.ID, "done"); err == nil {
		t.Error("pending→done 跨步应拒绝（必须经 doing）")
	}
	if _, err := a.GaeaTaskInboxSetStatus(v.ID, "abandoned"); err != nil {
		t.Fatalf("pending→abandoned 应放行: %v", err)
	}
	if _, err := a.GaeaTaskInboxSetStatus("ti-ffffffffffff", "doing"); err == nil || !strings.Contains(err.Error(), "任务不存在") {
		t.Errorf("未知任务应报不存在: %v", err)
	}
}

// TestTaskInboxListSpaceFilterSort space 过滤（work/play/空=全部）+ 排序
// 集成（状态组序 pending<doing<done<abandoned、组内 UpdatedAt 降序）。
func TestTaskInboxListSpaceFilterSort(t *testing.T) {
	a, dataRoot := taskInboxFixture(t)
	mk := func(id, space, status string, updated int64) taskinbox.Task {
		return taskinbox.Task{ID: id, Title: "t-" + id, Space: space,
			Status: taskinbox.Status(status), Source: "inbox", CreatedAt: 1, UpdatedAt: updated}
	}
	if err := saveTaskInbox(dataRoot, []taskinbox.Task{
		mk("ti-000000000001", "work", "doing", 100),
		mk("ti-000000000002", "work", "pending", 300),
		mk("ti-000000000003", "play", "pending", 999),
		mk("ti-000000000004", "work", "pending", 200),
		mk("ti-000000000005", "work", "done", 50),
	}); err != nil {
		t.Fatal(err)
	}
	work := a.GaeaTaskInboxList("work")
	if len(work) != 4 {
		t.Fatalf("List(work) 应 4 条: %+v", work)
	}
	wantOrder := []string{"ti-000000000002", "ti-000000000004", "ti-000000000001", "ti-000000000005"}
	for i, want := range wantOrder {
		if work[i].ID != want {
			t.Errorf("work 排序[%d] = %s, want %s（状态组序+组内时间降序）", i, work[i].ID, want)
		}
	}
	if play := a.GaeaTaskInboxList("play"); len(play) != 1 || play[0].ID != "ti-000000000003" {
		t.Fatalf("List(play) 应严格隔离只含 play 任务: %+v", play)
	}
	if all := a.GaeaTaskInboxList(""); len(all) != 5 {
		t.Fatalf("List(空) 应全部 5 条: %+v", all)
	}
	// 判据③钉死：FilterBySpace 严格相等，非法 space 串不泄漏跨空间任务
	if leak := a.GaeaTaskInboxList("WORK"); len(leak) != 0 {
		t.Errorf("非法 space 应空表不泄漏: %+v", leak)
	}
}

// TestSaveTaskExecVoiceWeixin 语音/微信双例：assistantID 判 source
// （空=voice、非空=weixin）、space 取 gaeaEffectiveSpace（默认 work、
// 切 play 生效）、Action/Target 存意图原值（审计链）。
func TestSaveTaskExecVoiceWeixin(t *testing.T) {
	a, _ := taskInboxFixture(t)

	// 例 1：语音（assistantID 空）→ source=voice、space=默认 work
	res := a.routeIntentModeForAssistant("存个任务 买咖啡", false, "")
	if !res.Handled || res.Reply != "已存入任务收件箱：买咖啡" {
		t.Fatalf("voice 例回复不符: %+v", res)
	}
	list := a.GaeaTaskInboxList("work")
	if len(list) != 1 {
		t.Fatalf("voice 例应落 1 条 work 任务: %+v", list)
	}
	v := list[0]
	if v.Title != "买咖啡" || v.Source != "voice" || v.Space != "work" || v.Status != taskinbox.Status("pending") {
		t.Errorf("voice 例字段不符: %+v", v)
	}
	if v.Action != "save_task" || v.Target != "买咖啡" {
		t.Errorf("Action/Target 应存意图原值: %+v", v)
	}

	// 例 2：微信（assistantID 非空）→ source=weixin、space 跟随 play
	taskInboxSetTestSpace(t, "play", "")
	wx := a.routeIntentWithResultForAssistant("记个任务 给老板发报告", "wx-assistant-1")
	if !wx.Handled || wx.Reply != "已存入任务收件箱：给老板发报告" {
		t.Fatalf("weixin 例回复不符: %+v", wx)
	}
	play := a.GaeaTaskInboxList("play")
	if len(play) != 1 {
		t.Fatalf("weixin 例应落 1 条 play 任务: %+v", play)
	}
	if play[0].Source != "weixin" || play[0].Space != "play" || play[0].Title != "给老板发报告" {
		t.Errorf("weixin 例字段不符: %+v", play[0])
	}
	if play[0].Action != "save_task" || play[0].Target != "给老板发报告" {
		t.Errorf("weixin 例审计字段不符: %+v", play[0])
	}
	// 空间隔离：voice 那条仍在 work，不串空间
	if work := a.GaeaTaskInboxList("work"); len(work) != 1 || work[0].Source != "voice" {
		t.Errorf("双例落库应空间隔离互不串: %+v", work)
	}
}

// TestSaveTaskExecSpaceFallback gaeaEffectiveSpace 返回空/非法（space.mode
// =off）时回退 work——后端入口的诚实取法，不丢任务。
func TestSaveTaskExecSpaceFallback(t *testing.T) {
	a, _ := taskInboxFixture(t)
	taskInboxSetTestSpace(t, "", "off") // EffectiveSessionSpace → ""
	if res := a.routeIntentModeForAssistant("存个任务 交水电费", false, ""); !res.Handled {
		t.Fatalf("应命中存任务: %+v", res)
	}
	list := a.GaeaTaskInboxList("work")
	if len(list) != 1 || list[0].Space != "work" {
		t.Fatalf("空空间应回退 work: %+v", list)
	}
}

// TestGaeaRouteIntentSaveTaskPreview 执行预览：dryRun=true 命中 save_task
// 且零落盘（Ctrl+K 预览-确认制）；dryRun=false 真执行落盘。
func TestGaeaRouteIntentSaveTaskPreview(t *testing.T) {
	a, dataRoot := taskInboxFixture(t)

	pre := a.GaeaRouteIntent("存个任务 买咖啡", true)
	if !pre.Handled || pre.Action != "save_task" || pre.Target != "买咖啡" {
		t.Fatalf("预览应命中 save_task: %+v", pre)
	}
	if !strings.Contains(pre.Reply, "将存为任务") || !strings.Contains(pre.Reply, "买咖啡") {
		t.Errorf("预览回复应描述将发生什么: %q", pre.Reply)
	}
	if _, err := os.Stat(taskInboxPath(dataRoot)); !os.IsNotExist(err) {
		t.Fatal("dry-run 预览必须零落盘")
	}

	exec := a.GaeaRouteIntent("存个任务 买咖啡", false)
	if !exec.Handled || exec.Action != "save_task" {
		t.Fatalf("执行应命中: %+v", exec)
	}
	list := a.GaeaTaskInboxList("")
	if len(list) != 1 || list[0].Source != "voice" || list[0].Title != "买咖啡" {
		t.Fatalf("执行应落盘 1 条 voice 任务: %+v", list)
	}
}
