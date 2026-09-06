import { describe, expect, it } from "vitest";
import { initScheduleSync, normalizeProject, notifyScheduleFileChanged, useScheduleStore } from "./store";
import { makeSampleProject } from "./sample";
import type { SchedProject } from "./types";

// store.sync.test.ts — 计划文件同步链（v4.113 刀4 / v4.114 刀5）：
// 水合（mock 文件缺失 → localStorage 迁移上文件）→ 编辑防抖自动保存 →
// agent 外部写 + notifyScheduleFileChanged 即时回读。
// 走 bridge mock（window.go 未定义即 mockApp），文件态即 window.__mockScheduleFile。

const mockFile = () => (window as unknown as { __mockScheduleFile?: { load(): { exists: boolean; project: string }; save(j: string): unknown } }).__mockScheduleFile;

describe("schedule 计划文件同步链（bridge mock）", () => {
  it("迁移→防抖自动保存→agent 外部写回读 全链", async () => {
    localStorage.removeItem("gaea.schedule.v1");
    localStorage.removeItem("gaea.schedule.v2"); // 历史键卫生
    useScheduleStore.setState({
      project: normalizeProject(makeSampleProject()),
      hydrated: false,
      sync: "idle",
      savedAt: null,
      syncError: null,
    });

    // ① 水合：mock 文件不存在 → 迁移分支落盘
    await initScheduleSync();
    const st = useScheduleStore.getState();
    expect(st.hydrated).toBe(true);
    expect(mockFile()?.load().exists).toBe(true);
    const seeded = JSON.parse(mockFile()!.load().project) as SchedProject;
    expect(seeded.tasks.length).toBeGreaterThan(0);

    // ② 编辑 → dirty → 防抖后 saved（mock Save 走真实 CPM 纯函数）
    useScheduleStore.getState().renameProject("同步链测试工程");
    expect(useScheduleStore.getState().sync).toBe("dirty");
    await new Promise((r) => setTimeout(r, 1200));
    expect(useScheduleStore.getState().sync).toBe("saved");
    const saved = JSON.parse(mockFile()!.load().project) as SchedProject;
    expect(saved.name).toBe("同步链测试工程");

    // ③ agent 外部写（schedule_apply 同语义）→ notify 即时回读
    const external = JSON.parse(mockFile()!.load().project) as SchedProject;
    external.name = "agent 改名";
    external.tasks.push({ id: "x1", name: "agent 新增任务", duration: 2, level: 1, progress: 0 });
    mockFile()!.save(JSON.stringify(external));
    notifyScheduleFileChanged();
    await new Promise((r) => setTimeout(r, 150));
    const after = useScheduleStore.getState();
    expect(after.project.name).toBe("agent 改名");
    expect(after.project.tasks.some((t) => t.id === "x1")).toBe(true);
    expect(after.sync).toBe("saved");
  });

  it("notifyScheduleFileChanged 未初始化时静默（板块未挂载也不炸）", () => {
    expect(() => notifyScheduleFileChanged()).not.toThrow();
  });
});
