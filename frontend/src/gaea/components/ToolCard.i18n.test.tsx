// v4.57 i18n 收尾冒烟：ToolCard（diffstat title / 输出头部 / 展开收起 / TaskLiveRow 查看分工）
// 三语字典接线。钉住 zh 断言原硬编码文案。
import { describe, expect, it, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { ToolCard } from "./ToolCard";
import { LocaleProvider } from "../lib/i18n";
import { setTaskCardActivityProvider, setTaskCardAmbiguityHandler, setTaskCardAmbiguityResolver, setTaskCardOpenHandler, setTaskCardOpenTarget } from "../lib/taskActivity";
import type { Item } from "../lib/store";

const renderT = (ui: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const toolItem: Item = {
  kind: "tool",
  id: "t1",
  name: "edit_file",
  args: "{\"path\":\"docs/a.md\",\"old_string\":\"a\",\"new_string\":\"b\\nc\"}",
  readOnly: false,
  status: "done",
  output: Array.from({ length: 40 }, (_, i) => `line ${i}`).join("\n") + "\n" + Array.from({ length: 40 }, (_, i) => `tail ${i}`).join("\n"),
};

describe("ToolCard i18n 冒烟", () => {
  afterEach(() => {
    cleanup();
    // TaskLiveRow 契约：未注入 provider 时整行不渲染 → 用例内注入，用例后卸载
    setTaskCardActivityProvider(null);
  });
  it("diffstat 芯片 title 与输出头部「输出 · NL」走字典（zh）", () => {
    renderT(<ToolCard item={toolItem} />);
    expect(screen.getByTitle("行级增减")).toBeTruthy();
    fireEvent.click(screen.getByText("edit_file"));
    expect(screen.getByText(/输出 · \d+L/)).toBeTruthy();
  });

  it("有界输出折叠开关：展开全部 N 行 / 收起输出（zh）", () => {
    renderT(<ToolCard item={toolItem} />);
    fireEvent.click(screen.getByText("edit_file"));
    const expand = screen.getByText(/展开全部 \d+ 行/);
    fireEvent.click(expand);
    expect(screen.getByText("收起输出")).toBeTruthy();
  });

  it("task 卡运行中显示「点击打开子代理会话」活动行（zh）", () => {
    setTaskCardActivityProvider(() => ({ lastText: "正在收集资料", state: "running" }));
    const taskItem: Item = {
      kind: "tool",
      id: "t2",
      name: "task",
      args: "{\"task\":\"调研\"}",
      readOnly: false,
      status: "running",
    };
    renderT(<ToolCard item={taskItem} />);
    expect(screen.getByTestId("task-live")).toBeTruthy();
    expect(screen.getByText("点击打开子代理会话")).toBeTruthy();
  });

  // v4.503 bash 展开态去重：行内命令主体与 $ 命令块不再同时呈现同一命令
  // （折叠行 `$ cmd…` mono 截断；展开后行内只留结果摘要，命令由 $ 块独占）。
  it("bash 展开后行内命令隐藏，$ 命令块独占（zh）", () => {
    const args = JSON.stringify({ command: "python tools/x.py --flag", timeout: 1 });
    renderT(
      <ToolCard
        item={{ kind: "tool", id: "tb", name: "bash", args, readOnly: false, status: "done", output: "ok\n" }}
      />,
    );
    // 折叠态：行内主体 + $ 命令块各出现一次（$ 块在折叠区常驻 DOM）
    expect(screen.getAllByText(/python tools\/x\.py --flag/).length).toBe(2);
    fireEvent.click(screen.getByText("bash"));
    // 展开态：行内主体隐藏，只剩 $ 命令块一处
    expect(screen.getAllByText(/python tools\/x\.py --flag/).length).toBe(1);
  });

  // v4.504 diff 完成刀：write_file 全增 diff 此前渲染空 original 空盒——
  // 现在展开态直接可见写入内容（+ 行）；edit_file 两侧（− 旧文 / + 新文）。
  it("write_file 展开态显示写入内容 + 行，不再空盒（zh）", () => {
    const args = JSON.stringify({ path: "报告/a.md", content: "# 标题\n第一段\n" });
    renderT(
      <ToolCard item={{ kind: "tool", id: "tw", name: "write_file", args, readOnly: false, status: "done" }} />,
    );
    fireEvent.click(screen.getByText("write_file"));
    expect(screen.getByText("+ # 标题")).toBeTruthy();
    expect(screen.getByText("+ 第一段")).toBeTruthy();
  });

  it("edit_file 展开态 −/+ 两侧同行可见（zh）", () => {
    const args = JSON.stringify({ path: "a.md", old_string: "旧句", new_string: "新句" });
    renderT(
      <ToolCard item={{ kind: "tool", id: "te", name: "edit_file", args, readOnly: false, status: "done" }} />,
    );
    fireEvent.click(screen.getByText("edit_file"));
    expect(screen.getByText("- 旧句")).toBeTruthy();
    expect(screen.getByText("+ 新句")).toBeTruthy();
  });

  // v4.504 只读工具 JSON 盒降噪：read_file 展开态不再渲染 {"path":…} 盒，
  // 可展开性由输出块承担（行内主体本就显示路径）。
  it("read_file 展开态无 args JSON 盒，输出块仍在（zh）", () => {
    const args = JSON.stringify({ path: "data/台账.xlsx" });
    renderT(
      <ToolCard item={{ kind: "tool", id: "tr", name: "read_file", args, readOnly: true, status: "done", output: "→ 3 行\n" }} />,
    );
    fireEvent.click(screen.getByText("read_file"));
    expect(screen.queryByText(/"path"/)).toBeNull();
    expect(screen.getByText("→ 3 行")).toBeTruthy();
  });
});

// ── Codex 式输出简化（2026-10）：bash 命令块 / 写类去双份 / 输出 12 行预览 ──
describe("ToolCard Codex 对齐：bash 命令块与输出有界", () => {
  type ToolItem = Extract<Item, { kind: "tool" }>;
  const bashItem = (nLines: number): ToolItem => ({
    kind: "tool",
    id: "bash-1",
    name: "bash",
    args: JSON.stringify({ command: "go test ./...", timeout: 120000 }),
    readOnly: false,
    status: "done",
    output: Array.from({ length: nLines }, (_, i) => `line ${i}`).join("\n"),
  });

  it("bash 展开态渲染 $ 命令块，不再摊 JSON args（timeout 等字段不可见）", () => {
    renderT(<ToolCard item={bashItem(3)} />);
    fireEvent.click(screen.getByText("bash"));
    expect(screen.getByText("$ go test ./...")).toBeTruthy();
    expect(screen.queryByText(/"command"/)).toBeNull();
    expect(screen.queryByText(/timeout/)).toBeNull();
  });

  it("bash 多行命令原样换行（JSON \\n 转义不再上屏）", () => {
    const multi: ToolItem = { ...bashItem(0), args: JSON.stringify({ command: "echo a\necho b" }) };
    renderT(<ToolCard item={multi} />);
    fireEvent.click(screen.getByText("bash"));
    expect(screen.getByText(/\$ echo a\s+echo b/)).toBeTruthy();
  });

  it("输出预览 12 行上限：30 行折叠为头部 + 折叠提示 + 展开全部按钮", () => {
    renderT(<ToolCard item={bashItem(30)} />);
    fireEvent.click(screen.getByText("bash"));
    expect(screen.getByText(/已折叠 18 行/)).toBeTruthy();
    expect(screen.queryByText(/line 29/)).toBeNull();
    fireEvent.click(screen.getByText(/展开全部 18 行/));
    expect(screen.getByText("收起输出")).toBeTruthy();
    expect(screen.getByText(/line 29/)).toBeTruthy();
  });

  it("写文件类展开态只出 diff，args JSON 不再重复渲染", () => {
    renderT(<ToolCard item={toolItem} />);
    fireEvent.click(screen.getByText("edit_file"));
    expect(screen.queryByText(/old_string/)).toBeNull();
    expect(screen.getByText(/输出 · /)).toBeTruthy();
  });
});

// ── v4.63 子代理卡片整卡可点：task 卡点击派发「打开会话」跳转 ──
describe("ToolCard 子代理卡片点击跳转（v4.63）", () => {
  afterEach(() => {
    cleanup();
    setTaskCardOpenHandler(null);
    setTaskCardOpenTarget(null);
    setTaskCardActivityProvider(null);
  });

  const taskCard: Item = {
    kind: "tool",
    id: "task-1",
    name: "task",
    args: "{}",
    readOnly: false,
    status: "done",
    output: "调研完成。\nSubagent reference: sa_abc123",
  };

  it("解析到 ref：点击整卡派发跳转（不再走展开折叠）", () => {
    const opened: string[] = [];
    setTaskCardOpenHandler((ref) => opened.push(ref));
    setTaskCardOpenTarget((ref) => ref);
    renderT(<ToolCard item={taskCard} />);
    fireEvent.click(screen.getByText("task"));
    expect(opened).toEqual(["sa_abc123"]);
  });

  it("无注入（解析不到 ref）：点击不跳转，维持展开折叠现状", () => {
    renderT(<ToolCard item={taskCard} />);
    fireEvent.click(screen.getByText("task"));
    // 展开行为：输出区可见（默认折叠，点击后展开）
    expect(screen.getByText(/输出 · /)).toBeTruthy();
  });

  it("空 ref 回退：唯一 running 命中时由 App 解析器给出目标 ref", () => {
    const opened: string[] = [];
    setTaskCardOpenHandler((ref) => opened.push(ref));
    setTaskCardOpenTarget((ref, args) => (ref === "" && String(args).includes("土壤修复") ? "sa_running9" : ""));
    renderT(
      <ToolCard
        item={{ ...taskCard, output: "", args: "{\"description\":\"土壤修复技术调研\"}", status: "running" }}
      />,
    );
    fireEvent.click(screen.getByText("task"));
    expect(opened).toEqual(["sa_running9"]);
  });
});

// ── v4.68 空 ref 多候选歧义：卡可点 → 点击派发歧义处理器（App 弹选择器）──
describe("ToolCard 多候选歧义点击（v4.68）", () => {
  afterEach(() => {
    cleanup();
    setTaskCardAmbiguityResolver(null);
    setTaskCardAmbiguityHandler(null);
    setTaskCardOpenHandler(null);
    setTaskCardOpenTarget(null);
    setTaskCardActivityProvider(null);
  });

  // 并行派发瞬间：无 ref（args 不带 continue_from、output 不含引用行）、
  // App 侧 ≥2 个 running 文本命中 → 歧义态
  const ambiguousCard: Item = {
    kind: "tool",
    id: "task-amb",
    name: "task",
    args: "{\"description\":\"土壤修复技术调研\"}",
    readOnly: false,
    status: "running",
    output: "调研进行中",
  };

  it("歧义态：整卡可点（cursor + aria「多个运行中任务，点击选择」），点击派发歧义处理器", () => {
    const fired: string[] = [];
    setTaskCardAmbiguityResolver((ref, args) => ref === "" && String(args).includes("土壤修复"));
    setTaskCardAmbiguityHandler((args) => { fired.push(String(args)); });
    renderT(<ToolCard item={ambiguousCard} />);
    const row = screen.getByLabelText("多个运行中任务，点击选择");
    expect(row).toBeTruthy();
    expect(row.className).toContain("cursor-pointer");
    fireEvent.click(row);
    expect(fired).toHaveLength(1);
    expect(fired[0]).toContain("土壤修复技术调研");
  });

  it("非歧义（resolver 判 false，模拟 0/1 候选）：卡不可点、点击不派发，维持展开折叠现状", () => {
    const fired: string[] = [];
    setTaskCardAmbiguityResolver(() => false);
    setTaskCardAmbiguityHandler((args) => { fired.push(String(args)); });
    renderT(<ToolCard item={ambiguousCard} />);
    expect(screen.queryByLabelText("多个运行中任务，点击选择")).toBeNull();
    fireEvent.click(screen.getByText("task"));
    expect(fired).toEqual([]);
    expect(screen.getByText(/输出 · /)).toBeTruthy();
  });

  it("只注册 resolver 未注册 handler：按现状渲染（不可点），绝不弹空选择器", () => {
    setTaskCardAmbiguityResolver(() => true);
    renderT(<ToolCard item={ambiguousCard} />);
    expect(screen.queryByLabelText("多个运行中任务，点击选择")).toBeNull();
    fireEvent.click(screen.getByText("task"));
    // 回退展开折叠现状
    expect(screen.getByText(/输出 · /)).toBeTruthy();
  });

  it("已有唯一目标（openTarget 解析到 ref）时歧义槽位不介入：仍走直跳链路", () => {
    const opened: string[] = [];
    const fired: string[] = [];
    // 带 Subagent reference 行的完成卡：openTarget 解析到唯一 ref
    const refCard: Item = {
      kind: "tool",
      id: "task-ref",
      name: "task",
      args: "{}",
      readOnly: false,
      status: "done",
      output: "调研完成。\nSubagent reference: sa_abc123",
    };
    setTaskCardOpenHandler((ref) => opened.push(ref));
    setTaskCardOpenTarget((ref) => ref);
    setTaskCardAmbiguityResolver(() => true);
    setTaskCardAmbiguityHandler((args) => { fired.push(String(args)); });
    renderT(<ToolCard item={refCard} />);
    fireEvent.click(screen.getByText("task"));
    expect(opened).toEqual(["sa_abc123"]);
    expect(fired).toEqual([]);
  });
});
