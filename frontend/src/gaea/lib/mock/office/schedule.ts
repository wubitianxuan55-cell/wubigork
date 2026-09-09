// mock/office/schedule.ts — 进度计划 mock 域（P4 结构刀2 拆分自 mock/office.ts）。
// 多工程走查态（v4.139 #15：原 v4.113 单文件会话内存改造为
// Map<rel, 计划 JSON> + 当前指针）——「每文件一工程」（设计
// docs/gaea-schedule-multi-project-design-2026-09.md §3.1）：板块 Save 落
// 当前指针文件、Load 读当前指针；多工程管理（Projects/Open/Create/Archive/
// Delete）在同一会话内存上演进（不落盘，刷新即复位，与真机索引语义同构）。
// 设计说明：mockScheduleCurrent 为可变 let 且 ScheduleProjectCreate/Delete 与
// mockScheduleFile.open 直接重赋值——TS 禁止对导入绑定重赋值（TS2632），故
// 进度计划状态/辅助函数与 Schedule* 方法必须聚合在本文件；调试钩子
// window.__mockScheduleFile 亦在此（仅 mock 存在，真机无此全局）。
import { computeCpm } from "../../../../schedule/cpm";
import { planFinishOf } from "../../../../schedule/deadline";
import { makeEmptyProject } from "../../../../schedule/sample";
import type { OfficeMethods } from "./types";

const SCHEDULE_DEFAULT_REL = "进度计划/当前计划.gsched.json";
/** 工程文件内容（rel → 计划 JSON；文件=权威，缺「当前计划」时板块走迁移分支） */
export const mockScheduleFiles = new Map<string, string>();
/** 注册表元数据（索引的 mock 形态：归档标记 + 更新时间；可由内容重建，非权威） */
interface MockScheduleMeta {
  archived: boolean;
  updatedAt: string;
}
const mockScheduleMeta = new Map<string, MockScheduleMeta>();
/** 当前工程指针（真机=Go 侧索引 current；mock=会话内存变量） */
let mockScheduleCurrent = SCHEDULE_DEFAULT_REL;

function mockScheduleNow(): string {
  return new Date().toISOString();
}

/** 落盘+登记（updatedAt 刷新；归档标记保留——归档只标记不改文件） */
function mockScheduleRegister(rel: string, raw: string): void {
  mockScheduleFiles.set(rel, raw);
  mockScheduleMeta.set(rel, { archived: mockScheduleMeta.get(rel)?.archived ?? false, updatedAt: mockScheduleNow() });
}

/** 工程名（name 必须真）：文件内 project.name 优先，坏 JSON 回落文件名去后缀 */
function mockScheduleNameOf(rel: string, raw: string | undefined): string {
  if (raw !== undefined) {
    try {
      const name = (JSON.parse(raw) as { name?: unknown }).name;
      if (typeof name === "string" && name.trim() !== "") return name;
    } catch { /* 坏 JSON 由真实 Go 侧拒绝，mock 回落文件名 */ }
  }
  return rel.split("/").pop()?.replace(/\.gsched\.json$/i, "") ?? rel;
}

/** 任务数摘要（简化口径：JSON 里 tasks 数组长度，与真机 Load+Analyze 逐文件同义） */
function mockScheduleTaskCount(raw: string | undefined): number {
  if (raw === undefined) return 0;
  try {
    const tasks = (JSON.parse(raw) as { tasks?: unknown }).tasks;
    return Array.isArray(tasks) ? tasks.length : 0;
  } catch { return 0; }
}

/** 索引视图（GaeaScheduleProjects 同形）：当前工程置顶，其余按 rel 稳定排序。 */
function mockScheduleSummaries(): {
  current: string;
  projects: {
    rel: string;
    name: string;
    archived: boolean;
    duration: number;
    taskCount: number;
    ok: boolean;
    updatedAt: string;
  }[];
} {
  const rels = [...new Set([...mockScheduleMeta.keys(), ...mockScheduleFiles.keys()])];
  rels.sort((a, b) => (a === mockScheduleCurrent ? -1 : b === mockScheduleCurrent ? 1 : a.localeCompare(b)));
  return {
    current: mockScheduleCurrent,
    projects: rels.map((rel) => {
      const raw = mockScheduleFiles.get(rel);
      // 摘要诚实口径：有内容才算 CPM（ok/duration 与板块同引擎纯函数）；
      // 未落盘/坏 JSON 给 0/false 占位，name 永远真实。
      let duration = 0;
      let ok = false;
      if (raw !== undefined) {
        try {
          const p = JSON.parse(raw) as Parameters<typeof planFinishOf>[0] & {
            tasks: Parameters<typeof computeCpm>[0];
            links: Parameters<typeof computeCpm>[1];
          };
          const r = computeCpm(p.tasks, p.links, { planFinish: planFinishOf(p), calendar: p.calendar, startDate: p.startDate });
          duration = r.duration;
          ok = r.ok;
        } catch { /* 占位 0/false */ }
      }
      return {
        rel,
        name: mockScheduleNameOf(rel, raw),
        archived: mockScheduleMeta.get(rel)?.archived ?? false,
        duration,
        taskCount: mockScheduleTaskCount(raw),
        ok,
        updatedAt: mockScheduleMeta.get(rel)?.updatedAt ?? "",
      };
    }),
  };
}

// 走查钩子：模拟 agent 外部写计划文件（schedule_apply 同语义），再 dispatch
// window focus 事件即可驱动板块轮询回读。仅 mock 存在，真机无此全局。
// v4.139 #15 向后兼容：load/save 缺省作用当前指针文件（与单文件时代等价）；
// 新增 list()/open(rel) 便于多工程走查（列表 / 切指针）。
const mockScheduleFile = {
  load: () => ({
    path: mockScheduleCurrent,
    exists: mockScheduleFiles.has(mockScheduleCurrent),
    project: mockScheduleFiles.get(mockScheduleCurrent) ?? "",
  }),
  save: (projectJSON: string, rel = "") => {
    const target = rel !== "" ? rel : mockScheduleCurrent;
    mockScheduleRegister(target, projectJSON);
    return { path: target, savedAt: "" };
  },
  list: () => mockScheduleSummaries(),
  open: (rel: string) => {
    if (!mockScheduleFiles.has(rel)) throw new Error(`工程不存在：${rel}`);
    mockScheduleCurrent = rel;
    return { current: mockScheduleCurrent };
  },
};
declare global {
  interface Window {
    __mockScheduleFile?: typeof mockScheduleFile;
  }
}
if (typeof window !== "undefined") window.__mockScheduleFile = mockScheduleFile;

// ── 进度计划方法（Schedule*：会话内存索引演进同源，v4.139 #15 刀1+刀2）──
export const scheduleMethods = {
  async ScheduleLoad() {
    // 走查态：读当前指针文件（exists=false → 板块走迁移分支把 localStorage/
    // 样例上「文件」）；多工程下指针由 ScheduleProjectOpen/Create 切换。
    const f = mockScheduleFile.load();
    return { ...f, project: f.project };
  },
  async ScheduleExportXlsx(projectJSON: string) {
    // 浏览器 mock 无 excelize：JSON 直通模拟 xlsx 往返（与真实导入同构）
    return btoa(unescape(encodeURIComponent(projectJSON)));
  },
  async ScheduleImportXlsx(xlsxBase64: string) {
    return decodeURIComponent(escape(atob(xlsxBase64)));
  },
  async ScheduleImportMpp(_mppBase64: string) {
    // 浏览器 mock 无 Go 侧 MPP 解析器：诚实报错（真实壳走 GaeaScheduleImportMpp）
    throw new Error("MPP 导入需在桌面端使用（浏览器预览态无二进制解析器）");
  },
  async ScheduleSave(projectJSON: string, rel = "") {
    // v4.139 #15：第二参 rel（空串=当前指针文件）——与真机 GaeaScheduleSave
    // 可选 rel 同构（板块保存显式带装载时的工程文件，防切换竞态）。
    const saved = mockScheduleFile.save(projectJSON, rel);
    // mock 也走真实 CPM 纯函数：保存回执的工期/关键数与板块口径一致
    let duration = 0;
    let critical = 0;
    try {
      const p = JSON.parse(projectJSON) as Parameters<typeof planFinishOf>[0] & {
        tasks: Parameters<typeof computeCpm>[0];
        links: Parameters<typeof computeCpm>[1];
      };
      const r = computeCpm(p.tasks, p.links, { planFinish: planFinishOf(p), calendar: p.calendar, startDate: p.startDate });
      duration = r.duration;
      critical = Object.values(r.rows).filter((x) => x.critical).length;
    } catch { /* 坏 JSON 由真实 Go 侧拒绝，mock 宽松回 0 */ }
    return {
      path: saved.path,
      savedAt: new Date().toTimeString().slice(0, 5),
      duration,
      critical,
    };
  },
  // ── 进度计划多工程（v4.139 #15 刀1+刀2）：会话内存索引演进 ──
  async ScheduleProjects() {
    return mockScheduleSummaries();
  },
  async ScheduleProjectOpen(rel: string) {
    return mockScheduleFile.open(rel);
  },
  async ScheduleProjectCreate(name: string) {
    const trimmed = name.trim();
    if (trimmed === "") throw new Error("工程名不能为空");
    // rel=进度计划/<名字>.gsched.json；同名加序号（Go 侧安全 slug 同语义，
    // 此后文件名不改——agent 显式 path 引用稳定性优先）。
    let rel = `进度计划/${trimmed}.gsched.json`;
    for (let n = 2; mockScheduleFiles.has(rel); n++) rel = `进度计划/${trimmed}-${n}.gsched.json`;
    const seed = makeEmptyProject();
    seed.name = trimmed;
    mockScheduleRegister(rel, JSON.stringify(seed));
    mockScheduleCurrent = rel; // 新建即自动切换（设计 §3.4）
    return { rel, current: mockScheduleCurrent };
  },
  async ScheduleProjectArchive(rel: string, archived: boolean) {
    // 归档=索引标记（文件保留原位，agent 显式 path 引用不失效；可反归档）
    mockScheduleMeta.set(rel, { archived, updatedAt: mockScheduleMeta.get(rel)?.updatedAt ?? mockScheduleNow() });
    return mockScheduleSummaries();
  },
  async ScheduleProjectDelete(rel: string) {
    // 物理删（mock=内存摘除）；删当前工程级联切指针：剩余第一个，无剩余回落缺省
    mockScheduleFiles.delete(rel);
    mockScheduleMeta.delete(rel);
    if (mockScheduleCurrent === rel) {
      // 与 Go 侧同语义：切剩余第一个**未归档**工程，无剩余回落缺省
      const rest = [...mockScheduleFiles.keys()].filter((r) => !mockScheduleMeta.get(r)?.archived).sort((a, b) => a.localeCompare(b));
      mockScheduleCurrent = rest[0] ?? SCHEDULE_DEFAULT_REL;
    }
    return mockScheduleSummaries();
  },
  async ScheduleProjectCopy(rel: string, name: string) {
    // 复制（v4.145 另存为）：源文件深拷贝仅改 name 落新文件；指针不动（文件级
    // 动作，同归档——复制品由用户自行切换打开）；同名 slug 加序号同 Create 语义
    const trimmed = name.trim();
    if (trimmed === "") throw new Error("工程名不能为空");
    const src = mockScheduleFiles.get(rel);
    if (src === undefined) throw new Error(`源工程不存在：${rel}`);
    let newRel = `进度计划/${trimmed}.gsched.json`;
    for (let n = 2; mockScheduleFiles.has(newRel); n++) newRel = `进度计划/${trimmed}-${n}.gsched.json`;
    const p = JSON.parse(src) as { name: string };
    p.name = trimmed;
    mockScheduleRegister(newRel, JSON.stringify(p));
    return mockScheduleSummaries();
  },
} satisfies Partial<OfficeMethods>;