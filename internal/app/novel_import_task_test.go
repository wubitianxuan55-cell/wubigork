package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/bookimport"
	"github.com/gaea/gaea/internal/gaea/tasks"
	"github.com/gaea/gaea/internal/project"
)

// 长书后台任务态（v4.291）：反推任务化 e2e——提交 → 队列执行 → 取预览。

func TestNovelOutlineReconstruct_TaskFlow(t *testing.T) {
	a := newTestTaskApp(t)
	a.writingState = &writingState{} // 裸 App 需手动初始化写作态
	startTasks(t, a)
	a.officeState.tasks.Register(tasks.KindOutlineReconstruct, a.outlineReconstructTaskHandler)

	// 60 章 × ~600 字导入工程（中篇，反推走规则兜底）
	novelsDir := t.TempDir()
	body := strings.Repeat("这一段正文用来撑起篇幅，讲一个完整的小事件且有起承转合。", 30)
	chapters := make([]importChapter, 0, 60)
	for i := 1; i <= 60; i++ {
		chapters = append(chapters, importChapter{
			Title:   "第" + strconvItoa(i) + "章 试炼",
			Content: "第" + strconvItoa(i) + "章正文。\n\n" + body,
		})
	}
	res, err := createImportedProject(novelsDir, "任务化测试", "玄幻", "默认", chapters, bookimport.Report{SplitStrategy: "booksource"})
	if err != nil {
		t.Fatalf("建导入工程: %v", err)
	}
	pm, err := project.Open(res.Path)
	if err != nil {
		t.Fatalf("打开工程: %v", err)
	}
	a.setPM(pm)

	// 提交
	st, err := a.NovelOutlineReconstructStart()
	if err != nil {
		t.Fatalf("提交反推任务: %v", err)
	}
	if st.TaskID == "" || st.Status != "queued" {
		t.Fatalf("提交回执不符: %+v", st)
	}

	// 等终态
	done, err := a.officeState.tasks.Wait(context.Background(), st.TaskID, 30*time.Second)
	if err != nil {
		t.Fatalf("等待任务: %v", err)
	}
	if done.Status != "succeeded" {
		t.Fatalf("任务应成功: %+v", done)
	}

	// 取预览
	got, err := a.NovelOutlineReconstructTaskGet()
	if err != nil {
		t.Fatalf("取任务结果: %v", err)
	}
	if got.Preview == nil || got.Preview.Tier != "mid" || len(got.Preview.Items) != 6 {
		t.Fatalf("预览不符: %+v", got)
	}
}

// 队列不可用与无任务两种诚实失败。
func TestNovelOutlineReconstructTask_Paths(t *testing.T) {
	// 队列不可用：Start 报错（App 无 tasks manager）
	a := newCharacterLibTestApp(t)
	if _, err := a.NovelOutlineReconstructStart(); err == nil {
		t.Fatal("无任务队列应报错")
	}

	// 队列可用但无反推任务：Get 报错
	a2 := newTestTaskApp(t)
	startTasks(t, a2)
	if _, err := a2.NovelOutlineReconstructTaskGet(); err == nil {
		t.Fatal("无反推任务应报错")
	}
}
