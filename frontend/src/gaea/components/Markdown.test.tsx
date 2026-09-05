import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
const openExternalMock = vi.hoisted(() => vi.fn());
vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, { get: () => () => Promise.resolve({}) }),
  openExternal: openExternalMock,
  onEvent: () => () => {},
}));
import { Markdown } from "./Markdown";
import { usePreviewStore } from "../lib/store";

describe("Markdown 本地文件预览", () => {
  it("本地文件链接渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="报告见 [汇总报告](reports/汇总.md)。" />);
    const btn = screen.getByRole("button", { name: /汇总报告/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("reports/汇总.md");
  });

  it("外部 URL 仍渲染为普通链接", () => {
    render(<Markdown text="参考 [文档](https://example.com/doc)" />);
    expect(screen.queryByRole("button")).toBeNull();
    expect(screen.getByRole("link", { name: /文档/ })).toBeTruthy();
  });

  it("纯文本里的绝对文件路径渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="输出文件：C:\\AI\\bangong\\福达利_成本测算准备资料_v8.3.docx" />);
    const btn = screen.getByRole("button", { name: /福达利_成本测算准备资料_v8\.3\.docx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("C:\\AI\\bangong\\福达利_成本测算准备资料_v8.3.docx");
  });

  it("纯文本里的相对文件路径渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="已生成方案文档：exports/方案.docx，同时更新 .gaea/exports/成本测算.xlsx。" />);
    const btn = screen.getByRole("button", { name: /\.gaea\/exports\/成本测算\.xlsx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe(".gaea/exports/成本测算.xlsx");
  });

  it("关键词引导的裸文件名渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="输出文件：成本测算.xlsx，已生成：报告.docx" />);
    const btn = screen.getByRole("button", { name: /成本测算\.xlsx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("成本测算.xlsx");
  });

  it("URL 与域名式文本不误判为本地文件", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="参考 https://example.com/reports/a.pdf 与 docs.example.com/a.pdf。" />);
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("文件路径带中文句号/括号等标点时只链接路径本身", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="保存到：报告.docx。（详见附件）" />);
    const btn = screen.getByRole("button", { name: /报告\.docx/ });
    // P0-2 chip：文件名 + 扩展名 badge（不含句号后的中文尾巴）
    expect(btn.textContent).toContain("报告.docx");
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("报告.docx");
  });

  it("代码块/内联代码里的路径不转成链接", () => {
    render(<Markdown text={"```\nC:\\AI\\bangong\\内部说明.xlsx\n```\n运行 `C:\\AI\\tools\\fix.bat` 即可。"} />);
    expect(screen.queryAllByRole("button").length).toBe(0);
  });
});

describe("Markdown 外链协议分流（1c）", () => {
  beforeEach(() => {
    openExternalMock.mockClear();
  });

  it("loopback 链接点击不交给系统（渲染文档不得探测本机服务）", () => {
    render(<Markdown text="探针 [内网](http://127.0.0.1:8080/api) 谢谢" />);
    const link = screen.getByRole("link", { name: /内网/ });
    expect(link).toBeTruthy();
    fireEvent.click(link);
    expect(openExternalMock).not.toHaveBeenCalled();
  });

  it("https 链接点击交给系统浏览器", () => {
    render(<Markdown text="参考 [文档](https://example.com/doc)" />);
    fireEvent.click(screen.getByRole("link", { name: /文档/ }));
    expect(openExternalMock).toHaveBeenCalledWith("https://example.com/doc");
  });
});

describe("Markdown 内嵌 HTML 白名单", () => {
  beforeEach(() => {
    openExternalMock.mockClear();
  });

  it("白名单标签正常渲染（b/sub/details/table/pre）", () => {
    const { container } = render(
      <Markdown
        text={[
          "<b>加粗</b> 与 H<sub>2</sub>O",
          "",
          "<details><summary>展开详情</summary>正文内容</details>",
          "",
          "<table><tr><td>单元格</td></tr></table>",
          "",
          "<pre>预格式文本</pre>",
        ].join("\n")}
      />,
    );
    expect(container.querySelector("b")?.textContent).toBe("加粗");
    expect(container.querySelector("sub")?.textContent).toBe("2");
    expect(container.querySelector("details > summary")?.textContent).toContain("展开详情");
    expect(container.querySelector("details")?.textContent).toContain("正文内容");
    expect(container.querySelector("td")?.textContent).toBe("单元格");
    expect(container.querySelector("pre")?.textContent).toBe("预格式文本");
  });

  it("消毒不误伤 md 管线自产 class（KaTeX 公式与围栏代码语言）", () => {
    const { container } = render(
      <Markdown text={"$a^2+b^2=c^2$\n\n```python\nprint(1)\n```"} />,
    );
    expect(container.querySelector(".katex")).toBeTruthy();
    expect(screen.getByText("python")).toBeTruthy();
  });

  it("script 节点与脚本体文本都不出现", () => {
    const { container } = render(<Markdown text={"前文\n\n<script>alert(1)</script>\n\n后文"} />);
    expect(container.querySelector("script")).toBeNull();
    expect(container.textContent).not.toContain("alert(1)");
    expect(container.textContent).toContain("前文");
    expect(container.textContent).toContain("后文");
  });

  it("img 事件属性被剥，安全 src 保留", () => {
    const { container } = render(
      <Markdown text={'<img src="https://example.com/logo.png" onerror="alert(1)" onload="alert(2)">'} />,
    );
    const img = container.querySelector("img");
    expect(img).toBeTruthy();
    expect(img!.getAttribute("onerror")).toBeNull();
    expect(img!.getAttribute("onload")).toBeNull();
    expect(img!.getAttribute("src")).toBe("https://example.com/logo.png");
  });

  it("javascript: href 被剥，点击不交给系统", () => {
    const { container } = render(<Markdown text={'<a href="javascript:alert(1)">诱导链接</a>'} />);
    const link = container.querySelector("a");
    expect(link).toBeTruthy();
    const href = link!.getAttribute("href");
    expect(href === null || href === "").toBe(true);
    expect(link!.textContent).toBe("诱导链接");
    fireEvent.click(link!);
    expect(openExternalMock).not.toHaveBeenCalled();
  });

  it("iframe/style/svg 等禁用标签不渲染且内容不泄漏", () => {
    const { container } = render(
      <Markdown text={'<iframe src="https://example.com">后备字</iframe>\n\n<style>body{}</style>\n\n<svg><circle /></svg>'} />,
    );
    expect(container.querySelector("iframe")).toBeNull();
    expect(container.querySelector("style")).toBeNull();
    expect(container.querySelector("svg")).toBeNull();
    expect(container.textContent).not.toContain("后备字");
    expect(container.textContent).not.toContain("body{}");
  });

  it("form/button 标签全剥；input 仅作任务列表复选框载体（type 按值受限）", () => {
    const { container } = render(
      <Markdown text={'<form action="/steal"><button onclick="alert(1)">提交</button></form>\n\n<input type="text" onclick="x()">'} />,
    );
    expect(container.querySelector("form")).toBeNull();
    expect(container.querySelector("button")).toBeNull();
    expect(container.textContent).not.toContain("提交");
    expect(container.innerHTML).not.toContain("onclick");
    // input 放行只为 remark-gfm 任务列表复选框：raw 的 type="text" 被剥值，
    // 不再渲染成文本输入框（无 type 的空 input 无交互面）。
    const rawInput = container.querySelector("input");
    expect(rawInput).not.toBeNull();
    expect(rawInput!.getAttribute("type")).toBeNull();
  });

  it("GFM 任务列表复选框不被消毒误伤（既有能力红线）", () => {
    const { container } = render(<Markdown text={"- [ ] 待办一\n- [x] 已办二\n"} />);
    const boxes = container.querySelectorAll('input[type="checkbox"]');
    expect(boxes.length).toBe(2);
    expect((boxes[0] as HTMLInputElement).checked).toBe(false);
    expect((boxes[1] as HTMLInputElement).checked).toBe(true);
    expect(container.textContent).toContain("待办一");
  });

  it("style/class/onclick 属性被剥，文本保留", () => {
    const { container } = render(
      <Markdown text={'<div style="color:red" class="evil" onclick="alert(1)">文字</div>'} />,
    );
    expect(container.innerHTML).not.toContain("color:red");
    expect(container.innerHTML).not.toContain("evil");
    expect(container.innerHTML).not.toContain("onclick");
    expect(container.textContent).toContain("文字");
  });

  it("data:image/* 的 img src 保留，其余 data: 剥成空串", () => {
    const { container } = render(
      <Markdown
        text={
          '<img src="data:image/png;base64,iVBORw0KGgo=" alt="内嵌图">\n\n' +
          '<img src="data:text/html;base64,PHNjcmlwdD4=" alt="坏图">'
        }
      />,
    );
    const imgs = container.querySelectorAll("img");
    expect(imgs.length).toBe(2);
    expect(imgs[0]!.getAttribute("src")).toBe("data:image/png;base64,iVBORw0KGgo=");
    expect(imgs[1]!.getAttribute("src")).toBe("");
  });

  it("raw HTML 外链点击走既有 1c 分流（https 交系统浏览器）", () => {
    render(<Markdown text={'<a href="https://example.com/doc">HTML外链</a>'} />);
    fireEvent.click(screen.getByRole("link", { name: /HTML外链/ }));
    expect(openExternalMock).toHaveBeenCalledWith("https://example.com/doc");
  });

  it("raw HTML loopback 链接点击不交给系统", () => {
    render(<Markdown text={'<a href="http://127.0.0.1:8080/api">内网HTML</a>'} />);
    fireEvent.click(screen.getByRole("link", { name: /内网HTML/ }));
    expect(openExternalMock).not.toHaveBeenCalled();
  });
});
