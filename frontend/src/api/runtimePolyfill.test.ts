import { describe, it, expect, vi, afterEach } from "vitest";
import { initRuntimePolyfill, parseSSEFrame, parseSSEStream, type ParsedSSEFrame } from "./runtimePolyfill";

describe("parseSSEFrame", () => {
  it("解析单 data 帧（无 event 字段 → message 语义）", () => {
    const frame = parseSSEFrame('data: {"kind":"text","text":"hi"}');
    expect(frame).toEqual({ event: "", data: ['{"kind":"text","text":"hi"}'] });
  });

  it("多行 data 按 SSE 规范以 \n 拼接", () => {
    const frame = parseSSEFrame('data: {"lines":\ndata: [1,2]}');
    expect(frame?.data.join('\n')).toBe('{"lines":\n[1,2]}');
  });

  it("event 字段：done 结束帧与 connected 连接帧", () => {
    expect(parseSSEFrame("event: done")).toEqual({ event: "done", data: [] });
    expect(parseSSEFrame('event: connected\ndata: {"id":"chat"}')).toEqual({
      event: "connected",
      data: ['{"id":"chat"}'],
    });
  });

  it("注释行（keep-alive）与空帧返回 null", () => {
    expect(parseSSEFrame(": keep-alive")).toBeNull();
    expect(parseSSEFrame("")).toBeNull();
  });
});

describe("parseSSEStream", () => {
  it("跨 chunk 边界拼接帧并逐帧产出（done 帧、keep-alive 注释帧）", async () => {
    const chunks = [
      'data: {"a":1}\n\nevent: d',
      'one\ndata: x\n\n: keep',
      '-alive\n\n',
    ];
    const frames: ParsedSSEFrame[] = [];
    for await (const f of parseSSEStream(chunks)) frames.push(f);
    expect(frames).toEqual([
      { event: "", data: ['{"a":1}'] },
      { event: "done", data: ["x"] },
    ]);
  });

  it("CRLF 行尾与流结束 flush 未以空行收尾的残留帧", async () => {
    const frames: ParsedSSEFrame[] = [];
    for await (const f of parseSSEStream(['data: {"b":2}\r\n\r\ndata: {"c":3}'])) frames.push(f);
    expect(frames).toEqual([
      { event: "", data: ['{"b":2}'] },
      { event: "", data: ['{"c":3}'] },
    ]);
  });
});

// ── v4.350：EventsOn 返回退订 + EventsOff 只摘自己（对齐 wails v2.13 桌面语义）──

interface PolyRuntime {
  EventsOn: (e: string, cb: (...a: unknown[]) => void) => (() => void) | void;
  EventsOff: (e: string, cb?: (...a: unknown[]) => void) => void;
  EventsOnMultiple: (e: string, cb: (...a: unknown[]) => void, n: number) => (() => void) | void;
  EventsEmit: (e: string, ...a: unknown[]) => void;
}

function freshRuntime(): PolyRuntime {
  delete (window as unknown as { runtime?: unknown }).runtime;
  delete (window as unknown as { __runtime_polyfill_initialized?: boolean }).__runtime_polyfill_initialized;
  vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(null, { status: 404 }))));
  initRuntimePolyfill();
  return (window as unknown as { runtime: PolyRuntime }).runtime;
}

afterEach(() => {
  vi.unstubAllGlobals();
  delete (window as unknown as { runtime?: unknown }).runtime;
  delete (window as unknown as { __runtime_polyfill_initialized?: boolean }).__runtime_polyfill_initialized;
});

describe("runtimePolyfill 事件退订语义（v4.350）", () => {
  it("EventsOn 返回退订函数；退订后不再收事件，共享通道他人不受影响", () => {
    const rt = freshRuntime();
    const a = vi.fn();
    const b = vi.fn();
    const offA = rt.EventsOn("evt", a);
    rt.EventsOn("evt", b);
    expect(typeof offA).toBe("function");

    offA!();
    rt.EventsEmit("evt", { n: 1 });
    expect(a).not.toHaveBeenCalled();
    expect(b).toHaveBeenCalledWith({ n: 1 });
  });

  it("EventsOff(带 callback) 只摘该监听者；最后一人退订后通道静默", () => {
    const rt = freshRuntime();
    const a = vi.fn();
    const b = vi.fn();
    rt.EventsOn("chan", a);
    rt.EventsOn("chan", b);
    rt.EventsOff("chan", a);
    rt.EventsEmit("chan", 1);
    expect(a).not.toHaveBeenCalled();
    expect(b).toHaveBeenCalledWith(1);
    rt.EventsOff("chan", b);
    rt.EventsEmit("chan", 2);
    expect(b).toHaveBeenCalledTimes(1);
  });

  it("EventsOnMultiple 仍返回退订函数（既有语义保持）", () => {
    const rt = freshRuntime();
    const cb = vi.fn();
    const off = rt.EventsOnMultiple("multi", cb, -1);
    expect(typeof off).toBe("function");
    off!();
    rt.EventsEmit("multi", 1);
    expect(cb).not.toHaveBeenCalled();
  });
});
