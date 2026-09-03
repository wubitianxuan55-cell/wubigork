import { describe, expect, it } from "vitest";
import { normalizeBrowserUrl } from "./browserPolicy";

describe("browserPolicy 地址栏策略", () => {
  it("裸域名补 https", () => {
    expect(normalizeBrowserUrl("example.com")).toEqual({ kind: "ok", url: "https://example.com/" });
  });
  it("http/https 放行", () => {
    expect(normalizeBrowserUrl("http://example.com/a").kind).toBe("ok");
  });
  it("危险 scheme 拒绝", () => {
    expect(normalizeBrowserUrl("javascript:alert(1)")).toEqual({ kind: "blocked", reason: "scheme" });
    expect(normalizeBrowserUrl("file:///C:/x")).toEqual({ kind: "blocked", reason: "scheme" });
  });
  it("loopback 拒绝", () => {
    expect(normalizeBrowserUrl("http://127.0.0.1:8080")).toEqual({ kind: "blocked", reason: "loopback" });
    expect(normalizeBrowserUrl("http://localhost:5173")).toEqual({ kind: "blocked", reason: "loopback" });
  });
  it("空/乱输入 invalid", () => {
    expect(normalizeBrowserUrl("   ")).toEqual({ kind: "invalid" });
  });
});
