import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { FileMenu } from "./FileMenu";

// jsdom 未实现 scrollIntoView（MenuContainer 的 useMenuScroll 会调用）
Element.prototype.scrollIntoView = () => {};

describe("FileMenu @ 文件菜单", () => {
  it("目录/文件条目渲染，文件显示扩展名徽标", () => {
    render(
      <FileMenu
        items={[
          { path: "docs/", name: "docs", isDir: true },
          { path: "docs/方案.docx", name: "方案.docx", isDir: false },
          { path: "docs/数据.xlsx", name: "数据.xlsx", isDir: false },
        ]}
        activeIndex={0}
        onPick={() => {}}
        onHover={() => {}}
      />,
    );
    expect(screen.getByText("docs/")).toBeTruthy();
    expect(screen.getByText("docx")).toBeTruthy();
    expect(screen.getByText("xlsx")).toBeTruthy();
  });
});
