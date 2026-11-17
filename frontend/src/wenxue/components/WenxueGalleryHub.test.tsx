// WenxueGalleryHub — 文学「插图」标签页测试（2026-11-18）：
// 清单加载/来源筛选/空态/大图灯箱开合。bridge 直接 mock（Hub 测试同款）。
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { WenxueGalleryHub } from "./WenxueGalleryHub";

const mocks = vi.hoisted(() => ({
  GalleryList: vi.fn(),
  Preview: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({ app: mocks }));
vi.mock("../../events", () => ({ subscribe: vi.fn(() => () => {}) }));

const ITEMS = [
  { rel: "雾港来信/.gaea/uploads/cover.png", name: "cover.png", source: "uploads" as const, size: 10240, modTime: 100 },
  { rel: "雾港来信/角色/立绘.jpg", name: "立绘.jpg", source: "workspace" as const, size: 2048, modTime: 200 },
];

beforeEach(() => {
  vi.clearAllMocks();
  mocks.GalleryList.mockResolvedValue(ITEMS);
  mocks.Preview.mockImplementation(async (rel: string) => ({
    kind: "image",
    dataUrl: `data:image/png;base64,${encodeURIComponent(rel)}`,
  }));
});

describe("WenxueGalleryHub 插图册", () => {
  it("清单加载：网格两卡、来源 chip、大小标注", async () => {
    render(<WenxueGalleryHub />);
    await waitFor(() => expect(screen.getByTestId("gallery-grid")).toBeTruthy());
    expect(screen.getByText("cover.png")).toBeTruthy();
    expect(screen.getByText("立绘.jpg")).toBeTruthy();
    expect(screen.getAllByText("生成").length).toBeGreaterThan(0);
    expect(screen.getByText("10 KB")).toBeTruthy();
  });

  it("来源筛选：生成/其他 过滤网格", async () => {
    render(<WenxueGalleryHub />);
    await waitFor(() => expect(screen.getByTestId("gallery-grid")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "生成" }));
    expect(screen.getByText("cover.png")).toBeTruthy();
    expect(screen.queryByText("立绘.jpg")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "其他" }));
    expect(screen.getByText("立绘.jpg")).toBeTruthy();
    expect(screen.queryByText("cover.png")).toBeNull();
  });

  it("空态：未选书/无图时引导文案在场", async () => {
    mocks.GalleryList.mockResolvedValue([]);
    render(<WenxueGalleryHub />);
    await waitFor(() => expect(screen.getByText("还没有插图")).toBeTruthy());
    expect(screen.getByText(/image_gen/)).toBeTruthy();
  });

  it("灯箱：点卡片开大图遮罩，点遮罩关闭", async () => {
    const { container } = render(<WenxueGalleryHub />);
    await waitFor(() => expect(screen.getByTestId("gallery-grid")).toBeTruthy());
    await waitFor(() => expect(container.querySelector(".wx-gallery-thumb img")).toBeTruthy());
    fireEvent.click(container.querySelector(".wx-gallery-thumb") as HTMLElement);
    expect(container.querySelector(".wx-gallery-lightbox")).toBeTruthy();
    fireEvent.click(container.querySelector(".wx-gallery-lightbox") as HTMLElement);
    expect(container.querySelector(".wx-gallery-lightbox")).toBeNull();
  });
});
