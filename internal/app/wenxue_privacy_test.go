package app

// wenxue_privacy_test.go — 图文私密保护行为锁（2026-11-23 无限制档小说四形态）。
// 覆盖：PIN 设置/校验/退避/回锁、隐藏清单与目录属性、列表锁定过滤、
// Overview 锁定态脱敏、AdultLevel 填充、日志脱敏、统一搜索 tuwen 摘除。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wenxuePrivacyWriteBook 测试书目录 + 书籍信息.md 档位行。
func wenxuePrivacyWriteBook(t *testing.T, root, name, level string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	info := "# 书籍信息\n\n- 书名：" + name + "\n\n- 成人等级：" + level + "\n"
	if level == "" {
		info = "# 书籍信息\n\n- 书名：" + name + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "书籍信息.md"), []byte(info), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestWenxuePrivacyPinLifecycle(t *testing.T) {
	a, root := isolateWenxueRuntime(t)

	// 未设 PIN：不锁。
	if a.wenxuePrivacyIsLocked() {
		t.Fatal("未设 PIN 不应处于锁定态")
	}
	// 非法 PIN 拒绝。
	if err := a.WenxuePrivacySetPin("123", ""); err == nil {
		t.Fatal("短 PIN 应拒绝")
	}
	if err := a.WenxuePrivacySetPin("12ab56", ""); err == nil {
		t.Fatal("非数字 PIN 应拒绝")
	}
	// 首次设置（无原 PIN）：进入锁定态。
	if err := a.WenxuePrivacySetPin("123456", ""); err != nil {
		t.Fatalf("首次设 PIN 失败: %v", err)
	}
	if !a.wenxuePrivacyIsLocked() {
		t.Fatal("设 PIN 后应处于锁定态")
	}
	// 修改 PIN：原 PIN 错误拒绝。
	if err := a.WenxuePrivacySetPin("654321", "000000"); err == nil {
		t.Fatal("原 PIN 错误应拒绝改 PIN")
	}
	// 解锁：错误 PIN / 正确 PIN。
	if ok, err := a.WenxuePrivacyUnlock("000000"); ok || err == nil {
		t.Fatal("错误 PIN 不应解锁")
	}
	if ok, err := a.WenxuePrivacyUnlock("123456"); !ok || err != nil {
		t.Fatalf("正确 PIN 应解锁: %v", err)
	}
	if a.wenxuePrivacyIsLocked() {
		t.Fatal("解锁后不应处于锁定态")
	}
	// 正确原 PIN 改 PIN：改完即锁，新 PIN 生效旧 PIN 失效。
	if err := a.WenxuePrivacySetPin("654321", "123456"); err != nil {
		t.Fatalf("改 PIN 失败: %v", err)
	}
	if !a.wenxuePrivacyIsLocked() {
		t.Fatal("改 PIN 后应回锁")
	}
	if ok, _ := a.WenxuePrivacyUnlock("123456"); ok {
		t.Fatal("旧 PIN 不应解锁")
	}
	if ok, err := a.WenxuePrivacyUnlock("654321"); !ok || err != nil {
		t.Fatalf("新 PIN 应解锁: %v", err)
	}
	// 手动回锁 + 重启语义（新 App 实例=内存态清零）。
	a.WenxuePrivacyLock()
	if !a.wenxuePrivacyIsLocked() {
		t.Fatal("手动回锁后应处于锁定态")
	}
	_ = root
}

func TestWenxuePrivacyPinLockout(t *testing.T) {
	a, _ := isolateWenxueRuntime(t)
	if err := a.WenxuePrivacySetPin("123456", ""); err != nil {
		t.Fatal(err)
	}
	var lockoutErr error
	for i := 0; i < 5; i++ {
		_, lockoutErr = a.WenxuePrivacyUnlock("000000")
		if i < 4 && lockoutErr == nil {
			t.Fatalf("第 %d 次错误 PIN 应报「不正确」", i+1)
		}
	}
	if lockoutErr == nil || !strings.Contains(lockoutErr.Error(), "临时锁定") {
		t.Fatalf("连续 5 次失败应进入退避，got %v", lockoutErr)
	}
	// 退避期内正确 PIN 也拒绝。
	if ok, err := a.WenxuePrivacyUnlock("123456"); ok || err == nil {
		t.Fatal("退避期内不应解锁")
	}
}

func TestWenxuePrivacyHiddenListAndAttribute(t *testing.T) {
	a, root := isolateWenxueRuntime(t)
	dir := wenxuePrivacyWriteBook(t, root, "私密书", "深度成人（无限制）")

	// 隐藏开关：清单写入 + 目录属性位。
	if err := a.WenxuePrivacySetBookHidden("私密书", true); err != nil {
		t.Fatalf("隐藏失败: %v", err)
	}
	if !wenxuePrivacyBookHidden("私密书") {
		t.Fatal("隐藏清单缺书名")
	}
	if attrs, err := windowsGetFileAttributes(dir); err != nil || attrs&fileAttributeHidden == 0 {
		t.Fatalf("目录应带隐藏属性: attrs=%d err=%v", attrs, err)
	}
	// 取消隐藏：清单摘除 + 属性位摘除。
	if err := a.WenxuePrivacySetBookHidden("私密书", false); err != nil {
		t.Fatalf("取消隐藏失败: %v", err)
	}
	if wenxuePrivacyBookHidden("私密书") {
		t.Fatal("隐藏清单应已摘除")
	}
	if attrs, err := windowsGetFileAttributes(dir); err != nil || attrs&fileAttributeHidden != 0 {
		t.Fatalf("目录隐藏属性应摘除: attrs=%d err=%v", attrs, err)
	}
	// 穿越/越界名拒绝。
	if err := a.WenxuePrivacySetBookHidden(`..\逃逸`, true); err == nil {
		t.Fatal("带路径分隔符书名应拒绝")
	}
	if err := a.WenxuePrivacySetBookHidden("不存在的书", true); err == nil {
		t.Fatal("不存在书名应拒绝")
	}
}

func TestWenxuePrivacyListFilterAndAdultLevel(t *testing.T) {
	a, root := isolateWenxueRuntime(t)
	wenxuePrivacyWriteBook(t, root, "普通书", "全年龄")
	wenxuePrivacyWriteBook(t, root, "无底线书", "深度成人（无限制）")
	wenxuePrivacyWriteBook(t, root, "旧书", "") // 缺档位行 → AdultLevel 空

	// 未设 PIN：列表全量 + adultLevel 填充。
	list := a.WenxueListWorkspaces()
	if len(list) != 3 {
		t.Fatalf("未设 PIN 列表应 3 书，got %d", len(list))
	}
	levels := map[string]string{}
	for _, v := range list {
		levels[v.Name] = v.AdultLevel
	}
	if levels["普通书"] != "全年龄" || levels["无底线书"] != "深度成人（无限制）" || levels["旧书"] != "" {
		t.Fatalf("adultLevel 填充不符: %v", levels)
	}
	// 隐藏无底线书：未设 PIN 也恒滤（2026-11-23 眼睛失效根修——点眼睛即生效；
	// 恢复路径=隐私区眼睛开关，死局不存在）。
	if err := a.WenxuePrivacySetBookHidden("无底线书", true); err != nil {
		t.Fatal(err)
	}
	list = a.WenxueListWorkspaces()
	if len(list) != 2 {
		t.Fatalf("未设 PIN 隐藏书也应出列表（2），got %d", len(list))
	}
	// 设 PIN → 锁定态：隐藏书仍不在（恒滤语义与锁定态无关）。
	if err := a.WenxuePrivacySetPin("123456", ""); err != nil {
		t.Fatal(err)
	}
	list = a.WenxueListWorkspaces()
	if len(list) != 2 {
		t.Fatalf("锁定态列表应滤隐藏书（2），got %d", len(list))
	}
	for _, v := range list {
		if v.Name == "无底线书" {
			t.Fatal("隐藏书不应出现在列表")
		}
	}
	// 解锁 → 仍不在（恒滤；管理面走 Overview）。
	if ok, err := a.WenxuePrivacyUnlock("123456"); !ok || err != nil {
		t.Fatal(err)
	}
	if len(a.WenxueListWorkspaces()) != 2 {
		t.Fatal("解锁后隐藏书也应被恒滤（管理面走 Overview）")
	}
	// 现书豁免：隐藏当前书不出 ListWorkspaces（FileTree/引擎锚依赖 current
	// 条目），但出 ListProjectSessions（侧栏/首页浏览面恒滤，含现书）。
	dir := filepath.Join(root, "无底线书")
	if got := a.WenxueSwitchWorkspace(dir); !strings.EqualFold(got, dir) {
		t.Fatalf("切到隐藏书失败: %s", got)
	}
	list = a.WenxueListWorkspaces()
	found := false
	for _, v := range list {
		if v.Name == "无底线书" && v.Current {
			found = true
		}
	}
	if !found {
		t.Fatal("现书（隐藏）应保留在 ListWorkspaces（current 豁免）")
	}
	for _, g := range a.WenxueListProjectSessions() {
		if g.Name == "无底线书" {
			t.Fatal("隐藏书（含现书）分组不应出现在项目会话列表")
		}
	}
}

func TestWenxuePrivacyOverviewRedaction(t *testing.T) {
	a, root := isolateWenxueRuntime(t)
	wenxuePrivacyWriteBook(t, root, "普通书", "全年龄")
	wenxuePrivacyWriteBook(t, root, "无底线书", "深度成人（无限制）")
	if err := a.WenxuePrivacySetBookHidden("无底线书", true); err != nil {
		t.Fatal(err)
	}
	if err := a.WenxuePrivacySetPin("123456", ""); err != nil {
		t.Fatal(err)
	}
	ov := a.WenxuePrivacyOverview()
	if !ov.PinSet || ov.Unlocked {
		t.Fatalf("总览应 pinSet=true unlocked=false: %+v", ov)
	}
	if ov.HiddenCount != 1 {
		t.Fatalf("hiddenCount 应 1，got %d", ov.HiddenCount)
	}
	for _, b := range ov.Books {
		if b.Hidden && (b.Name != "" || b.Path != "") {
			t.Fatalf("锁定态隐藏书名/路径应置空: %+v", b)
		}
		if !b.Hidden && b.Name == "" {
			t.Fatal("非隐藏书不应被脱敏")
		}
	}
	// 解锁态：隐藏书全量明文（面板编辑需要）。
	if ok, err := a.WenxuePrivacyUnlock("123456"); !ok || err != nil {
		t.Fatal(err)
	}
	ov = a.WenxuePrivacyOverview()
	found := false
	for _, b := range ov.Books {
		if b.Hidden && b.Name == "无底线书" && b.AdultLevel == wenxueAdultUnrestricted {
			found = true
		}
	}
	if !found {
		t.Fatalf("解锁态总览应含隐藏书明文条目: %+v", ov.Books)
	}
	// 遮罩开关读写。
	if !ov.MaskOnBlur {
		t.Fatal("遮罩开关应默认开")
	}
	if err := a.WenxuePrivacySetMaskOnBlur(false); err != nil {
		t.Fatal(err)
	}
	if a.WenxuePrivacyOverview().MaskOnBlur {
		t.Fatal("遮罩开关关闭后应读出 false")
	}
}

func TestWenxuePrivacyRedactPath(t *testing.T) {
	a, root := isolateWenxueRuntime(t)
	wenxuePrivacyWriteBook(t, root, "无底线书", "深度成人（无限制）")
	wenxuePrivacyWriteBook(t, root, "普通书", "全年龄")
	plain := filepath.Join(root, "普通书", "章节", "001.md")
	if got := wenxueRedactPath(plain); got != plain {
		t.Fatalf("普通书路径不应脱敏: %s", got)
	}
	sensitive := filepath.Join(root, "无底线书", "章节", "001.md")
	got := wenxueRedactPath(sensitive)
	if strings.Contains(got, "无底线书") || strings.Contains(got, "001.md") {
		t.Fatalf("无限制书路径应脱敏（书名与文件名都不留）: %s", got)
	}
	if !strings.Contains(got, "私密书籍") {
		t.Fatalf("脱敏应带「私密书籍」标记: %s", got)
	}
	// 隐藏清单书（非无限制档）同样脱敏。
	wenxuePrivacyWriteBook(t, root, "清藏书", "全年龄")
	if err := a.WenxuePrivacySetBookHidden("清藏书", true); err != nil {
		t.Fatal(err)
	}
	hidden := filepath.Join(root, "清藏书", "设定", "世界观.md")
	if got := wenxueRedactPath(hidden); strings.Contains(got, "清藏书") {
		t.Fatalf("隐藏书路径应脱敏: %s", got)
	}
	// tuwen 根外路径原样。
	if got := wenxueRedactPath(`C:\AI\office\doc.md`); got != `C:\AI\office\doc.md` {
		t.Fatalf("根外路径不应脱敏: %s", got)
	}
	// 空串原样。
	if got := wenxueRedactPath(""); got != "" {
		t.Fatalf("空串应原样: %q", got)
	}
}

func TestWenxuePrivacyUnifiedSearchDrop(t *testing.T) {
	a, _ := isolateWenxueRuntime(t)
	tuwen := wenxueWorkspaceRoot()
	// keyword 组：tuwen 内命中摘除、办公命中保留。
	kw := a.wenxueDropTuwenKeywordHits([]WorkspaceSearchHit{
		{Path: "docs/normal.md", Name: "normal.md"},
		{Path: filepath.ToSlash(filepath.Join(tuwen, "无底线书", "章节", "001.md")), Name: "001.md"},
	})
	if len(kw) != 1 || kw[0].Name != "normal.md" {
		t.Fatalf("keyword 组应摘 tuwen 命中: %+v", kw)
	}
	// files 组同口径。
	fh := wenxueDropTuwenFileHits([]FileSemanticHit{
		{Path: "docs/normal.md"},
		{Path: filepath.ToSlash(filepath.Join(tuwen, "无底线书", "章节", "002.md"))},
	})
	if len(fh) != 1 || fh[0].Path != "docs/normal.md" {
		t.Fatalf("files 组应摘 tuwen 命中: %+v", fh)
	}
	// 谓词边界：相对路径（办公工作区相对）不误伤。
	if wenxuePathInTuwenRoot("docs/normal.md") {
		t.Fatal("相对路径不应判为 tuwen 内")
	}
}

func TestWenxuePrivacyAutoHideOnCreate(t *testing.T) {
	a, root := isolateWenxueRuntime(t)
	// 建无限制书：自动入隐藏清单 + 目录属性。
	if _, err := a.WenxueCreateNovel("新书", `{"adultLevel":"深度成人（无限制）"}`); err != nil {
		t.Fatalf("建书失败: %v", err)
	}
	if !wenxuePrivacyBookHidden("新书") {
		t.Fatal("无限制书应自动入隐藏清单")
	}
	if attrs, err := windowsGetFileAttributes(filepath.Join(root, "新书")); err != nil || attrs&fileAttributeHidden == 0 {
		t.Fatalf("新书目录应带隐藏属性: attrs=%d err=%v", attrs, err)
	}
	// 建普通书：不隐藏。
	if _, err := a.WenxueCreateNovel("普书", `{"adultLevel":"全年龄"}`); err != nil {
		t.Fatalf("建书失败: %v", err)
	}
	if wenxuePrivacyBookHidden("普书") {
		t.Fatal("普通书不应入隐藏清单")
	}
}
