// bridgeFacades.test.ts — S2.3 bridge 分面运行时门控
// （docs/gaea-space-shell-design.md §7：类型级 + 运行时双保险）
import { afterEach, describe, expect, it } from "vitest";
import { workApp, playApp, sharedApp } from "./bridge";

// 类型级红线：workApp 上不存在 WhisperMemories（编译错误）；
// 负向断言需绕过类型系统，仅验证运行时门控。
type AnyRecord = Record<string, unknown>;
const asAny = <T,>(v: T): AnyRecord => v as unknown as AnyRecord;

// ── FE3-06：未登记名 fail-closed 的「不触达真实绑定」证据工装 ──────────
// realApp() 在 window.go.app 存在时按方法名遍历各板块门面取值。这里塞一个
// 记录取值的探针门面：任何一次属性读取都会被计数。
type GoNs = Record<string, unknown>;
let goProbeReads: string[] = [];
function installGoProbe(): void {
  goProbeReads = [];
  const ns: GoNs = new Proxy({} as GoNs, {
    get(_t, prop) {
      goProbeReads.push(String(prop));
      return undefined;
    },
  });
  (globalThis as unknown as { window: { go?: unknown } }).window.go = {
    app: { CoreB: ns, OfficeB: ns, MemoryB: ns, CostB: ns, ModelB: ns, VoiceB: ns, ChatB: ns, NovelB: ns, ImageB: ns, CharLibB: ns },
  };
}
function uninstallGoProbe(): void {
  delete (globalThis as unknown as { window: { go?: unknown } }).window.go;
}
afterEach(() => {
  uninstallGoProbe();
});

describe("bridge 空间门面（workApp / playApp / sharedApp）", () => {
  it("workApp：work 方法可达、play 专属方法被门控（undefined）", () => {
    expect(typeof workApp.TaskList).toBe("function"); // work：任务中心
    expect(typeof workApp.XlsxPlanEdit).toBe("function"); // work：办公
    expect(asAny(workApp).WhisperMemories).toBeUndefined(); // play 专属，工位门面不可见
    expect(asAny(workApp).WhisperEpisodes).toBeUndefined();
  });

  it("playApp：play/shared 可达、work 专属方法被门控", () => {
    expect(typeof playApp.WhisperMemories).toBe("function"); // play：轻语记忆
    expect(typeof playApp.GaeaSpaceList).toBe("function"); // shared：空间枚举
    expect(asAny(playApp).XlsxPlanEdit).toBeUndefined(); // work 专属，乐园门面不可见
    expect(asAny(playApp).TaskList).toBeUndefined();
  });

  it("sharedApp：仅 shared 方法可达", () => {
    expect(typeof sharedApp.GaeaSpaceList).toBe("function");
    expect(typeof sharedApp.GaeaSpaceActive).toBe("function");
    expect(asAny(sharedApp).TaskList).toBeUndefined(); // work 专属
    expect(asAny(sharedApp).WhisperMemories).toBeUndefined(); // play 专属
  });

  it("门面不是 Promise（then 探针返回 undefined，避免 await 误判）", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    expect((workApp as any).then).toBeUndefined();
  });

  // FE3-06：未登记名 fail-closed 的运行时证据——拒绝必须发生在「解析真实绑定」之前，
  // 否则门面只是返回 undefined 的假门控（真实绑定已被读取/副作用已发生）。
  it("FE3-06：未登记名在 work/play/shared 三门面一律被拒，且不触达真实绑定", () => {
    installGoProbe();

    // 正控：已登记名（TaskList→work）确实要遍历真实板块门面取方法 —— 证明探针是
    // 活的。没有这一步，下面的负控会因「探针根本没接上」而永真，失去证据力。
    expect(asAny(workApp).TaskList).toBeUndefined(); // 探针门面不提供函数 ⇒ 解析结果 undefined
    expect(goProbeReads.length).toBeGreaterThan(0);

    // 负控：未登记名（含 Go 侧真实存在的 legacy 直调绑定）三门面一律拒绝，
    // 且一次真实绑定属性读取都不发生（fail-closed 在 resolveBinding 之前短路）。
    goProbeReads = [];
    for (const name of ["AddCustomEngine", "ApplyBranch", "GaeaBenchmarkList", "NoSuchMethodAtAll"]) {
      expect(asAny(workApp)[name]).toBeUndefined();
      expect(asAny(playApp)[name]).toBeUndefined();
      expect(asAny(sharedApp)[name]).toBeUndefined();
    }
    // 改坏锚点：bindingSpaceOf 兜底改回 "work"（或 isSharedBinding 放行未登记名）
    // → 这里会读到真实绑定，断言 FAIL。
    expect(goProbeReads).toEqual([]);
  });
});
