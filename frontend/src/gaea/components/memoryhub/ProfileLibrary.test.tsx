import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { ProfileLibrary } from "./ProfileLibrary";

// 回归：后端空结果（nil 切片）序列化成 null、画像 tags 为 null 时，
// 页面不能崩溃（此前 conflicts.length / f.tags.length 直接读 null 抛 TypeError）。
vi.mock("../../lib/bridge", () => ({
  app: {
    ProfileList: vi.fn(async () => [
      {
        name: "prefers-tabs",
        title: "用户偏好",
        description: "喜欢先给大纲再展开",
        type: "user",
        kind: "semantic",
        tags: null,
        body: "细节",
      },
    ]),
    ProfileConflicts: vi.fn(async () => null),
    ProfileSave: vi.fn(async () => {}),
    ProfileDelete: vi.fn(async () => {}),
    ProfileResolveConflict: vi.fn(async () => {}),
  },
}));

describe("ProfileLibrary 用户画像", () => {
  it("后端返回 null 时正常打开并展示画像事实", async () => {
    render(<ProfileLibrary />);
    // 画像事实正常渲染（tags 为 null 不崩溃）
    expect(await screen.findByText("用户偏好")).toBeTruthy();
    expect(screen.getByText("喜欢先给大纲再展开")).toBeTruthy();
    // conflicts 为 null 时不显示冲突横幅
    await waitFor(() => expect(screen.queryByText(/画像与遗留 facts 冲突/)).toBeNull());
  });
});
