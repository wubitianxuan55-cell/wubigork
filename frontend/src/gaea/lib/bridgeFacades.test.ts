// bridgeFacades.test.ts — S2.3 bridge 分面运行时门控
// （docs/gaea-space-shell-design.md §7：类型级 + 运行时双保险）
import { describe, expect, it } from "vitest";
import { workApp, playApp, sharedApp } from "./bridge";

// 类型级红线：workApp 上不存在 WhisperMemories（编译错误）；
// 负向断言需绕过类型系统，仅验证运行时门控。
type AnyRecord = Record<string, unknown>;
const asAny = <T,>(v: T): AnyRecord => v as unknown as AnyRecord;

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
});
