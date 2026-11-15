import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WenxueSettingsHub } from "./WenxueSettingsHub";

// 数据面 mock：设定/ 目录两文档（世界观全卡 + 空骨架）+ 时间线/ 一文件
// （合法锚+中文序号+无锚三类行）；写读闭环按 CAS 语义。
const docs: Record<string, { text: string; mtimeMs: number }> = {
  "设定/世界观.md": { text: "# 世界观\n\n## 格局\n\n九州三域。\n\n- 灵气自东来\n- 魔物据北荒", mtimeMs: 1000 },
  "设定/规则.md": { text: "# 规则\n\n（待补）", mtimeMs: 2000 },
  "时间线/时间线.md": { text: "# 时间线\n\n- 第3章 · 主角入门派\n- 无锚事件行\n- 第十二章 · 中文序号\n- 第1章 · 前情", mtimeMs: 3000 },
};

const requestText = vi.fn();

vi.mock("../lib/store", () => ({
  useComposerInsertStore: {
    getState: () => ({ requestText }),
  },
}));

vi.mock("../lib/bridge", () => ({
  app: {
    ListDir: vi.fn(async (rel: string) => {
      if (rel === "") {
        return [{ name: "设定", isDir: true, size: 0 }, { name: "时间线", isDir: true, size: 0 }];
      }
      return Object.keys(docs)
        .filter((p) => p.startsWith(`${rel}/`))
        .map((p) => ({ name: p.split("/")[1], isDir: false, size: docs[p].text.length }));
    }),
    GaeaReadTextFile: vi.fn(async (rel: string) => ({
      path: rel, text: docs[rel].text, size: docs[rel].text.length,
      mtimeMs: docs[rel].mtimeMs, encoding: "utf-8", eol: "LF",
    })),
    GaeaWriteWorkspaceFile: vi.fn(async (rel: string, content: string, expectedMtimeMs: number) => {
      if (docs[rel] && docs[rel].mtimeMs !== expectedMtimeMs) {
        throw new Error("stale");
      }
      docs[rel] = { text: content, mtimeMs: (docs[rel]?.mtimeMs ?? 0) + 1 };
      return { path: rel, size: content.length, mtimeMs: docs[rel].mtimeMs };
    }),
    GaeaDeleteWorkspacePaths: vi.fn(async () => undefined),
  },
}));

import { app } from "../lib/bridge";

describe("WenxueSettingsHub", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("两组列表：设定文档与时间线分组可见（名字来自首行标题）", async () => {
    render(<WenxueSettingsHub />);
    await waitFor(() => expect(screen.getByTestId("wenxue-settings-doc-世界观")).toBeTruthy());
    expect(screen.getByTestId("wenxue-settings-doc-规则")).toBeTruthy();
    expect(screen.getByTestId("wenxue-settings-doc-时间线")).toBeTruthy();
    // 同源说明条 + 落盘自动刷新提示（两处均在位）
    expect(screen.getByText(/与 agent 的 write_file/)).toBeTruthy();
    expect(screen.getByText(/wenxue_timeline_append/)).toBeTruthy();
  });

  it("设定文档：轻渲染阅读态 + 编辑态 CAS 保存", async () => {
    render(<WenxueSettingsHub />);
    await waitFor(() => screen.getByTestId("wenxue-settings-doc-世界观"));
    fireEvent.click(screen.getByTestId("wenxue-settings-doc-世界观"));
    const doc = await screen.findByTestId("wenxue-settings-doc");
    expect(doc.textContent).toContain("九州三域。");
    expect(doc.textContent).toContain("灵气自东来");
    fireEvent.click(screen.getByRole("button", { name: /编辑/ }));
    const editor = await screen.findByRole("textbox", { name: /编辑设定 设定\/世界观.md/ });
    fireEvent.change(editor, { target: { value: docs["设定/世界观.md"].text + "\n南疆有瘴。" } });
    fireEvent.click(screen.getByRole("button", { name: /保存（CAS）/ }));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    const [rel, content] = vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls.at(-1)!;
    expect(rel).toBe("设定/世界观.md");
    expect(String(content)).toContain("南疆有瘴。");
  });

  it("时间线：结构化事件视图按章序（未锚最前），中文序号如实提示", async () => {
    render(<WenxueSettingsHub />);
    await waitFor(() => screen.getByTestId("wenxue-settings-doc-时间线"));
    fireEvent.click(screen.getByTestId("wenxue-settings-doc-时间线"));
    const tl = await screen.findByTestId("wenxue-settings-timeline");
    const text = tl.textContent ?? "";
    // 未锚组排最前、章号升序：无锚 → 第1章 → 第3章；中文序号行保留并附提示
    expect(text.indexOf("无锚事件行")).toBeGreaterThan(-1);
    expect(text.indexOf("无锚事件行")).toBeLessThan(text.indexOf("前情"));
    expect(text.indexOf("前情")).toBeLessThan(text.indexOf("主角入门派"));
    expect(text).toContain("第1章");
    expect(text).toContain("中文序号不可锚定");
    expect(text).toContain("第十二章 · 中文序号");
  });

  it("AI 生成预设：Composer 注入生成指令（目标文件+摸底要求）", async () => {
    render(<WenxueSettingsHub />);
    await waitFor(() => screen.getByTestId("wenxue-settings-doc-世界观"));
    fireEvent.click(screen.getByTestId("wenxue-settings-gen"));
    fireEvent.click(screen.getByTestId("wenxue-settings-gen-worldview"));
    await waitFor(() => expect(requestText).toHaveBeenCalled());
    const prompt = String(requestText.mock.calls[0][0]);
    expect(prompt).toContain("设定/世界观.md");
    expect(prompt).toContain("write_file");
    expect(prompt).toContain("wenxue_chapter_list");
    expect(prompt).toContain("wenxue_character_list");
  });

  it("agent 落盘即见：filetree 通道变更落在设定/时间线目录 → 自动重载", async () => {
    const handlers: Array<(data: unknown) => void> = [];
    (window as unknown as { runtime: unknown }).runtime = {
      EventsOn: (_event: string, cb: (data: unknown) => void) => {
        handlers.push(cb);
        return () => {};
      },
    };
    try {
      render(<WenxueSettingsHub />);
      await waitFor(() => screen.getByTestId("wenxue-settings-doc-世界观"));
      expect(handlers.length).toBeGreaterThan(0);
      docs["设定/势力.md"] = { text: "# 势力\n\n北盟。", mtimeMs: 4000 };
      for (const cb of handlers) cb({ paths: ["设定/势力.md"] });
      await waitFor(() => expect(screen.getByTestId("wenxue-settings-doc-势力")).toBeTruthy());
    } finally {
      delete (window as unknown as { runtime?: unknown }).runtime;
    }
  });

  it("新建空白文档写标题骨架", async () => {
    render(<WenxueSettingsHub />);
    await waitFor(() => screen.getByTestId("wenxue-settings-doc-世界观"));
    fireEvent.click(screen.getByTestId("wenxue-settings-new"));
    fireEvent.change(screen.getByLabelText("文档名"), { target: { value: "地理" } });
    fireEvent.click(screen.getByRole("button", { name: /创\s*建/ }));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    const [rel, content] = vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls.at(-1)!;
    expect(rel).toBe("设定/地理.md");
    expect(String(content)).toContain("# 地理");
  });
});
