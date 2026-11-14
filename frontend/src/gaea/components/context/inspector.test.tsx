// inspector.tsx 定向测试（jsdom）：分类折叠组 / 行内搜索 / 归档组 / 分页上限 /
// 文件活动聚合树（chips、排序切换、路径过滤、点击预览）/ 空态。
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { ContextSurfaceNode, FileActivity } from "../../lib/types";
import { ContextBrowserTree, FileActivityTree, NodeDetailPanel } from "./inspector";

// 组件链会经由 ContextView（CAT_COLORS）触达 bridge；jsdom 无 Wails 全局，按
// ContextView.test.tsx 惯例 mock 掉 seam（store 的 usePreviewStore 保持真实实现）。
vi.mock("../../lib/bridge", () => ({
  app: new Proxy({}, { get: () => () => Promise.reject(new Error("unused in inspector tests")) }),
  openExternal: () => {},
  onEvent: () => () => {},
  onReady: () => () => {},
}));

// 走 useT：钉住 zh 让中文文案断言成立
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const LONG_TEXT = "助手回复全文：" + "内".repeat(64); // >56 字符，触发展开/收起

const NODES: ContextSurfaceNode[] = [
  { seq: 1, cat: "system", tokens: 2100, text: "系统提示词正文标记甲" },
  { seq: 2, cat: "assistant", tokens: 1000, text: LONG_TEXT },
  { seq: 3, cat: "tool", tokens: 8000, text: "工具输出正文标记乙" },
];

const ARCHIVE: ContextSurfaceNode[] = [
  { seq: 30, cat: "user", tokens: 12000, text: "归档正文标记丙", gone: 30 },
];

const FILES: FileActivity[] = [
  { seq: 5, ts: 1750000005, turn: 1, step: 1, tool: "read_file", action: "read", path: "src/a.ts" },
  { seq: 9, ts: 1750000100, turn: 1, step: 2, tool: "read_file", action: "read", path: "src/a.ts" },
  { seq: 12, ts: 1750000200, turn: 2, step: 1, tool: "write_file", action: "write", path: "docs/b.md" },
  { seq: 15, ts: 1750000060, turn: 2, step: 2, tool: "grep_files", action: "read", path: "src/c.ts" },
  { seq: 18, ts: 1750000070, turn: 2, step: 3, tool: "list_dir", action: "dir", path: "assets" },
];

const chipBtn = (label: string) => {
  const el = screen.getByText(label).closest("button");
  if (!el) throw new Error(`chip button not found: ${label}`);
  return el;
};

beforeEach(() => {
  document.body.innerHTML = "";
});

describe("ContextBrowserTree 上下文浏览器（分类折叠组）", () => {
  it("渲染六分类行：色点行含分类名、N 项徽标、≈tokens 与占比；默认全部收起", () => {
    const { container } = renderT(<ContextBrowserTree nodes={NODES} archive={ARCHIVE} />);
    // 组行：系统提示词 1 项 ≈2.1k (19%)（总 11.1k）
    const sysRow = screen.getByText("系统提示词").closest("button");
    expect(sysRow?.textContent).toContain("1 项");
    expect(sysRow?.textContent).toContain("≈2.1k");
    expect(sysRow?.textContent).toContain("(19%)");
    expect(screen.getByText("助手消息").closest("button")?.textContent).toContain("(9%)");
    expect(screen.getByText("工具结果").closest("button")?.textContent).toContain("1 项");
    // 标题行右侧提示：活跃 3 · ≈11.1k
    expect(screen.getByText(/活跃 3/).textContent).toContain("≈11.1k");
    // 默认收起：节点正文不可见
    expect(screen.queryByText(/标记甲/)).toBeNull();
    expect(screen.queryByText(/标记乙/)).toBeNull();
    // 归档组默认收起，但行可见
    expect(screen.getByText(/归档 1/)).toBeTruthy();
    expect(screen.queryByText(/标记丙/)).toBeNull();
    expect(container).toBeTruthy();
  });

  it("点分类行展开节点列表；长文本节点可展开全文/收起", () => {
    renderT(<ContextBrowserTree nodes={NODES} archive={[]} />);
    fireEvent.click(screen.getByText("系统提示词"));
    expect(screen.getByText(/标记甲/)).toBeTruthy();
    fireEvent.click(screen.getByText("助手消息"));
    expect(screen.getByText(/助手回复全文/)).toBeTruthy();
    // 长文本 >56 字符 → 展开/收起（交互同现有 NodeRow）
    fireEvent.click(screen.getByText("展开"));
    expect(screen.getByText("收起")).toBeTruthy();
    expect(screen.getByText(LONG_TEXT)).toBeTruthy();
    fireEvent.click(screen.getByText("收起"));
    expect(screen.queryByText("收起")).toBeNull();
    expect(screen.queryByText(LONG_TEXT)).toBeNull();
  });

  it("行内搜索框过滤节点文本：项数徽标随之变化", () => {
    renderT(<ContextBrowserTree nodes={NODES} archive={ARCHIVE} />);
    const input = screen.getByPlaceholderText("过滤节点文本…");
    fireEvent.change(input, { target: { value: "标记乙" } });
    expect(screen.getByText("工具结果").closest("button")?.textContent).toContain("1 项");
    expect(screen.getByText("系统提示词").closest("button")?.textContent).toContain("0 项");
    // 归档组计数同样被搜索过滤（标记丙不匹配 → 0）
    expect(screen.getByText(/归档 0/)).toBeTruthy();
    fireEvent.click(screen.getByText("工具结果"));
    expect(screen.getByText(/标记乙/)).toBeTruthy();
    expect(screen.queryByText(/标记甲/)).toBeNull();
    // 清空恢复
    fireEvent.change(input, { target: { value: "" } });
    expect(screen.getByText("系统提示词").closest("button")?.textContent).toContain("1 项");
  });

  it("归档折叠组：展开显示被压缩节点并带「已压缩」徽标", () => {
    renderT(<ContextBrowserTree nodes={NODES} archive={ARCHIVE} />);
    fireEvent.click(screen.getByText(/归档 1/));
    expect(screen.getByText(/标记丙/)).toBeTruthy();
    expect(screen.getAllByText("已压缩").length).toBeGreaterThan(0);
  });

  it("助手消息 >200 项：展开先显前 100 行 + 「显示全部 N 项」", () => {
    const MANY: ContextSurfaceNode[] = Array.from({ length: 250 }, (_, i) => ({
      seq: 1000 + i,
      cat: "assistant" as const,
      tokens: 1,
      text: `助手节点正文 #${i}`,
    }));
    renderT(<ContextBrowserTree nodes={MANY} archive={[]} />);
    expect(screen.getByText("助手消息").closest("button")?.textContent).toContain("250 项");
    fireEvent.click(screen.getByText("助手消息"));
    expect(screen.getByText("助手节点正文 #0")).toBeTruthy();
    expect(screen.getByText("助手节点正文 #99")).toBeTruthy();
    expect(screen.queryByText("助手节点正文 #100")).toBeNull();
    fireEvent.click(screen.getByText("显示全部 250 项"));
    expect(screen.getByText("助手节点正文 #100")).toBeTruthy();
    expect(screen.getByText("助手节点正文 #249")).toBeTruthy();
  });

  it("空态：无节点无归档时显示占位文案", () => {
    renderT(<ContextBrowserTree nodes={[]} archive={[]} />);
    expect(screen.getByText("暂无上下文节点")).toBeTruthy();
  });
});

describe("NodeDetailPanel header 全文", () => {
  it("工具分类展开 request_header 的真实 schema 与估算 token；不混入 system 正文", () => {
    renderT(
      <NodeDetailPanel
        headerFocus="tools"
        d={{
          seq: 2,
          kind: "header",
          text: "系统提示词正文",
          tools: [
            { name: "read_file", schema: '{"type":"object"}', tokens: 12, source: "builtin-workspace" },
            { name: "bash" },
          ],
          lines: 1,
        }}
      />,
    );
    expect(screen.getByText("工具 schema · 2")).toBeTruthy();
    expect(screen.getAllByText("read_file").length).toBeGreaterThan(0);
    expect(screen.getByText('{"type":"object"}')).toBeTruthy();
    expect(screen.getByText("builtin-workspace")).toBeTruthy();
    expect(screen.getByText("该请求头只记录了工具名，没有 schema。")).toBeTruthy();
    expect(screen.queryByText("系统提示词正文")).toBeNull();
  });

  it("用户消息渲染非图片附件卡：类型/大小/缺失态与预览入口", () => {
    renderT(
      <NodeDetailPanel
        d={{
          seq: 5,
          kind: "user_message",
          text: "看附件",
          lines: 1,
          attachments: [
            {
              ref: ".gaea/uploads/report-a.xlsx",
              path: "C:/demo/.gaea/uploads/report-a.xlsx",
              name: "report-a.xlsx",
              mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
              size: 2048,
              exists: true,
            },
            {
              ref: ".gaea/uploads/spec.pdf",
              path: "C:/demo/.gaea/uploads/spec.pdf",
              name: "spec.pdf",
              exists: false,
            },
          ],
        }}
      />,
    );
    expect(screen.getByTestId("ctx-node-attachments").textContent).toContain("附件 · 2");
    expect(screen.getByText("report-a.xlsx")).toBeTruthy();
    expect(screen.getByText("2.0 KB")).toBeTruthy();
    expect(screen.getAllByText("文件不存在").length).toBeGreaterThan(0);
    const cards = screen.getAllByTestId("ctx-file-attachment-card");
    expect((cards[0] as HTMLButtonElement).disabled).toBe(false);
    expect((cards[1] as HTMLButtonElement).disabled).toBe(true);
  });

  it("图片与非图片附件按正文出现顺序混排为同一组卡片", () => {
    const { container } = renderT(
      <NodeDetailPanel
        d={{
          seq: 6,
          kind: "user_message",
          text: "先看 .gaea/uploads/spec.pdf，再看 C:/pics/a.png",
          lines: 1,
          images: [{ ref: "C:/pics/a.png", path: "C:/pics/a.png", exists: false }],
          attachments: [{
            ref: ".gaea/uploads/spec.pdf",
            path: "C:/demo/.gaea/uploads/spec.pdf",
            name: "spec.pdf",
            exists: false,
          }],
        }}
      />,
    );
    const cards = Array.from(container.querySelectorAll("[data-attachment-kind]"));
    expect(cards.map((card) => card.getAttribute("data-attachment-kind"))).toEqual(["file", "image"]);
  });
});

describe("FileActivityTree 文件活动（按文件聚合树）", () => {
  it("渲染聚合树行：路径 + 读写次数徽标 + 最新时间；纯目录行不可点", () => {
    const { container } = renderT(<FileActivityTree files={FILES} />);
    const row = (p: string) => container.querySelector(`button[data-file="${p}"]`);
    expect(row("src/a.ts")?.textContent).toContain("src/a.ts");
    expect(row("src/a.ts")?.textContent).toContain("读 2");
    expect(row("docs/b.md")?.textContent).toContain("写 1");
    expect(row("assets")?.textContent).toContain("目录 1");
    expect(row("assets")?.hasAttribute("disabled")).toBe(true);
    // 行内时间存在（时区相关，只断言时间形态）
    expect(row("src/a.ts")?.textContent).toMatch(/:\d{2}/);
  });

  it("过滤 chips 计数与切换：全部/读取/写入/搜索/图片", () => {
    const { container } = renderT(<FileActivityTree files={FILES} />);
    expect(chipBtn("全部").textContent).toContain("5");
    expect(chipBtn("读取").textContent).toContain("3");
    expect(chipBtn("写入").textContent).toContain("1");
    expect(chipBtn("搜索").textContent).toContain("1"); // grep_files；list_dir 是 dir 不计入
    expect(chipBtn("图片").textContent).toContain("0");
    fireEvent.click(chipBtn("写入"));
    expect(screen.getByTitle("docs/b.md")).toBeTruthy();
    expect(container.querySelector('button[data-file="src/a.ts"]')).toBeNull();
    fireEvent.click(chipBtn("全部"));
    expect(container.querySelector('button[data-file="src/a.ts"]')).toBeTruthy();
  });

  it("排序三胶囊：按次数（默认）/按最新/按路径，并列确定性决胜", () => {
    const { container } = renderT(<FileActivityTree files={FILES} />);
    const order = () =>
      Array.from(container.querySelectorAll("button[data-file]")).map((b) => b.getAttribute("data-file"));
    // 按次数（默认）：a.ts(2) → b.md(200 最新) → assets(70) → c.ts(60)
    expect(order()).toEqual(["src/a.ts", "docs/b.md", "assets", "src/c.ts"]);
    fireEvent.click(screen.getByText("按最新"));
    expect(order()).toEqual(["docs/b.md", "src/a.ts", "assets", "src/c.ts"]);
    fireEvent.click(screen.getByText("按路径"));
    expect(order()).toEqual(["assets", "docs/b.md", "src/a.ts", "src/c.ts"]);
  });

  it("排序扩展：按行数（缺省时按最新决胜）与按名称", () => {
    localStorage.removeItem("gaea.context.prefs");
    const { container } = renderT(<FileActivityTree files={FILES} />);
    const order = () =>
      Array.from(container.querySelectorAll("button[data-file]")).map((b) => b.getAttribute("data-file"));
    fireEvent.click(screen.getByText("按行数"));
    expect(order()).toEqual(["docs/b.md", "src/a.ts", "assets", "src/c.ts"]);
    fireEvent.click(screen.getByText("按名称"));
    expect(order()).toEqual(["src/a.ts", "assets", "docs/b.md", "src/c.ts"]);
  });

  it("路径过滤输入框 + 汇总行「N 个文件」；无匹配时空态", () => {
    const { container } = renderT(<FileActivityTree files={FILES} />);
    expect(screen.getByText(/个文件/).textContent).toContain("4 个文件");
    const input = screen.getByPlaceholderText("按路径过滤…");
    fireEvent.change(input, { target: { value: "docs" } });
    expect(screen.getByTitle("docs/b.md")).toBeTruthy();
    expect(container.querySelector('button[data-file="src/a.ts"]')).toBeNull();
    expect(screen.getByText(/个文件/).textContent).toContain("1 个文件");
    fireEvent.change(input, { target: { value: "zzz-no-hit" } });
    expect(screen.getByText("无匹配文件")).toBeTruthy();
  });

  it("行点击打开预览：注入 onOpenFile 生效；纯目录行不触发", () => {
    const onOpen = vi.fn();
    const { container } = renderT(<FileActivityTree files={FILES} onOpenFile={onOpen} />);
    fireEvent.click(container.querySelector('button[data-file="src/a.ts"]') as HTMLElement);
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(onOpen).toHaveBeenCalledWith("src/a.ts");
    fireEvent.click(container.querySelector('button[data-file="assets"]') as HTMLElement);
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("默认预览链：不传 onOpenFile 时内部走 usePreviewStore", async () => {
    const { usePreviewStore } = await import("../../lib/store");
    const { container } = renderT(<FileActivityTree files={FILES} />);
    fireEvent.click(container.querySelector('button[data-file="src/a.ts"]') as HTMLElement);
    expect(usePreviewStore.getState().previewFile).toBe("src/a.ts");
  });

  it("节点详情走注入的自定义源（FE1-04）：下传 fetchNodeDetail 时操作行详情按注入源回读", async () => {
    // 排序初值读 localStorage 偏好（同文件「排序三胶囊」用例会写入「按路径」，
    // 令首行变 assets）——本用例钉默认「按次数」序，先清偏好。
    localStorage.removeItem("gaea.context.prefs");
    const fetchDetail = vi.fn(async (seq: number) => ({
      seq,
      kind: "tool_result" as const,
      tool: "read_file",
      output: `自定义源详情 ${seq}`,
    }));
    const { container } = renderT(
      <FileActivityTree files={FILES} sessionPath="/sess/x" fetchNodeDetail={fetchDetail} />,
    );
    // 展开首条文件的操作日志（默认按次数排序，首行=src/a.ts，聚合 2 次 read）
    fireEvent.click(screen.getAllByTestId("file-ops-toggle")[0]);
    // 点开第一条操作行的「详情」→ 必须调注入的自定义源（缺省源会让原罪等
    // 自定义源页面串数据源——同一页面两套回读通道）
    fireEvent.click(screen.getAllByTestId("file-op-detail-btn")[0]);
    await vi.waitFor(() => expect(fetchDetail).toHaveBeenCalledTimes(1));
    expect(fetchDetail).toHaveBeenCalledWith(5);
    // 详情内容来自注入源
    await vi.waitFor(() => expect(screen.getByText("自定义源详情 5")).toBeTruthy());
    expect(container.textContent).not.toContain("回读失败");
  });

  it("操作日志可定位到上下文浏览器对应节点", () => {
    localStorage.removeItem("gaea.context.prefs");
    const onLocateOp = vi.fn();
    renderT(<FileActivityTree files={FILES} onLocateOp={onLocateOp} />);
    fireEvent.click(screen.getAllByTestId("file-ops-toggle")[0]);
    fireEvent.click(screen.getAllByTestId("file-op-locate-btn")[0]);
    expect(onLocateOp).toHaveBeenCalledWith(5);
  });

  it("钉住历史步骤后文件活动只显示截至该步的操作", () => {
    const { container } = renderT(
      <FileActivityTree
        files={FILES}
        step={{
          seq: 8,
          ts: 1750000008,
          turn: 1,
          step: 1,
          category: { system: 0, tools: 0, user: 0, skill: 0, inject: 0, assistant: 0, tool: 0 },
        }}
      />,
    );
    expect(screen.getByTestId("ctx-files-step-scope").textContent).toContain("T1·S1");
    expect(container.querySelector('button[data-file="src/a.ts"]')).toBeTruthy();
    expect(container.querySelector('button[data-file="docs/b.md"]')).toBeNull();
    expect(container.querySelector('button[data-file="src/c.ts"]')).toBeNull();
  });

  it("空态：无文件活动时复用现有文案键", () => {
    renderT(<FileActivityTree files={[]} />);
    expect(screen.getByText(/暂无文件活动/)).toBeTruthy();
  });
});

