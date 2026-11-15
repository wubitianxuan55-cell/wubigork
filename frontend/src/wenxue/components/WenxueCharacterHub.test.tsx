import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WenxueCharacterHub } from "./WenxueCharacterHub";

// 数据面 mock：角色/ 目录两卡（林晚全卡 + 空骨架卡），写读闭环按 CAS 语义。
const cards: Record<string, { text: string; mtimeMs: number }> = {
  "角色/林晚.md": { text: "# 林晚\n\n## 设定\n\n法医，32 岁。\n\n## 当前状态\n\n第 3 章后停职。", mtimeMs: 1000 },
  "角色/陆则.md": { text: "# 陆则\n\n（待补）", mtimeMs: 2000 },
};

vi.mock("../lib/bridge", () => ({
  app: {
    ListDir: vi.fn(async (rel: string) => {
      if (rel === "") return [{ name: "角色", isDir: true, size: 0 }];
      return Object.keys(cards).map((p) => ({ name: p.split("/")[1], isDir: false, size: cards[p].text.length }));
    }),
    GaeaReadTextFile: vi.fn(async (rel: string) => ({
      path: rel, text: cards[rel].text, size: cards[rel].text.length,
      mtimeMs: cards[rel].mtimeMs, encoding: "utf-8", eol: "LF",
    })),
    GaeaWriteWorkspaceFile: vi.fn(async (rel: string, content: string, expectedMtimeMs: number) => {
      if (cards[rel] && cards[rel].mtimeMs !== expectedMtimeMs) {
        throw new Error("stale");
      }
      cards[rel] = { text: content, mtimeMs: (cards[rel]?.mtimeMs ?? 0) + 1 };
      return { path: rel, size: content.length, mtimeMs: cards[rel].mtimeMs };
    }),
    GaeaDeleteWorkspacePaths: vi.fn(async () => undefined),
  },
}));

import { app } from "../lib/bridge";

describe("WenxueCharacterHub", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("列出角色目录的角色卡（名字/字数/摘要来自 md 卡）", async () => {
    render(<WenxueCharacterHub />);
    await waitFor(() => expect(screen.getByTestId("wenxue-char-card-林晚")).toBeTruthy());
    expect(screen.getByTestId("wenxue-char-card-陆则")).toBeTruthy();
    // 与工具集同源的说明条在位
    expect(screen.getByText(/wenxue_character_list\/read\/write/)).toBeTruthy();
  });

  it("点卡进入编辑，保存走 CAS 写（GaeaWriteWorkspaceFile→WenxueWriteWorkspaceFile 同源 md 卡）", async () => {
    render(<WenxueCharacterHub />);
    await waitFor(() => screen.getByTestId("wenxue-char-card-林晚"));
    fireEvent.click(screen.getByTestId("wenxue-char-card-林晚"));
    const editor = await screen.findByRole("textbox", { name: /编辑角色卡 角色\/林晚.md/ });
    fireEvent.change(editor, { target: { value: cards["角色/林晚.md"].text + "\n新增：左手机会有疤。" } });
    fireEvent.click(screen.getByText("保存（CAS）"));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    expect(vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls[0][0]).toBe("角色/林晚.md");
    expect(String(vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls[0][1])).toContain("左手机会有疤");
  });

  it("新建角色写六节骨架卡", async () => {
    render(<WenxueCharacterHub />);
    await waitFor(() => screen.getByTestId("wenxue-char-card-林晚"));
    fireEvent.click(screen.getByText("新建角色"));
    fireEvent.change(screen.getByLabelText("角色名"), { target: { value: "苏眠" } });
    fireEvent.click(screen.getByRole("button", { name: /创\s*建/ }));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    const [rel, content] = vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls.at(-1)!;
    expect(rel).toBe("角色/苏眠.md");
    expect(String(content)).toContain("# 苏眠");
    expect(String(content)).toContain("## 当前状态");
  });
});
