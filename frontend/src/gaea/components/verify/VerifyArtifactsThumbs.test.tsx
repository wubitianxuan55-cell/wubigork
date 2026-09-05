import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { VerifyArtifactsThumbs } from "./VerifyArtifactsThumbs";
import { LocaleProvider } from "../../lib/i18n";
import type { PreviewResult } from "../../lib/types";

// 桥接 mock（参考 ChangesPanel.test.tsx 的 vi.hoisted + vi.mock 模式）：
// 组件下行绑定仅 ListDir（列产物目录）与 Preview（逐页 image dataUrl + 目录
// 探测定性）。真实目录布局样例见 lib/verifyArtifacts.test.ts。
const mocks = vi.hoisted(() => ({
  listDir: vi.fn<(rel: string) => Promise<unknown>>(),
  preview: vi.fn<(rel: string) => Promise<PreviewResult>>(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    ListDir: (rel: string) => mocks.listDir(rel),
    Preview: (rel: string) => mocks.preview(rel),
  },
}));

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const img = (rel: string): PreviewResult => ({
  path: rel,
  name: rel.split("/").pop() ?? rel,
  ext: ".png",
  size: 128,
  kind: "image",
  body: "",
  dataUrl: `data:image/png;base64,${encodeURIComponent(rel)}`,
  error: "",
});

const entry = (name: string, isDir: boolean) => ({ name, isDir, size: isDir ? 0 : 128 });

// 真实布局：产物根有 before/after 两个子目录 + 两份审计 PDF；
// before 2 页、after 1 页（页数变化 → 第 2 页改前单侧 + 无此页占位）。
const ROOT = ".gaea/work/journal/verify/ev_1";

beforeEach(() => {
  mocks.listDir.mockReset();
  mocks.preview.mockReset();
});

afterEach(() => {
  try { localStorage.setItem("gaea-lang", "zh"); } catch { /* ignore */ }
});

describe("VerifyArtifactsThumbs 通道 B 逐页缩略图", () => {
  it("artifacts 为空：不渲染入口（懒加载前提，绝不空拉）", () => {
    renderT(<VerifyArtifactsThumbs artifacts="" />);
    expect(screen.queryByTestId("verify-thumbs-toggle")).toBeNull();
    expect(mocks.listDir).not.toHaveBeenCalled();
  });

  it("懒加载：未展开不拉数据；展开触发 ListDir（绝对路径已相对化）+ 逐页 Preview", async () => {
    mocks.listDir.mockImplementation(async (rel: string) => {
      if (rel === ROOT) return [entry("before.pdf", false), entry("before", true), entry("after", true)];
      if (rel === `${ROOT}/before`) return [entry("before-1.png", false), entry("before-2.png", false)];
      if (rel === `${ROOT}/after`) return [entry("after-1.png", false)];
      return [];
    });
    mocks.preview.mockImplementation(async (rel: string) => img(rel));

    renderT(<VerifyArtifactsThumbs artifacts="C:/ws/.gaea/work/journal/verify/ev_1" />);
    expect(mocks.listDir).not.toHaveBeenCalled(); // 展开前零调用

    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    await waitFor(() => expect(screen.getAllByTestId("verify-thumb-cell").length).toBeGreaterThan(0));

    // 绝对路径 → 按固定标记相对化后调用 ListDir（根 + before/ + after/）
    expect(mocks.listDir).toHaveBeenCalledWith(ROOT);
    expect(mocks.listDir).toHaveBeenCalledWith(`${ROOT}/before`);
    expect(mocks.listDir).toHaveBeenCalledWith(`${ROOT}/after`);
    // 逐页 Preview：3 张页面图各一次
    expect(mocks.preview).toHaveBeenCalledWith(`${ROOT}/before/before-1.png`);
    expect(mocks.preview).toHaveBeenCalledWith(`${ROOT}/before/before-2.png`);
    expect(mocks.preview).toHaveBeenCalledWith(`${ROOT}/after/after-1.png`);
    await waitFor(() =>
      expect(screen.getAllByTestId("verify-thumb-cell")).toHaveLength(3),
    );
  });

  it("before/after 成对呈现：页码标注 + 改前/改后 caption + 缺页「无此页」诚实占位", async () => {
    mocks.listDir.mockImplementation(async (rel: string) => {
      if (rel === ROOT) return [entry("before", true), entry("after", true)];
      if (rel === `${ROOT}/before`) return [entry("before-2.png", false), entry("before-1.png", false)];
      if (rel === `${ROOT}/after`) return [entry("after-1.png", false)];
      return [];
    });
    mocks.preview.mockImplementation(async (rel: string) => img(rel));

    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    await screen.findByAltText("第 1 页（改前）");

    // 乱序输入已按页码排序：第 1 页在前
    const rows = screen.getAllByText(/^第 \d+ 页$/);
    expect(rows.map((r) => r.textContent)).toEqual(["第 1 页", "第 2 页"]);
    // 改前 ×2（第 1/2 页）、改后 ×1（仅第 1 页）
    expect(screen.getAllByText("改前")).toHaveLength(2);
    expect(screen.getAllByText("改后")).toHaveLength(1);
    // 第 2 页改后缺失：诚实「无此页」占位（而非伪造页面图）
    expect(screen.getByTestId("verify-thumb-missing")).toBeTruthy();
    // 缩略图 src 为 Preview 返回的 dataUrl
    const firstImg = screen.getByAltText("第 1 页（改前）") as HTMLImageElement;
    await waitFor(() => expect(firstImg.getAttribute("src")).toMatch(/^data:image\/png;base64,/));
  });

  it("展开过的数据保留：收起再展开不重拉（懒加载语义）", async () => {
    mocks.listDir.mockImplementation(async (rel: string) => {
      if (rel === ROOT) return [entry("flat-page-1.png", false)];
      return [];
    });
    mocks.preview.mockImplementation(async (rel: string) => img(rel));

    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    const toggle = screen.getByTestId("verify-thumbs-toggle");
    fireEvent.click(toggle);
    await screen.findByAltText("第 1 页");
    const callsAfterFirst = mocks.listDir.mock.calls.length;
    fireEvent.click(toggle); // 收起
    fireEvent.click(toggle); // 再展开
    await waitFor(() => expect(screen.getByTestId("verify-thumbs-grid")).toBeTruthy());
    expect(mocks.listDir.mock.calls.length).toBe(callsAfterFirst);
  });

  it("诚实降级：目录不存在（Preview 探测「文件不存在」）", async () => {
    mocks.listDir.mockResolvedValue([]);
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "error",
      body: "", dataUrl: "", error: "文件不存在",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("产物目录不存在");
    expect(screen.queryByTestId("verify-thumb-cell")).toBeNull();
  });

  it("诚实降级：结构化错误码优先——ListDir 拒绝 GAEADIR_NOT_FOUND → missing（省一次 Preview 探测）", async () => {
    // 新后端（gaea_listdir.go）：缺失目录 promise reject `Error [CODE]: message`
    mocks.listDir.mockRejectedValue("Error [GAEADIR_NOT_FOUND]: 目录不存在: C:/ws/x（stat 失败）");
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "text", body: "", dataUrl: "", error: "",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("产物目录不存在");
    // 码已定性：不再发 Preview 探测
    expect(mocks.preview).not.toHaveBeenCalled();
  });

  it("诚实降级：GAEADIR_NOT_DIR（是文件不是目录）同归 missing", async () => {
    mocks.listDir.mockRejectedValue("Error [GAEADIR_NOT_DIR]: 不是目录: C:/ws/x");
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "text", body: "", dataUrl: "", error: "",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("产物目录不存在");
    expect(mocks.preview).not.toHaveBeenCalled();
  });

  it("诚实降级：GAEADIR_READ_FAILED（权限）无四态码 → 走 Preview 探测兜底（语义不变落 empty）", async () => {
    mocks.listDir.mockRejectedValue("Error [GAEADIR_READ_FAILED]: 读取目录失败: x（拒绝访问）");
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "error",
      body: "", dataUrl: "", error: "目录无法预览",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("产物目录为空");
    expect(mocks.preview).toHaveBeenCalledWith(ROOT);
  });

  it("诚实降级：旧后端兜底——ListDir 拒绝但错误串无码 → Preview 文案匹配照常定性", async () => {
    mocks.listDir.mockRejectedValue("listdir boom");
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "error",
      body: "", dataUrl: "", error: "文件不存在",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("产物目录不存在");
    expect(mocks.preview).toHaveBeenCalledWith(ROOT);
  });

  it("诚实降级：目录在但列为空（探测「目录无法预览」）", async () => {
    mocks.listDir.mockResolvedValue([]);
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "error",
      body: "", dataUrl: "", error: "目录无法预览",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("产物目录为空");
  });

  it("诚实降级：目录有条目但无逐页页面图（如仅审计 PDF 的渲染降级）", async () => {
    mocks.listDir.mockResolvedValue([entry("before.pdf", false), entry("after.pdf", false)]);
    mocks.preview.mockResolvedValue({
      path: ROOT, name: ROOT, ext: "", size: 0, kind: "text",
      body: "", dataUrl: "", error: "",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("没有可识别的逐页页面图");
  });

  it("诚实降级：绝对路径无法相对化（不在工作区）→ 指引外部打开", async () => {
    renderT(<VerifyArtifactsThumbs artifacts="C:/elsewhere/verify/abc" />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    const fault = await screen.findByTestId("verify-thumbs-fault");
    expect(fault.textContent).toContain("无法定位产物目录");
    expect(mocks.listDir).not.toHaveBeenCalled();
  });

  it("单页 Preview 失败：该页失败占位，其余页正常（不拖垮整批）", async () => {
    mocks.listDir.mockImplementation(async (rel: string) => {
      if (rel === ROOT) return [entry("before", true)];
      if (rel === `${ROOT}/before`) return [entry("before-1.png", false), entry("before-2.png", false)];
      return [];
    });
    mocks.preview.mockImplementation(async (rel: string) => {
      if (rel === `${ROOT}/before/before-1.png`) {
        return {
          path: rel, name: "before-1.png", ext: ".png", size: 0, kind: "error",
          body: "", dataUrl: "", error: "读取失败",
        };
      }
      return img(rel);
    });

    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    await screen.findByAltText("第 2 页（改前）");
    await waitFor(() => expect(screen.getAllByTestId("verify-thumb-fail")).toHaveLength(1));
    // 另一页不受拖累：正常渲染 dataUrl
    const okImg = screen.getByAltText("第 2 页（改前）") as HTMLImageElement;
    expect(okImg.getAttribute("src")).toMatch(/^data:image\/png;base64,/);
  });

  it("非图片负载（kind≠image）按失败占位处理，不伪造页面图", async () => {
    mocks.listDir.mockImplementation(async (rel: string) => {
      if (rel === ROOT) return [entry("before", true)];
      if (rel === `${ROOT}/before`) return [entry("before-1.png", false)];
      return [];
    });
    mocks.preview.mockResolvedValue({
      path: `${ROOT}/before/before-1.png`, name: "before-1.png", ext: ".png", size: 0,
      kind: "text", body: "not an image", dataUrl: "", error: "",
    });
    renderT(<VerifyArtifactsThumbs artifacts={ROOT} />);
    fireEvent.click(screen.getByTestId("verify-thumbs-toggle"));
    await waitFor(() => expect(screen.getAllByTestId("verify-thumb-fail")).toHaveLength(1));
    expect(screen.queryByAltText("第 1 页")).toBeNull();
  });
});
