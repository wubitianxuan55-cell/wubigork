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

  it("工具行使用可展开按钮语义，并关联正文区域", () => {
    renderT(<ToolCard item={toolItem} />);
    const button = screen.getByText("edit_file").closest("button");
    expect(button).toBeTruthy();
    expect(button?.getAttribute("aria-expanded")).toBe("false");
    const controls = button?.getAttribute("aria-controls");
    expect(controls).toBeTruthy();
    expect(document.getElementById(controls!)).toBeTruthy();
    fireEvent.click(button!);
    expect(button?.getAttribute("aria-expanded")).toBe("true");
  });

  it("有界输出折叠开关：展开全部 N 行 / 收起输出（zh）", () => {
    renderT(<ToolCard item={toolItem} />);
    fireEvent.click(screen.getByText("edit_file"));
    const expand = screen.getByText(/展开全部 \d+ 行/);
    fireEvent.click(expand);
    expect(screen.getByText("收起输出")).toBeTruthy();
  });

  it("短输出没有隐藏内容时不给空展开入口", () => {
    renderT(
      <ToolCard
        item={{
          kind: "tool", id: "short", name: "read_file", args: "{\"path\":\"a.md\"}",
          readOnly: true, status: "done", output: "ok\n",
        }}
      />,
    );
    const button = screen.getByText("read_file").closest("button");
    expect(button).toBeTruthy();
    expect((button as HTMLButtonElement).disabled).toBe(true);
    expect(button?.getAttribute("aria-expanded")).toBeNull();
  });

  it("长输出按实际隐藏量开放，并在头部复用同一 totalLines", () => {
    const output = Array.from({ length: 30 }, (_, i) => `line ${i}`).join("\n");
    renderT(
      <ToolCard
        item={{
          kind: "tool", id: "long", name: "read_file", args: "{\"path\":\"a.md\"}",
          readOnly: true, status: "done", output,
        }}
      />,
    );
    const button = screen.getByText("read_file").closest("button");
    expect(button?.getAttribute("aria-expanded")).toBe("false");
    fireEvent.click(button!);
    expect(screen.getByText("输出 · 30L")).toBeTruthy();
    expect(screen.getByText(/展开全部 18 行/)).toBeTruthy();
  });

  it("task 卡运行中显示当前活动与等待/查看入口（zh）", () => {
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
    expect(screen.getByText("正在收集资料")).toBeTruthy();
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
    // 灵犀皮肤批（batch3 UI）：± 标记独立成 label span，行文本按 data-diff 行断言
    const addRow = screen.getByText("# 标题").closest("[data-diff]");
    expect(addRow?.textContent).toBe("+ # 标题");
    expect(addRow?.getAttribute("data-diff")).toBe("add");
    expect(screen.getByText("第一段").closest("[data-diff]")?.getAttribute("data-diff")).toBe("add");
  });

  it("edit_file 展开态 −/+ 两侧同行可见（zh）", () => {
    const args = JSON.stringify({ path: "a.md", old_string: "旧句", new_string: "新句" });
    renderT(
      <ToolCard item={{ kind: "tool", id: "te", name: "edit_file", args, readOnly: false, status: "done" }} />,
    );
    fireEvent.click(screen.getByText("edit_file"));
    const delRow = screen.getByText("旧句").closest("[data-diff]");
    const addRow = screen.getByText("新句").closest("[data-diff]");
    expect(delRow?.getAttribute("data-diff")).toBe("del");
    expect(addRow?.getAttribute("data-diff")).toBe("add");
    expect(delRow?.textContent).toBe("- 旧句");
    expect(addRow?.textContent).toBe("+ 新句");
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
    expect(screen.getByText(/line 29/)).toBeTruthy();
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
    fireEvent.click(screen.getByTestId("delegate-head"));
    expect(opened).toEqual(["sa_abc123"]);
  });

  it("无注入（解析不到 ref）：点击不跳转，只保留摘要", () => {
    renderT(<ToolCard item={taskCard} />);
    fireEvent.click(screen.getByTestId("delegate-head"));
    // 简化卡不在消息流内渲染输出；无会话目标时只保留摘要。
    expect(screen.queryByText(/输出 · /)).toBeNull();
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
    fireEvent.click(screen.getByTestId("delegate-head"));
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

  it("非歧义（resolver 判 false，模拟 0/1 候选）：卡不可点、点击不派发，只保留摘要", () => {
    const fired: string[] = [];
    setTaskCardAmbiguityResolver(() => false);
    setTaskCardAmbiguityHandler((args) => { fired.push(String(args)); });
    renderT(<ToolCard item={ambiguousCard} />);
    expect(screen.queryByLabelText("多个运行中任务，点击选择")).toBeNull();
    fireEvent.click(screen.getByTestId("delegate-head"));
    expect(fired).toEqual([]);
    expect(screen.queryByText(/输出 · /)).toBeNull();
  });

  it("只注册 resolver 未注册 handler：按摘要渲染（不可点），绝不弹空选择器", () => {
    setTaskCardAmbiguityResolver(() => true);
    renderT(<ToolCard item={ambiguousCard} />);
    expect(screen.queryByLabelText("多个运行中任务，点击选择")).toBeNull();
    fireEvent.click(screen.getByTestId("delegate-head"));
    expect(screen.queryByText(/输出 · /)).toBeNull();
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
    fireEvent.click(screen.getByTestId("delegate-head"));
    expect(opened).toEqual(["sa_abc123"]);
    expect(fired).toEqual([]);
  });
});

// ── 2026-11-18 委派卡重设计（DelegationCard）：一等卡形态钉死 ──
describe("ToolCard 委派卡形态（DelegationCard 重设计）", () => {
  afterEach(() => {
    cleanup();
    setTaskCardActivityProvider(null);
    setTaskCardOpenHandler(null);
    setTaskCardOpenTarget(null);
  });

  it("完成卡：子代理徽标 + 已完成状态词 + 结果摘要（不再渲染裸 task 名行）", () => {
    const card: Item = {
      kind: "tool",
      id: "d1",
      name: "task",
      args: '{"description":"土壤修复调研"}',
      readOnly: false,
      status: "done",
      output: "调研完成。\nSubagent reference: sa_x1",
    };
    renderT(<ToolCard item={card} />);
    expect(screen.getByTestId("delegate-head")).toBeTruthy();
    expect(screen.getByText("子代理")).toBeTruthy();
    expect(screen.getByText("已完成")).toBeTruthy();
    expect(screen.getByTestId("delegate-result").textContent).toContain("调研完成。");
    expect(screen.queryByText("task")).toBeNull();
  });

  it("运行卡：运行中状态词 + 当前活动；引擎保留在标题提示中", () => {
    setTaskCardActivityProvider(() => ({ lastText: "正在收集资料", state: "running" }));
    const card: Item = {
      kind: "tool",
      id: "d2",
      name: "task",
      args: '{"description":"调研","engine":"qwen3.5"}',
      readOnly: false,
      status: "running",
    };
    renderT(<ToolCard item={card} />);
    expect(screen.getByText("运行中")).toBeTruthy();
    expect(screen.getByText("调研").getAttribute("title")).toContain("qwen3.5");
    expect(screen.getByTestId("task-live").textContent).toContain("正在收集资料");
  });

  it("运行卡无活动 provider 时显示等待回传，而不是空白活动条", () => {
    const card: Item = {
      kind: "tool",
      id: "d2b",
      name: "task",
      args: '{"description":"法规检索","run_in_background":true}',
      readOnly: false,
      status: "running",
    };
    renderT(<ToolCard item={card} />);
    expect(screen.getByTestId("task-live").textContent).toContain("已派发，等待子代理回传进展");
    expect(screen.getByText("后台")).toBeTruthy();
  });

  it("有唯一会话目标时，点击卡片进入会话且不在卡内展开输出", () => {
    const opened: string[] = [];
    setTaskCardOpenHandler((ref) => opened.push(ref));
    setTaskCardOpenTarget((ref) => ref);
    const card: Item = {
      kind: "tool",
      id: "d3",
      name: "task",
      args: '{"description":"成本复核"}',
      readOnly: false,
      status: "done",
      output: "成本复核完成。\nSubagent reference: sa_x3",
    };
    renderT(<ToolCard item={card} />);
    expect(screen.getByText("查看详情")).toBeTruthy();
    expect(screen.queryByText(/输出 · /)).toBeNull();
    fireEvent.click(screen.getByTestId("delegate-head"));
    expect(opened).toEqual(["sa_x3"]);
  });
});
