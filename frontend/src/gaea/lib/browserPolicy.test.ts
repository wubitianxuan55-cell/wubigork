import { describe, expect, it } from "vitest";
import { classifyExternalLink, normalizeBrowserUrl } from "./browserPolicy";

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

describe("classifyExternalLink 渲染文档外链分流（1c）", () => {
  it("http/https 放行给系统浏览器", () => {
    expect(classifyExternalLink("https://example.com/a")).toEqual({ kind: "open", url: "https://example.com/a" });
    expect(classifyExternalLink("http://example.com")).toEqual({ kind: "open", url: "http://example.com/" });
  });
  it("loopback 拒（渲染文档不得探测本机服务）", () => {
    expect(classifyExternalLink("http://127.0.0.1:8080/api")).toEqual({ kind: "blocked", reason: "loopback" });
    expect(classifyExternalLink("https://localhost/x")).toEqual({ kind: "blocked", reason: "loopback" });
  });
  it("危险 scheme 拒", () => {
    expect(classifyExternalLink("javascript:alert(1)")).toEqual({ kind: "blocked", reason: "scheme" });
    expect(classifyExternalLink("file:///C:/x")).toEqual({ kind: "blocked", reason: "scheme" });
    expect(classifyExternalLink("data:text/html,x")).toEqual({ kind: "blocked", reason: "scheme" });
    expect(classifyExternalLink("vbscript:x")).toEqual({ kind: "blocked", reason: "scheme" });
  });
  it("mailto/tel 交系统处理器", () => {
    expect(classifyExternalLink("mailto:a@b.com")).toEqual({ kind: "open", url: "mailto:a@b.com" });
    expect(classifyExternalLink("tel:10086")).toEqual({ kind: "open", url: "tel:10086" });
  });
  it("相对路径/锚点/空值拒", () => {
    expect(classifyExternalLink("#anchor")).toEqual({ kind: "blocked", reason: "scheme" });
    expect(classifyExternalLink("docs/a.md")).toEqual({ kind: "blocked", reason: "scheme" });
    expect(classifyExternalLink("")).toEqual({ kind: "blocked", reason: "scheme" });
  });
});
