// mock-office-schedule-projects.test.ts — 进度计划多工程 mock 全链（v4.139 #15）
//
// 覆盖 mock/office.ts 的多工程会话内存（Map<rel, 内容> + 当前指针）：
// 既有 Save/Load 走查钩子向后兼容（缺省作用当前指针文件）、Projects 摘要
// （name 真实/taskCount=tasks 长度/CPM ok）、Create（名字→rel、自动切换）、
// Save 显式 rel 的多文件隔离、Open 切指针（不存在的 rel 拒绝）、Archive
// （标记不删文件）、Delete（物理删 + 删当前工程级联切指针）。
// 走 bridge mock 单例（jsdom 无 window.go 即 mock），文件态经
// window.__mockScheduleFile 钩子核验（与 store.sync.test.ts 同范式）。

import { describe, expect, it } from "vitest";
import { app } from "./bridge";

interface HookSummary {
  rel: string;
  name: string;
  archived: boolean;
  duration: number;
  taskCount: number;
  ok: boolean;
  updatedAt: string;
}
interface HookShape {
  load(): { path: string; exists: boolean; project: string };
  save(projectJSON: string, rel?: string): { path: string; savedAt: string };
  list(): { current: string; projects: HookSummary[] };
  open(rel: string): { current: string };
}
const hook = () =>
  (window as unknown as { __mockScheduleFile?: HookShape }).__mockScheduleFile!;

const DEFAULT_REL = "进度计划/当前计划.gsched.json";
const B2_REL = "进度计划/办公楼二期.gsched.json";

/** 最小合法计划（无搭接两条并行叶任务：CPM ok，duration=单任务工期） */
function plan(name: string, tasks: number): string {
  return JSON.stringify({
    name,
    startDate: "2026-09-07",
    tasks: Array.from({ length: tasks }, (_, i) => ({
      id: `t${i + 1}`,
      name: `任务${i + 1}`,
      duration: 2,
      level: 1,
      progress: 0,
    })),
    links: [],
  });
}

describe("进度计划多工程 mock 全链（v4.139 #15）", () => {
  it("向后兼容：走查钩子 save/load 缺省作用当前指针文件（单文件时代等价）", () => {
    expect(hook().load().exists).toBe(false);
    expect(hook().load().path).toBe(DEFAULT_REL);
    hook().save(plan("当前计划", 2));
    const f = hook().load();
    expect(f.exists).toBe(true);
    expect(f.path).toBe(DEFAULT_REL);
    expect((JSON.parse(f.project) as { name: string }).name).toBe("当前计划");
  });

  it("Projects 摘要：name 必须真、taskCount=tasks 长度、CPM ok/duration 与板块同引擎", async () => {
    const v = await app.ScheduleProjects();
    expect(v.current).toBe(DEFAULT_REL);
    const cur = v.projects.find((p) => p.rel === DEFAULT_REL)!;
    expect(cur.name).toBe("当前计划");
    expect(cur.taskCount).toBe(2);
    expect(cur.ok).toBe(true);
    expect(cur.duration).toBeGreaterThan(0);
    expect(cur.archived).toBe(false);
    expect(cur.updatedAt).not.toBe("");
  });

  it("Create：名字→rel（进度计划/<名>.gsched.json）、空工程落盘、自动切为当前", async () => {
    const created = await app.ScheduleProjectCreate("办公楼二期");
    expect(created.rel).toBe(B2_REL);
    expect(created.current).toBe(B2_REL);
    const f = hook().load();
    expect(f.path).toBe(B2_REL);
    expect(f.exists).toBe(true);
    const seeded = JSON.parse(f.project) as { name: string; tasks: unknown[] };
    expect(seeded.name).toBe("办公楼二期");
    expect(Array.isArray(seeded.tasks)).toBe(true);
    // 列表中当前工程置顶
    const v = await app.ScheduleProjects();
    expect(v.projects[0].rel).toBe(B2_REL);
    expect(v.projects.map((p) => p.rel)).toContain(DEFAULT_REL);
  });

  it("Create 同名加序号；空名拒绝", async () => {
    const dup = await app.ScheduleProjectCreate("办公楼二期");
    expect(dup.rel).toBe("进度计划/办公楼二期-2.gsched.json");
    await expect(app.ScheduleProjectCreate("   ")).rejects.toThrow();
    // 收拾掉序号工程，免影响后续用例
    await app.ScheduleProjectDelete(dup.rel);
  });

  it("Save 显式 rel：多文件互不串写（指针隔离），Load 恒读指针", async () => {
    // 显式写旧工程（指针仍指办公楼二期）
    await app.ScheduleSave(plan("当前计划-v2", 3), DEFAULT_REL);
    // 缺省写当前指针文件
    await app.ScheduleSave(plan("办公楼二期-v2", 5));
    expect(
      (JSON.parse(hook().load().project) as { name: string }).name,
    ).toBe("办公楼二期-v2");
    // 切指针后读到的就是另一份内容
    const opened = await app.ScheduleProjectOpen(DEFAULT_REL);
    expect(opened.current).toBe(DEFAULT_REL);
    const f = hook().load();
    expect(f.path).toBe(DEFAULT_REL);
    expect((JSON.parse(f.project) as { name: string }).name).toBe("当前计划-v2");
  });

  it("Open 不存在的 rel 拒绝（fail-closed）；走查钩子 open 同语义同步", async () => {
    await expect(
      app.ScheduleProjectOpen("进度计划/不存在.gsched.json"),
    ).rejects.toThrow();
    expect(() => hook().open("进度计划/不存在.gsched.json")).toThrow();
    // 钩子 list 与绑定 Projects 同源
    expect(hook().list().current).toBe(DEFAULT_REL);
  });

  it("Archive：索引标记（归档可反），文件保留原位、指针不动", async () => {
    const a = await app.ScheduleProjectArchive(DEFAULT_REL, true);
    expect(a.projects.find((p) => p.rel === DEFAULT_REL)!.archived).toBe(true);
    expect(hook().load().path).toBe(DEFAULT_REL);
    expect(hook().load().exists).toBe(true);
    const b = await app.ScheduleProjectArchive(DEFAULT_REL, false);
    expect(b.projects.find((p) => p.rel === DEFAULT_REL)!.archived).toBe(false);
  });

  it("Delete：物理删 + 删当前工程级联切指针（剩余第一个），删空回落缺省", async () => {
    const d = await app.ScheduleProjectDelete(DEFAULT_REL);
    expect(d.projects.some((p) => p.rel === DEFAULT_REL)).toBe(false);
    expect(d.current).toBe(B2_REL);
    expect(hook().load().path).toBe(B2_REL);
    // 删到空：指针回落缺省（文件缺失 → 板块首启走迁移分支）
    await app.ScheduleProjectDelete(B2_REL);
    const v = await app.ScheduleProjects();
    expect(v.current).toBe(DEFAULT_REL);
    expect(v.projects.length).toBe(0);
    expect(hook().load().exists).toBe(false);
  });

  it("Copy（v4.145 另存为）：深拷贝仅改 name 落新文件、同名加序号；指针不动", async () => {
    // 上一用例删空：指针已回落缺省，先落源文件
    hook().save(plan("当前计划", 2));
    const c = await app.ScheduleProjectCopy(DEFAULT_REL, "调整2");
    const newRel = "进度计划/调整2.gsched.json";
    expect(c.current).toBe(DEFAULT_REL); // 指针不动（文件级动作）
    expect(c.projects.some((p) => p.rel === newRel && p.name === "调整2")).toBe(true);
    hook().open(newRel); // 走查钩子切指针读副本内容
    const raw = JSON.parse(hook().load().project) as { name: string };
    expect(raw.name).toBe("调整2");
    // 同名加序号 + 源不存在拒绝 + 空名拒绝
    await app.ScheduleProjectCopy(DEFAULT_REL, "调整2");
    await app.ScheduleProjectOpen("进度计划/调整2-2.gsched.json");
    expect(hook().load().exists).toBe(true);
    await expect(app.ScheduleProjectCopy("进度计划/不存在.gsched.json", "x")).rejects.toThrow();
    await expect(app.ScheduleProjectCopy(DEFAULT_REL, "  ")).rejects.toThrow();
  });
});
