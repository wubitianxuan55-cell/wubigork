import { describe, expect, it } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MarkdownContent } from './MarkdownContent'

describe('MarkdownContent GFM 渲染', () => {
  it('渲染 GFM 表格', () => {
    render(
      <MarkdownContent
        source={'| 角色 | 定位 |\n| --- | --- |\n| 林晚 | 主角 |\n| 顾辞 | 反派 |'}
      />,
    )
    const table = document.querySelector('table')
    expect(table).toBeTruthy()
    expect(screen.getByText('角色')).toBeTruthy()
    expect(screen.getByText('林晚')).toBeTruthy()
    expect(screen.getByText('顾辞')).toBeTruthy()
  })

  it('渲染标题与列表等基础语法', () => {
    render(<MarkdownContent source={'# 世界观\n\n- A\n- B'} />)
    expect(screen.getByRole('heading', { level: 1, name: '世界观' })).toBeTruthy()
    expect(screen.getByText('A')).toBeTruthy()
    expect(screen.getByText('B')).toBeTruthy()
  })
})

describe("MarkdownContent 缺省代码高亮（v4.233）", () => {
  it("无覆盖时块级代码走 ChatCodeBlock（暗面板+异步 hljs 令牌）", async () => {
    const goFence = "```go\npackage main\n\nfunc main() {}\n```";
    const { container } = render(<MarkdownContent source={goFence} />);
    expect(container.querySelector(".hl-scope-dark")).toBeTruthy();
    await waitFor(() => expect(container.querySelector(".hljs-keyword")).toBeTruthy());
  });

  it("行内代码不进面板（交还 .md-content code 默认样式）", () => {
    const { container } = render(<MarkdownContent source={"用 `npm run build` 构建"} />);
    expect(container.querySelector(".hl-scope-dark")).toBeNull();
    expect(container.querySelector("code")?.textContent).toBe("npm run build");
  });

  it("传了 components（GenUI 缝）：尊重调用方，零变化", () => {
    const goFence = "```go\npackage main\n```";
    const { container } = render(
      <MarkdownContent source={goFence} components={{ code: (p) => <code>{String(p.children)}</code> }} />,
    );
    expect(container.querySelector(".hl-scope-dark")).toBeNull();
    expect(container.querySelector("code")?.textContent).toContain("package main");
  });
});
