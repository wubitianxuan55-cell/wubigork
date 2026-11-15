import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WenxueCharacterHub } from "./WenxueCharacterHub";

// 数据面 mock：角色/ 目录两卡（林晚全卡 + 空骨架卡），写读闭环按 CAS 语义。
// 角色库 mock（只读联动面）：CharacterList 返回两张库卡；写向绑定（Save/Delete/
// Import*/Sync）全部挂监视——红线断言=整个面板零调用。
const cards: Record<string, { text: string; mtimeMs: number }> = {
  "角色/林晚.md": { text: "# 林晚\n\n## 设定\n\n法医，32 岁。\n\n## 当前状态\n\n第 3 章后停职。", mtimeMs: 1000 },
  "角色/陆则.md": { text: "# 陆则\n\n（待补）", mtimeMs: 2000 },
};

const libChars = [
  {
    id: "lib-1", name: "沈青梧", kind: "custom", gender: "女", age: "28",
    tags: ["中医", "冷面"], roleType: "supporting",
    background: "祖传药铺的第三代传人。", motivation: "查清父亲失踪的真相",
    appearance: "眉眼细长，常穿青布衫。", figure: "清瘦", personality: "外冷内热，话少理清。",
    arc: "从旁观到入局", status: "Alive", notes: "", dialogueSamples: ["药不瞒价。", "先把脉，再说话。"],
  },
  { id: "lib-2", name: "陈砚", kind: "custom", roleType: "protagonist", background: "落魄书生。", status: "Missing" },
];

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
    // 角色库只读联动面（v2 新建角色导入通道）
    CharacterList: vi.fn(async () => ({ items: libChars, total: libChars.length })),
    // 红线监视：写向/登记绑定——面板任何路径都不允许触到
    CharacterSave: vi.fn(async () => { throw new Error("红线：面板不得写角色库"); }),
    CharacterDelete: vi.fn(async () => { throw new Error("红线"); }),
    CharacterImportProject: vi.fn(async () => { throw new Error("红线"); }),
    CharacterImportPreview: vi.fn(async () => { throw new Error("红线"); }),
    CharacterSyncProject: vi.fn(async () => { throw new Error("红线"); }),
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
    // 与工具集同源的说明条在位（v2 左栏）+ 回写自动刷新提示
    expect(screen.getByText(/wenxue_character_\*/)).toBeTruthy();
    expect(screen.getByText(/wenxue_character_write/)).toBeTruthy();
  });

  it("点卡进阅读态（结构化六节渲染），编辑态保存走 CAS 写", async () => {
    render(<WenxueCharacterHub />);
    await waitFor(() => screen.getByTestId("wenxue-char-card-林晚"));
    fireEvent.click(screen.getByTestId("wenxue-char-card-林晚"));
    // 阅读态：结构化节块（设定/当前状态 有内容渲染；外貌等缺节显占位）
    const detail = await screen.findByTestId("wenxue-char-detail");
    expect(detail.textContent).toContain("法医，32 岁。");
    expect(detail.textContent).toContain("第 3 章后停职。");
    expect(detail.textContent).toContain("外貌 · 未填写");
    // 进编辑态（裸文本 CAS）
    fireEvent.click(screen.getByRole("button", { name: /编辑/ }));
    const editor = await screen.findByRole("textbox", { name: /编辑角色卡 角色\/林晚.md/ });
    fireEvent.change(editor, { target: { value: cards["角色/林晚.md"].text + "\n新增：左手机会有疤。" } });
    fireEvent.click(screen.getByRole("button", { name: /保存（CAS）/ }));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    const [rel, content] = vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls.at(-1)!;
    expect(rel).toBe("角色/林晚.md");
    expect(String(content)).toContain("左手机会有疤");
  });

  it("新建空白卡写六节骨架", async () => {
    render(<WenxueCharacterHub />);
    await waitFor(() => screen.getByTestId("wenxue-char-card-林晚"));
    fireEvent.click(screen.getByTestId("wenxue-char-new"));
    fireEvent.change(screen.getByLabelText("角色名"), { target: { value: "苏眠" } });
    fireEvent.click(screen.getByRole("button", { name: /创\s*建/ }));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    const [rel, content] = vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls.at(-1)!;
    expect(rel).toBe("角色/苏眠.md");
    expect(String(content)).toContain("# 苏眠");
    expect(String(content)).toContain("## 当前状态");
  });

  it("从角色库导入：字段映射六节落工作空间卡；红线=只读（写向绑定零调用）", async () => {
    render(<WenxueCharacterHub />);
    await waitFor(() => screen.getByTestId("wenxue-char-card-林晚"));
    fireEvent.click(screen.getByTestId("wenxue-char-new"));
    // 切到「从角色库导入」tab（antd tab：role=tab）
    fireEvent.click(screen.getByRole("tab", { name: /从角色库导入/ }));
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));
    await waitFor(() => expect(screen.getByTestId("wenxue-lib-char-沈青梧")).toBeTruthy());
    expect(screen.getByTestId("wenxue-lib-char-陈砚")).toBeTruthy();
    // 只调了只读查询
    expect(vi.mocked(app.CharacterList)).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("wenxue-lib-char-沈青梧"));
    fireEvent.click(screen.getByTestId("wenxue-lib-import"));
    await waitFor(() => expect(vi.mocked(app.GaeaWriteWorkspaceFile)).toHaveBeenCalled());
    const [rel, content] = vi.mocked(app.GaeaWriteWorkspaceFile).mock.calls.at(-1)!;
    expect(rel).toBe("角色/沈青梧.md");
    const md = String(content);
    expect(md).toContain("# 沈青梧");
    expect(md).toContain("## 设定");
    expect(md).toContain("祖传药铺的第三代传人。");
    expect(md).toContain("动机：查清父亲失踪的真相");
    expect(md).toContain("## 外貌");
    expect(md).toContain("眉眼细长，常穿青布衫。");
    expect(md).toContain("## 口头禅");
    expect(md).toContain("- 药不瞒价。");
    expect(md).toContain("## 当前状态");
    expect(md).toContain("状态：存活");
    // 红线：面板全程零写向调用
    expect(vi.mocked(app.CharacterSave)).not.toHaveBeenCalled();
    expect(vi.mocked(app.CharacterDelete)).not.toHaveBeenCalled();
    expect(vi.mocked(app.CharacterImportProject)).not.toHaveBeenCalled();
    expect(vi.mocked(app.CharacterSyncProject)).not.toHaveBeenCalled();
  });

  it("agent 回写即见：filetree 通道变更路径落在角色目录 → 列表自动重载", async () => {
    // subscribe 退化形态：无 wails runtime 时返回 noop——这里桩 window.runtime
    // 捕获订阅回调，模拟 agent wenxue_character_write 落盘后的通道推送。
    const handlers: Array<(data: unknown) => void> = [];
    (window as unknown as { runtime: unknown }).runtime = {
      EventsOn: (_event: string, cb: (data: unknown) => void) => {
        handlers.push(cb);
        return () => {};
      },
    };
    try {
      render(<WenxueCharacterHub />);
      await waitFor(() => screen.getByTestId("wenxue-char-card-林晚"));
      expect(handlers.length).toBeGreaterThan(0);
      // 模拟 agent 用 wenxue_character_write 新写了一张卡 → filetree 通道推送
      cards["角色/白澈.md"] = { text: "# 白澈\n\n## 设定\n\n渡船人。", mtimeMs: 3000 };
      for (const cb of handlers) cb({ paths: ["角色/白澈.md"] });
      await waitFor(() => expect(screen.getByTestId("wenxue-char-card-白澈")).toBeTruthy());
      // 只读联动面不被误触
      expect(vi.mocked(app.CharacterList)).not.toHaveBeenCalled();
    } finally {
      delete (window as unknown as { runtime?: unknown }).runtime;
    }
  });
});
