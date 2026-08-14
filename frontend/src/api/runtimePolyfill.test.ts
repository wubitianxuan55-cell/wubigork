import { describe, it, expect } from "vitest";
import { parseSSEFrame, parseSSEStream, type ParsedSSEFrame } from "./runtimePolyfill";

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
