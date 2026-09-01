import { describe, expect, it } from "vitest";
import { parseSidebarOpenResult } from "./sidebarOpen";

const okEnv = (data: unknown, code = "ok") =>
  JSON.stringify({ ok: true, success: true, code, data });

describe("sidebarOpen 解析器（v4.25 模型主动打开）", () => {
  it("非 sidebar_open 工具直接返回 null", () => {
    const res = okEnv({ kind: "file", path_abs: "C:/w/a.docx", path_rel: "a.docx" });
    expect(parseSidebarOpenResult("write_file", '{"path":"a.docx"}', res)).toBeNull();
    expect(parseSidebarOpenResult("", "{}", res)).toBeNull();
  });

  it("成功 envelope 返回 kind 与 pathRel（path_rel 优先）", () => {
    const res = okEnv({ kind: "file", path_abs: "C:/w/docs/报告.docx", path_rel: "docs/报告.docx" });
    expect(parseSidebarOpenResult("sidebar_open", '{"path":"docs/报告.docx"}', res)).toEqual({
      kind: "file",
      pathRel: "docs/报告.docx",
    });
    expect(
      parseSidebarOpenResult(
        "sidebar_open",
        '{"path":"out","kind":"directory"}',
        okEnv({ kind: "directory", path_abs: "C:/w/out", path_rel: "out" }),
      ),
    ).toEqual({ kind: "directory", pathRel: "out" });
  });

  it("path_rel 缺失时回退 path_abs，再回退 args.path", () => {
    expect(
      parseSidebarOpenResult("sidebar_open", '{"path":"a.txt"}', okEnv({ kind: "file", path_abs: "C:/w/a.txt" })),
    ).toEqual({ kind: "file", pathRel: "C:/w/a.txt" });
    expect(parseSidebarOpenResult("sidebar_open", '{"path":"a.txt"}', okEnv({ kind: "file" }))).toEqual({
      kind: "file",
      pathRel: "a.txt",
    });
  });

  it("失败 envelope（ok=false / 失败 code）返回 null", () => {
    const fail = JSON.stringify({ ok: false, success: false, code: "validation_error", error: "path 必填" });
    expect(parseSidebarOpenResult("sidebar_open", '{"path":""}', fail)).toBeNull();
    expect(
      parseSidebarOpenResult("sidebar_open", "{}", okEnv({ kind: "file", path_rel: "a.txt" }, "no_workspace")),
    ).toBeNull();
  });

  it("kind 非法或缺失返回 null", () => {
    expect(
      parseSidebarOpenResult("sidebar_open", '{"path":"a"}', okEnv({ kind: "folder", path_rel: "a" })),
    ).toBeNull();
    expect(parseSidebarOpenResult("sidebar_open", '{"path":"a"}', okEnv({ path_rel: "a" }))).toBeNull();
  });

  it("data 缺失或非对象返回 null", () => {
    expect(parseSidebarOpenResult("sidebar_open", "{}", okEnv(undefined))).toBeNull();
    expect(parseSidebarOpenResult("sidebar_open", "{}", okEnv("file"))).toBeNull();
    expect(parseSidebarOpenResult("sidebar_open", "{}", JSON.stringify({ ok: true, code: "ok" }))).toBeNull();
  });

  it("坏 JSON 一律 null 不抛", () => {
    expect(parseSidebarOpenResult("sidebar_open", "{}", "not json {")).toBeNull();
    expect(parseSidebarOpenResult("sidebar_open", "{}", "")).toBeNull();
    expect(parseSidebarOpenResult("sidebar_open", "bad args", okEnv({ kind: "file", path_rel: "a.txt" }))).toEqual({
      kind: "file",
      pathRel: "a.txt",
    });
    expect(() => parseSidebarOpenResult("sidebar_open", "{", "{")).not.toThrow();
  });

  it("无法解析出任何路径时返回 null", () => {
    expect(parseSidebarOpenResult("sidebar_open", "{}", okEnv({ kind: "file", path_abs: "  " }))).toBeNull();
  });
});
