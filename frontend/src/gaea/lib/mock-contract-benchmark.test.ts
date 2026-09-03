// B 线欠账「dev mock 补受控测评五名 + 资源遥测」契约层 mock 冒烟测试：
// 模型中心「Herdsman 受控测评」段（BenchmarkSection → api/engines
// getBenchmarkList/startBenchmark/exportBenchmark/streamProbe → Go GaeaBenchmark*
// 同名 legacy 绑定面）与「本地资源占用」块（ResourceMonitor/MainLayout/
// ModuleLauncher → GetModelMonitor）。v4.56 走查发现 ?mock=1 下 mock 缺这批绑定：
// 测评段 load() 报「无法连接 Herdsman 测评接口」、资源块报「资源加载失败」。
// 锁定 mock 契约：每名存在性 + 返回形状关键键，键名与消费方解构字段一一对应
// （BenchmarkSection 读 status/summary/cases、probeBadge 读 ok/error、
// ResourceMonitor 读 engines/stats/comfyRunning 三键）。
//
// 契约原则与 mock-contract-herdsman 同源：
//   · 查询类（List/Detail）中性空态（空列表/零计数），绝不 throw——load() 的
//     Promise.all 里 rejected 会 message.warning 报错横幅；
//   · 动作类诚实失败：Start/Export 的 Go 契约为 (string, error)，失败即 rejected
//     promise（消费方 catch → message.error）；StreamProbe 形状自带 ok/error 键，
//     返回 ok:false 而非 throw（probeBadge 对 !ok 渲染「✗ {error}」红标）。
//
// 说明：六名与 Herdsman 族同为 legacy 绑定面（bridge.ts LegacySurfaceNames
// 显式排除，AppBindings 不消费），故这里经 bridge 单例 app 以窄化视图同名调用
// ——与 window.go.app.App 兼容代理的路由方法名一致。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** legacy 绑定面窄化视图：仅暴露受控测评五名 + 资源遥测（mock 实现见 mock/model.ts）。 */
const benchmark = app as unknown as {
  GaeaBenchmarkList(): Promise<unknown[]>;
  GaeaBenchmarkStart(req: unknown): Promise<string>;
  GaeaBenchmarkDetail(id: string): Promise<{
    id: string;
    created_at: string;
    finished_at?: string;
    status: string;
    config: Record<string, unknown>;
    summary: {
      total_cases: number;
      succeeded: number;
      failed: number;
      canceled: number;
      avg_duration_ms: number;
      avg_ttft_ms: number;
      avg_tps: number;
    };
    cases: unknown[];
  }>;
  GaeaBenchmarkExport(id: string, dir: string): Promise<string>;
  GaeaBenchmarkStreamProbe(model: string): Promise<{
    model: string;
    ok: boolean;
    ttft_ms: number;
    chunks: number;
    tokens: number;
    duration_ms: number;
    max_gap_ms: number;
    avg_gap_ms: number;
    completed: boolean;
    interrupted: boolean;
    error?: string;
  }>;
  GetModelMonitor(): Promise<{
    engines?: unknown[];
    stats?: Record<string, unknown>;
    comfyRunning?: boolean;
  }>;
};

/** 把 rejected promise 归一为捕获到的错误值（供逐键断言 message）。 */
async function catchRejection(p: Promise<unknown>): Promise<unknown> {
  return p.then(
    () => null,
    (err: unknown) => err,
  );
}

describe("mock 契约 · 受控测评方法族 + 资源遥测（模型中心测评段/资源块）", () => {
  it("六名均存在于 mock 绑定面（契约：可调用、不 undefined）", () => {
    const names = [
      "GaeaBenchmarkList",
      "GaeaBenchmarkStart",
      "GaeaBenchmarkDetail",
      "GaeaBenchmarkExport",
      "GaeaBenchmarkStreamProbe",
      "GetModelMonitor",
    ] as const;
    for (const name of names) {
      expect(typeof benchmark[name], `${name} 应为 function`).toBe("function");
    }
  });

  it("GaeaBenchmarkList 中性空态：空数组（BenchmarkSection 走「暂无测评记录」EmptyState，不报错）", async () => {
    await expect(benchmark.GaeaBenchmarkList()).resolves.toHaveLength(0);
  });

  it("GaeaBenchmarkDetail 零计数明细：cases 空列表 + summary 全零 + status 非成功语义", async () => {
    const d = await benchmark.GaeaBenchmarkDetail("mock-run");
    expect(d.id).toBe("mock-run");
    expect(Array.isArray(d.cases)).toBe(true);
    expect(d.cases).toHaveLength(0);
    expect(d.summary.total_cases).toBe(0);
    expect(d.summary.succeeded).toBe(0);
    expect(d.summary.failed).toBe(0);
    // 非空 status 且不落入 succeeded/running/pending 任何「有数据」语义，
    // 消费方 statusTone 对未知字串落 neutral 灰标。
    expect(d.status.length).toBeGreaterThan(0);
    expect(["succeeded", "running", "pending"]).not.toContain(d.status);
    expect(typeof d.config).toBe("object");
  });

  it("GaeaBenchmarkStart 诚实拒绝：rejected promise + 非空 message（消费方 catch → message.error，不假成功）", async () => {
    const err = await catchRejection(benchmark.GaeaBenchmarkStart({}));
    expect(err).toBeInstanceOf(Error);
    expect((err as Error).message.length).toBeGreaterThan(0);
  });

  it("GaeaBenchmarkExport 诚实拒绝：rejected promise + 非空 message（导出按钮 catch → message.error）", async () => {
    const err = await catchRejection(benchmark.GaeaBenchmarkExport("mock-run", "~/tmp"));
    expect(err).toBeInstanceOf(Error);
    expect((err as Error).message.length).toBeGreaterThan(0);
  });

  it("GaeaBenchmarkStreamProbe 诚实失败形状：ok:false + completed/interrupted false + error 非空（probeBadge 渲染 ✗ 红标）", async () => {
    const r = await benchmark.GaeaBenchmarkStreamProbe("mock-model");
    expect(r.model).toBe("mock-model");
    expect(r.ok).toBe(false);
    expect(r.completed).toBe(false);
    expect(r.interrupted).toBe(false);
    // error 必填非空：红标文案直接取 r.error
    expect(typeof r.error).toBe("string");
    expect((r.error as string).length).toBeGreaterThan(0);
    // 零指标：无后端时不编造 TTFT/分块数值
    expect(r.ttft_ms).toBe(0);
    expect(r.chunks).toBe(0);
    expect(r.tokens).toBe(0);
  });

  it("GetModelMonitor 三键齐备的中性空态：engines 空列表 + stats 对象 + comfyRunning false（computeResourceSnapshot 全零快照）", async () => {
    const m = await benchmark.GetModelMonitor();
    expect(Array.isArray(m.engines)).toBe(true);
    expect(m.engines).toHaveLength(0);
    expect(m.stats).toEqual({}); // 无读数：不编造 CPU/显存数值，快照推导为 0%
    expect(m.comfyRunning).toBe(false);
  });

  it("GetEngines 空列表空态：引擎管理段走既有空态，不再报 is-not-a-function 横幅", async () => {
    const list = await (benchmark as unknown as { GetEngines(): Promise<unknown[]> }).GetEngines();
    expect(Array.isArray(list)).toBe(true);
    expect(list).toHaveLength(0);
  });
});
