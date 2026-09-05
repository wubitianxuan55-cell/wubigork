// B 线欠账「dev mock 补 Herdsman 方法族」契约层 mock 冒烟测试：
// 模型中心「模型库 / 本地调度 / 受控测评」段（HerdsmanCatalogSection /
// SchedulingSection / BenchmarkSection → api/engines getHerdsmanCatalog 等 →
// Go Herdsman* 同名 legacy 绑定面）。v4.55 走查发现 ?mock=1 下 mock 缺绑定，
// 调度段 load() 抛「... is not a function」级运行态报错（「当前运行中的本地
// 模型不可用」）。锁定 mock 契约：每名存在性 + 返回形状关键键，键名与消费方
// 解构字段一一对应（SchedulingSection 读 running/error、HerdsmanCatalogSection
// 读 models/total/installed、runOp 读 ok/status/message）。
// v4.100 T1 起 HerdsmanModelCatalog 带图像模型走查样例（画室「模型目录」tab），
// 断言从「中性空态」改为「列表与计数一致 + 无 error」；其余各名保持原契约。
//
// 说明：各名与 Get/SetEngineFailover 同为 legacy 绑定面（bridge.ts
// LegacySurfaceNames 显式排除，AppBindings 认领待后端合入后收口），故这里经
// bridge 单例 app 以窄化视图同名调用——与 window.go.app.App 兼容代理的路由
// 方法名一致。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** legacy 绑定面窄化视图：仅暴露 Herdsman 方法族（mock 实现见 mock/model.ts）。 */
const herdsman = app as unknown as {
  HerdsmanModelCatalog(): Promise<{
    models: unknown[];
    total: number;
    installed: number;
    running: number;
    source: string;
    error?: string;
  }>;
  HerdsmanLaunchPresets(): Promise<unknown[]>;
  HerdsmanModelStats(): Promise<{
    total: number;
    since: string;
    per_model: unknown[];
    source: string;
    error?: string;
  }>;
  HerdsmanModelStart(model: string): Promise<{ ok: boolean; status: string; message: string }>;
  HerdsmanModelStop(model: string): Promise<{ ok: boolean; status: string; message: string }>;
  HerdsmanModelDownload(model: string): Promise<{ ok: boolean; status: string; message: string }>;
  HerdsmanModelUninstall(model: string): Promise<{ ok: boolean; status: string; message: string }>;
};

describe("mock 契约 · Herdsman 方法族（模型中心模型库/调度段）", () => {
  it("七名均存在于 mock 绑定面（契约：可调用、不 undefined）", () => {
    const names = [
      "HerdsmanModelCatalog",
      "HerdsmanLaunchPresets",
      "HerdsmanModelStats",
      "HerdsmanModelStart",
      "HerdsmanModelStop",
      "HerdsmanModelDownload",
      "HerdsmanModelUninstall",
    ] as const;
    for (const name of names) {
      expect(typeof herdsman[name], `${name} 应为 function`).toBe("function");
    }
  });

  it("HerdsmanModelCatalog 走查样例（v4.100 T1 画室目录视图）：列表与计数一致 + 无 error（调度段不报错）", async () => {
    // v4.100 T1 起 mock 从「中性空态」改为带图像模型样例（画室「模型目录」tab
    // 走查有料；模型名与 mock/imagehub.ts 资产样例同源）。本断言随之从
    // 「空列表 + 计数 0」改为「一致性不变量」：计数必须从列表派生（与真实后端
    // 同一不变量），且无 error 键——SchedulingSection 的 runningErr 保持 null，
    // 不渲染报错文案的原目的不变。
    const c = await herdsman.HerdsmanModelCatalog();
    expect(Array.isArray(c.models)).toBe(true);
    const rows = c.models as Array<{ installed?: boolean; running?: boolean }>;
    expect(c.total).toBe(rows.length);
    expect(c.installed).toBe(rows.filter((m) => m.installed).length);
    expect(c.running).toBe(rows.filter((m) => m.running).length);
    expect(typeof c.source).toBe("string");
    // 无 error 键：SchedulingSection 的 runningErr 才为 null，不再渲染报错文案
    expect(c.error).toBeUndefined();
  });

  it("HerdsmanLaunchPresets 返回空数组（模型卡片不渲染「启动预设」芯片）", async () => {
    await expect(herdsman.HerdsmanLaunchPresets()).resolves.toHaveLength(0);
  });

  it("HerdsmanModelStats 空聚合：total=0、per_model 空列表（统计面板不渲染）", async () => {
    const s = await herdsman.HerdsmanModelStats();
    expect(s.total).toBe(0);
    expect(Array.isArray(s.per_model)).toBe(true);
    expect(s.per_model).toHaveLength(0);
    expect(typeof s.since).toBe("string");
    expect(typeof s.source).toBe("string");
  });

  it("生命周期四操作诚实返回 ok:false（runOp 对 !ok 弹 message.error，不假成功）", async () => {
    const ops = [
      "HerdsmanModelStart",
      "HerdsmanModelStop",
      "HerdsmanModelDownload",
      "HerdsmanModelUninstall",
    ] as const;
    for (const name of ops) {
      const r = await herdsman[name]("mock-model");
      expect(r.ok, `${name} 应返回 ok:false`).toBe(false);
      expect(typeof r.status, `${name} 应带 status`).toBe("string");
      expect(r.status.length, `${name} status 非空`).toBeGreaterThan(0);
      // message 必须非空：runOp 的失败提示文案直接取 r.message
      expect(r.message.length, `${name} message 非空`).toBeGreaterThan(0);
    }
  });
});
