import { afterEach, describe, expect, it } from "vitest";
import { initScheduleSync, normalizeProject, notifyScheduleFileChanged, useScheduleStore } from "./store";
import { makeEmptyProject, makeSampleProject } from "./sample";
import type { SchedProject } from "./types";
import { SCHEDULE_FILE_PATH } from "./gschedSummary";
import type { ScheduleProjectMutatedOk, ScheduleProjectSummary, ScheduleProjectsOk } from "./api";

// store.sync.test.ts — 计划文件同步链（v4.113 刀4 / v4.114 刀5 / v4.139 #15 刀1 多工程）：
// 水合（mock 文件缺失 → localStorage 迁移上文件；currentPath 以 Load 回执 path 为准）→
// 编辑防抖自动保存（显式带当前工程 rel，切换竞态的结构性防护）→ agent 外部写 +
// notifyScheduleFileChanged 即时回读。多工程切换/新建/归档/删除/列表缓存走
// window.go stub（真壳同款 Gaea 前缀绑定面；不经 mock/office.ts——它由并行刀
// 独占改造中）。基础链路仍走 bridge mock（window.go 未定义即 mockApp），
// 文件态即 window.__mockScheduleFile。

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
      currentPath: "sentinel/水合前占位.gsched.json", // v4.139 #15：水合后应被 Load 回执 path 覆盖
    });

    // ① 水合：mock 文件不存在 → 迁移分支落盘；currentPath 写入 Load 回执 path
    await initScheduleSync();
    const st = useScheduleStore.getState();
    expect(st.hydrated).toBe(true);
    expect(st.currentPath).toBe(SCHEDULE_FILE_PATH);
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

// ── 多工程切换流（v4.139 #15 刀1，设计 §3.3/3.4）────────────────────────────
// window.go stub 直装 Gaea 前缀绑定（真壳同款命名；短名一并注册，bridge 的
// LegacySurfaceNames 映射由并行刀补齐前后都能命中）。

interface SurfaceStub {
  load: () => Promise<{ path: string; exists: boolean; project: string }>;
  save: (projectJSON: string, rel: string) => Promise<{ path: string; savedAt: string; duration: number; critical: number }>;
  projects?: () => Promise<ScheduleProjectsOk>;
  open?: (rel: string) => Promise<{ current: string }>;
  create?: (name: string) => Promise<{ rel: string; current: string }>;
  archive?: (rel: string, archived: boolean) => Promise<ScheduleProjectMutatedOk>;
  del?: (rel: string) => Promise<ScheduleProjectMutatedOk>;
}

/** 把多工程绑定面装到 window.go.app（bridge realApp 代理按方法名路由到它） */
function stubScheduleSurface(s: SurfaceStub): void {
  const ns: Record<string, unknown> = {
    GaeaScheduleLoad: s.load,
    ScheduleLoad: s.load,
    GaeaScheduleSave: s.save,
    ScheduleSave: s.save,
  };
  if (s.projects) { ns.GaeaScheduleProjects = s.projects; ns.ScheduleProjects = s.projects; }
  if (s.open) { ns.GaeaScheduleProjectOpen = s.open; ns.ScheduleProjectOpen = s.open; }
  if (s.create) { ns.GaeaScheduleProjectCreate = s.create; ns.ScheduleProjectCreate = s.create; }
  if (s.archive) { ns.GaeaScheduleProjectArchive = s.archive; ns.ScheduleProjectArchive = s.archive; }
  if (s.del) { ns.GaeaScheduleProjectDelete = s.del; ns.ScheduleProjectDelete = s.del; }
  (window as unknown as { go?: unknown }).go = { app: { ScheduleTestB: ns } };
}

const OLD = "进度计划/当前计划.gsched.json";

function summary(rel: string, name: string, archived = false): ScheduleProjectSummary {
  return { rel, name, archived, duration: 3, taskCount: 2, ok: true, updatedAt: "2026-09-07T10:00:00Z" };
}

function seeded(p: SchedProject, currentPath: string, sync: "saved" | "dirty" = "saved"): void {
  useScheduleStore.setState({
    project: p,
    hydrated: true,
    sync,
    savedAt: null,
    syncError: null,
    currentPath,
    selectedId: null,
    past: [],
    future: [],
  });
}

describe("多工程切换流（window.go stub，v4.139 #15 刀1）", () => {
  afterEach(() => {
    delete (window as unknown as { go?: unknown }).go; // 还原 bridge mock 通道
  });

  it("自动保存显式带当前工程 rel（防「切换瞬间旧内容写进新文件」）", async () => {
    const CUSTOM = "进度计划/自定义工程.gsched.json";
    const saves: Array<{ rel: string; name: string }> = [];
    stubScheduleSurface({
      load: async () => ({ path: CUSTOM, exists: true, project: JSON.stringify(makeSampleProject()) }),
      save: async (projectJSON, rel) => {
        saves.push({ rel, name: (JSON.parse(projectJSON) as SchedProject).name });
        return { path: rel, savedAt: "10:00", duration: 3, critical: 0 };
      },
    });
    seeded(normalizeProject(makeSampleProject()), CUSTOM);
    useScheduleStore.getState().renameProject("rel 测试");
    expect(useScheduleStore.getState().sync).toBe("dirty");
    await new Promise((r) => setTimeout(r, 1200)); // 过防抖窗口
    expect(useScheduleStore.getState().sync).toBe("saved");
    expect(saves).toHaveLength(1);
    expect(saves[0].rel).toBe(CUSTOM); // 第二参=当前工程 rel
    expect(saves[0].name).toBe("rel 测试");
  });

  it("openProject：dirty 先冲刷旧 rel → 载入新工程 → 指针/历史/选中整体落位", async () => {
    const NEW = "进度计划/办公楼二期.gsched.json";
    const files = new Map<string, SchedProject>([
      [OLD, { ...makeSampleProject(), name: "旧工程" }],
      [NEW, { ...makeSampleProject(), name: "二期工程", tasks: [{ id: "n1", name: "二期任务", duration: 5, level: 1, progress: 0 }] }],
    ]);
    const saves: Array<{ rel: string; name: string }> = [];
    let current = OLD;
    stubScheduleSurface({
      load: async () => ({ path: current, exists: true, project: JSON.stringify(files.get(current)) }),
      save: async (projectJSON, rel) => {
        saves.push({ rel, name: (JSON.parse(projectJSON) as SchedProject).name });
        files.set(rel, JSON.parse(projectJSON) as SchedProject);
        return { path: rel, savedAt: "10:00", duration: 3, critical: 0 };
      },
      open: async (rel) => { current = rel; return { current: rel }; },
    });
    seeded(normalizeProject(files.get(OLD)!), OLD, "dirty"); // 防抖在途同语义
    useScheduleStore.setState({ selectedId: "t1", past: [makeEmptyProject()], future: [makeEmptyProject()] });

    await useScheduleStore.getState().openProject(NEW);
    const st = useScheduleStore.getState();
    expect(saves).toHaveLength(1);
    expect(saves[0].rel).toBe(OLD); // 冲刷写旧 rel，绝不写新文件
    expect(saves[0].name).toBe("旧工程");
    expect(st.currentPath).toBe(NEW);
    expect(st.project.name).toBe("二期工程");
    expect(st.project.tasks.some((t) => t.id === "n1")).toBe(true);
    expect(st.selectedId).toBeNull(); // 切换=新工程新历史
    expect(st.past).toHaveLength(0);
    expect(st.future).toHaveLength(0);
    expect(st.sync).toBe("saved");
    expect(st.syncError).toBeNull();
  });

  it("openProject 绑定失败：fail-closed 保持原工程/currentPath，syncError 有值", async () => {
    const a = normalizeProject({ ...makeSampleProject(), name: "原工程" });
    stubScheduleSurface({
      load: async () => ({ path: OLD, exists: true, project: JSON.stringify(a) }),
      save: async (_projectJSON, rel) => ({ path: rel, savedAt: "10:00", duration: 3, critical: 0 }),
      open: async () => { throw new Error("索引不可写"); },
    });
    seeded(a, OLD);

    await useScheduleStore.getState().openProject("进度计划/不存在.gsched.json");
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(OLD);
    expect(st.project).toBe(a); // 同一引用：未半切换
    expect(st.syncError).toContain("索引不可写");
  });

  it("createProject：重水合到新 rel，工程列表含新条目", async () => {
    const NEW_REL = "进度计划/xin-gong-cheng.gsched.json";
    const files = new Map<string, SchedProject>([[OLD, { ...makeSampleProject(), name: "原工程" }]]);
    let current = OLD;
    let list: ScheduleProjectSummary[] = [summary(OLD, "原工程")];
    stubScheduleSurface({
      load: async () => ({ path: current, exists: files.has(current), project: JSON.stringify(files.get(current) ?? makeEmptyProject()) }),
      save: async (projectJSON, rel) => {
        files.set(rel, JSON.parse(projectJSON) as SchedProject);
        return { path: rel, savedAt: "10:00", duration: 0, critical: 0 };
      },
      create: async (name) => {
        files.set(NEW_REL, { ...makeEmptyProject(), name });
        list = [...list, summary(NEW_REL, name)];
        current = NEW_REL; // Go 侧：登记索引并自动切为当前
        return { rel: NEW_REL, current: NEW_REL };
      },
      projects: async () => ({ current, projects: list }),
    });
    seeded(normalizeProject(files.get(OLD)!), OLD);

    await useScheduleStore.getState().createProject("新工程");
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(NEW_REL);
    expect(st.project.name).toBe("新工程");
    expect(st.projects.some((p) => p.rel === NEW_REL && p.name === "新工程")).toBe(true);
    expect(st.selectedId).toBeNull();
    expect(st.past).toHaveLength(0);
    expect(st.syncError).toBeNull();
  });

  it("importAsProject（v4.144）：Create 切指针后内容替换+立即落盘+列表刷新；原工程文件不动", async () => {
    const NEW_REL = "进度计划/qian-zheng-tiao-zheng-2.gsched.json";
    const files = new Map<string, SchedProject>([[OLD, { ...makeSampleProject(), name: "调整1" }]]);
    let current = OLD;
    let list: ScheduleProjectSummary[] = [summary(OLD, "调整1")];
    stubScheduleSurface({
      load: async () => ({ path: current, exists: files.has(current), project: JSON.stringify(files.get(current) ?? makeEmptyProject()) }),
      save: async (projectJSON, rel) => {
        files.set(rel, JSON.parse(projectJSON) as SchedProject);
        return { path: rel, savedAt: "10:00", duration: 0, critical: 0 };
      },
      create: async (name) => {
        files.set(NEW_REL, { ...makeEmptyProject(), name });
        list = [...list, summary(NEW_REL, name)];
        current = NEW_REL; // Go 侧：登记索引并自动切为当前
        return { rel: NEW_REL, current: NEW_REL };
      },
      projects: async () => ({ current, projects: list }),
    });
    seeded(normalizeProject(files.get(OLD)!), OLD);

    const imported: SchedProject = {
      ...makeSampleProject(),
      name: "签证调整2",
      tasks: [{ id: "i1", name: "导入任务", duration: 4, level: 1, progress: 0 }],
    };
    await expect(useScheduleStore.getState().importAsProject(imported)).resolves.toBe(true);
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(NEW_REL); // 独立新文件成为当前工程
    expect(st.project.name).toBe("签证调整2"); // 内容=导入内容
    expect(st.project.tasks.some((t) => t.id === "i1")).toBe(true);
    expect(st.sync).toBe("saved"); // importProject 后已立即冲刷（不悬在防抖窗口）
    expect(files.get(NEW_REL)?.tasks.some((t) => t.id === "i1")).toBe(true); // 落盘新 rel
    expect(files.get(OLD)?.tasks.some((t) => t.id === "i1")).toBe(false); // 原工程文件未被触碰
    expect(st.projects.some((p) => p.rel === NEW_REL && p.name === "签证调整2")).toBe(true);
    expect(st.syncError).toBeNull();
  });

  it("importAsProject 名字空兜底「导入工程」：slug/文件内容/列表摘要同源此名", async () => {
    const NEW_REL = "进度计划/导入工程.gsched.json";
    const files = new Map<string, SchedProject>([[OLD, { ...makeSampleProject(), name: "调整1" }]]);
    let current = OLD;
    let list: ScheduleProjectSummary[] = [summary(OLD, "调整1")];
    stubScheduleSurface({
      load: async () => ({ path: current, exists: files.has(current), project: JSON.stringify(files.get(current) ?? makeEmptyProject()) }),
      save: async (projectJSON, rel) => {
        files.set(rel, JSON.parse(projectJSON) as SchedProject);
        return { path: rel, savedAt: "10:00", duration: 0, critical: 0 };
      },
      create: async (name) => {
        files.set(NEW_REL, { ...makeEmptyProject(), name });
        list = [...list, summary(NEW_REL, name)];
        current = NEW_REL;
        return { rel: NEW_REL, current: NEW_REL };
      },
      projects: async () => ({ current, projects: list }),
    });
    seeded(normalizeProject(files.get(OLD)!), OLD);

    await expect(useScheduleStore.getState().importAsProject({ ...makeSampleProject(), name: "  " })).resolves.toBe(true);
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(NEW_REL);
    expect(st.project.name).toBe("导入工程");
    expect(st.projects.some((p) => p.rel === NEW_REL && p.name === "导入工程")).toBe(true);
  });

  it("importAsProject Create 失败：返回 false，当前工程原状（fail-closed）", async () => {
    const files = new Map<string, SchedProject>([[OLD, { ...makeSampleProject(), name: "调整1" }]]);
    stubScheduleSurface({
      load: async () => ({ path: OLD, exists: true, project: JSON.stringify(files.get(OLD)!) }),
      save: async (projectJSON, rel) => {
        files.set(rel, JSON.parse(projectJSON) as SchedProject);
        return { path: rel, savedAt: "10:00", duration: 0, critical: 0 };
      },
      create: async () => { throw new Error("磁盘写入失败"); },
    });
    seeded(normalizeProject(files.get(OLD)!), OLD);

    await expect(useScheduleStore.getState().importAsProject({ ...makeSampleProject(), name: "另一工程" })).resolves.toBe(false);
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(OLD); // 指针未动
    expect(st.project.name).toBe("调整1"); // 工程未被替换
    expect(st.syncError).toContain("磁盘写入失败"); // 错误诚实上屏（指示器）
  });

  it("deleteProject 删当前工程：重水合到返回的 current，列表摘除被删项", async () => {
    const B = "进度计划/办公楼二期.gsched.json";
    const files = new Map<string, SchedProject>([
      [OLD, { ...makeSampleProject(), name: "将被删除" }],
      [B, { ...makeSampleProject(), name: "二期工程" }],
    ]);
    let current = OLD;
    let list: ScheduleProjectSummary[] = [summary(OLD, "将被删除"), summary(B, "二期工程")];
    stubScheduleSurface({
      load: async () => ({ path: current, exists: true, project: JSON.stringify(files.get(current)) }),
      save: async (projectJSON, rel) => {
        files.set(rel, JSON.parse(projectJSON) as SchedProject);
        return { path: rel, savedAt: "10:00", duration: 3, critical: 0 };
      },
      del: async (rel) => {
        files.delete(rel);
        list = list.filter((p) => p.rel !== rel);
        current = B; // Go 侧：删当前工程先切到剩余第一个
        return { current, projects: list };
      },
    });
    seeded(normalizeProject(files.get(OLD)!), OLD);

    await useScheduleStore.getState().deleteProject(OLD);
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(B);
    expect(st.project.name).toBe("二期工程");
    expect(st.projects.some((p) => p.rel === OLD)).toBe(false);
    expect(st.projects.some((p) => p.rel === B)).toBe(true);
  });

  it("archiveProject 归档非当前工程：列表更新，指针与工程内容不动", async () => {
    const C = "进度计划/旧年计划.gsched.json";
    const a = normalizeProject({ ...makeSampleProject(), name: "原工程" });
    let list: ScheduleProjectSummary[] = [summary(OLD, "原工程"), summary(C, "旧年计划")];
    stubScheduleSurface({
      load: async () => ({ path: OLD, exists: true, project: JSON.stringify(a) }),
      save: async (_projectJSON, rel) => ({ path: rel, savedAt: "10:00", duration: 3, critical: 0 }),
      archive: async (rel, archived) => {
        list = list.map((p) => (p.rel === rel ? { ...p, archived } : p));
        return { current: OLD, projects: list };
      },
    });
    seeded(a, OLD);

    await useScheduleStore.getState().archiveProject(C, true);
    const st = useScheduleStore.getState();
    expect(st.currentPath).toBe(OLD);
    expect(st.project).toBe(a);
    expect(st.projects.find((p) => p.rel === C)?.archived).toBe(true);
    expect(st.projects.find((p) => p.rel === OLD)?.archived).toBe(false);
  });

  it("refreshProjects 失败静默：不抛错且缓存保留旧值", async () => {
    const sentinel: ScheduleProjectSummary[] = [summary(OLD, "原工程")];
    stubScheduleSurface({
      load: async () => ({ path: OLD, exists: true, project: JSON.stringify(makeSampleProject()) }),
      save: async (_projectJSON, rel) => ({ path: rel, savedAt: "10:00", duration: 3, critical: 0 }),
      projects: async () => { throw new Error("索引损坏"); },
    });
    seeded(normalizeProject(makeSampleProject()), OLD);
    useScheduleStore.setState({ projects: sentinel });

    await expect(useScheduleStore.getState().refreshProjects()).resolves.toBeUndefined();
    expect(useScheduleStore.getState().projects).toBe(sentinel);
    expect(useScheduleStore.getState().syncError).toBeNull();
  });
});
