// SubagentThread — 子代理对话全面板视图（v4.27 Codex 式点击下钻）测试：
// 消息流渲染（system/user/assistant 思考+正文/tool）、返回回调、运行中
// 3s 轮询实时刷新（fake timers 确定性）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, act, cleanup } from "@testing-library/react";
import { SubagentThread } from "./SubagentThread";
import { LocaleProvider } from "../lib/i18n";
import type { SubagentTranscriptView } from "../lib/types";

const transcript: SubagentTranscriptView = {
  ref: "sa_2_b2b2b2b2",
  task: "调研竞品表格 Agent 能力并总结可蒸馏点",
  messages: [
    { role: "system", content: "你是子代理，专注完成派发任务。" },
    { role: "user", content: "调研竞品表格 Agent 能力并总结可蒸馏点" },
    { role: "assistant", reasoning: "先检索竞品资料", content: "开始检索公开信息。" },
    { role: "assistant", toolCalls: [{ id: "call_1", name: "web_fetch", arguments: "{\"url\":\"https://example.com\"}" }] },
    { role: "tool", name: "web_fetch", toolCallId: "call_1", content: "三家竞品表格交互结论：A 支持选区即图表。" },
    { role: "assistant", content: "正在比对三家竞品的表格选中→图表链路…" },
  ],
};

const mocks = vi.hoisted(() => ({
  SubagentTranscript: vi.fn(),
  SubagentFollowUp: vi.fn(),
  onEvent: vi.fn(() => () => {}),
  onSubagentText: vi.fn(() => () => {}),
}));

vi.mock("../lib/bridge", () => ({
  app: mocks,
  onEvent: mocks.onEvent,
  onSubagentText: mocks.onSubagentText,
}));

// SubagentThread 走 useT：钉住 zh 让「进行中/N 条/思考」等中文断言继续成立
const wrap = (node: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{node}</LocaleProvider>;
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.SubagentTranscript.mockResolvedValue(transcript);
});

afterEach(() => {
  vi.useRealTimers();
});

describe("SubagentThread 子代理对话全面板（v4.27）", () => {
  it("渲染消息流：system 弱化 / user 右对齐 / assistant 思考+正文 / tool 结果", async () => {
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="调研竞品表格 Agent 能力并总结可蒸馏点" status="running" onBack={() => {}} />));

    expect(await screen.findByText("你是子代理，专注完成派发任务。")).toBeTruthy();
    // 任务标题（头部）与 user 消息正文同文案 → getAllByText
    expect(screen.getAllByText("调研竞品表格 Agent 能力并总结可蒸馏点").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("开始检索公开信息。")).toBeTruthy();
    // 工具调用行与 tool 结果消息都可能含 web_fetch → 用 getAllByText
    expect(screen.getAllByText(/web_fetch/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/三家竞品表格交互结论/).length).toBeGreaterThan(0);
    // 头部：状态徽标 + 消息计数 + 刷新按钮
    expect(screen.getByText("进行中")).toBeTruthy();
    expect(screen.getByText("6 条")).toBeTruthy();
    // 思考块默认折叠（AssistantMessage 思考体常驻 DOM、CSS 折叠呈现，
    // 以 aria-expanded 断言状态；v4.63 与主对话同款渲染器）
    const thinkBtns = screen.getAllByRole("button", { name: /思考/ });
    expect(thinkBtns.length).toBeGreaterThanOrEqual(1);
    expect(thinkBtns.every((b) => b.getAttribute("aria-expanded") === "false")).toBe(true);
    fireEvent.click(thinkBtns[0]);
    expect(thinkBtns[0].getAttribute("aria-expanded")).toBe("true");
  });

  it("头部返回按钮触发 onBack", async () => {
    const onBack = vi.fn();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={onBack} />));
    await screen.findByText("任务");
    fireEvent.click(screen.getByRole("button", { name: /分工/ }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("已完成：只拉取一次，不轮询", async () => {
    vi.useFakeTimers();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(1);
    mocks.SubagentTranscript.mockClear();
    act(() => { vi.advanceTimersByTime(9000); });
    expect(mocks.SubagentTranscript).not.toHaveBeenCalled();
  });

  it("运行中：每 3s 轮询刷新对话（实时显示）", async () => {
    vi.useFakeTimers();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="running" onBack={() => {}} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(1);
    mocks.SubagentTranscript.mockClear();

    act(() => { vi.advanceTimersByTime(3000); });
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(1);

    act(() => { vi.advanceTimersByTime(6000); });
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(3);
  });

  it("assistant 正文按主对话同款 Markdown 渲染（加粗/列表/代码块）", async () => {
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      messages: [
        ...transcript.messages,
        {
          role: "assistant",
          content:
            "**重点结论**：方案可行。\n\n- 要点一：本地保真\n- 要点二：零绑定面\n\n```ts\nconst ok = true\n```",
        },
      ],
    });
    render(
      wrap(
        <SubagentThread
          sessionPath="s1.jsonl"
          target="sa_2_b2b2b2b2"
          task="任务"
          status="completed"
          onBack={() => {}}
        />,
      ),
    );
    // 加粗文字与列表项由 react-markdown 结构化成独立元素（非整段纯文本）
    expect(await screen.findByText("重点结论", { selector: "strong" })).toBeTruthy();
    expect(screen.getByText("要点一：本地保真")).toBeTruthy();
    expect(screen.getByText("要点二：零绑定面")).toBeTruthy();
    expect(screen.getByText("const ok = true")).toBeTruthy();
  });

  // v4.62 P1 逐 token 流式（v4.62.1 分道）：增量经专用通道 onSubagentText
  // 按 ref 路由到本会话 tab，以实时行渲染在消息流尾部；权威快照追上后缓冲
  // 让位（reconcile）。
  it("运行中 subagent_text 增量实时渲染，快照接管后缓冲清空", async () => {
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="running" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");

    // 收集专用通道订阅回调（流式），事件广播给每一个
    type SubEventCb = (e: { kind?: string; text?: string; subagentRef?: string }) => void;
    const cbs = (mocks.onSubagentText.mock.calls as unknown as SubEventCb[][]).map((c) => c[0]);
    expect(cbs.length).toBeGreaterThanOrEqual(1);

    // 他人会话的增量不入缓冲
    act(() => {
      cbs.forEach((cb) => cb({ kind: "subagent_text", text: "别家增量", subagentRef: "sa_other" }));
    });
    // 本会话增量逐块追加
    act(() => {
      cbs.forEach((cb) => cb({ kind: "subagent_text", text: "正在比对三家竞品", subagentRef: "sa_2_b2b2b2b2" }));
      cbs.forEach((cb) => cb({ kind: "subagent_text", text: "的表格链路…", subagentRef: "sa_2_b2b2b2b2" }));
    });
    expect(screen.getByTestId("agent-thread-streaming")).toBeTruthy();
    expect(screen.getByTestId("agent-thread-streaming").textContent).toContain("正在比对三家竞品的表格链路…");
    expect(screen.getByTestId("agent-thread-streaming").textContent).not.toContain("别家增量");

    // 快照追上（尾条 assistant 已含缓冲开头）→ 缓冲清空，权威渲染接管
    //（正文经 MemoMarkdown 拆元素，改以消息计数与缓冲行消失断言）
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      messages: [
        ...transcript.messages,
        { role: "assistant", content: "正在比对三家竞品的表格链路…对比结论如下。" },
      ],
    });
    fireEvent.click(screen.getByRole("button", { name: "刷新对话" }));
    await screen.findByText("7 条");
    expect(screen.queryByTestId("agent-thread-streaming")).toBeNull();
  });

  // v4.62.1 回归钉子：流式增量绝不上 gaea-event（seq↔账本 1:1，上去了会
  // 制造不可愈合缺口触发反复 resync 打断对话窗）——SubagentThread 不得从
  // onEvent 订阅流式。
  it("流式订阅走专用通道：onEvent 只收工具事件，不承担 subagent_text", async () => {
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="running" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");

    // gaea-event 通道（onEvent）收到 subagent_text 时，组件不得进入流式渲染
    type SubEventCb = (e: { kind?: string; text?: string; subagentRef?: string }) => void;
    const eventCbs = (mocks.onEvent.mock.calls as unknown as SubEventCb[][]).map((c) => c[0]);
    act(() => {
      eventCbs.forEach((cb) => cb({ kind: "subagent_text", text: "不该被流式消费", subagentRef: "sa_2_b2b2b2b2" }));
    });
    expect(screen.queryByTestId("agent-thread-streaming")).toBeNull();
    // 专用通道订阅确实已建立
    expect(mocks.onSubagentText).toHaveBeenCalled();
  });

  // v4.63 mt_/长文本有界输出（Codex 式）：mt_ 标签页的文档级输出默认限高
  // 滚动，「展开全部/收起」切换 + 字数标注；不再全量铺开成一面墙。
  it("mt_ 标签页长输出有界渲染：默认限高，展开后可收起", async () => {
    const long = ["段落", "段落"].join("\n\n").repeat(300);
    mocks.SubagentTranscript.mockResolvedValue({
      ref: "mt_9",
      task: "摘要文档",
      messages: [
        { role: "user", content: "summarize_file · 摘要 文档.docx" },
        { role: "assistant", content: long },
      ],
    });
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="mt_9" task="摘要文档" status="completed" onBack={() => {}} />));
    await screen.findByText("摘要文档");
    expect(screen.getByTestId("agent-thread-bounded")).toBeTruthy();
    expect(screen.getByTestId("agent-thread-bounded-toggle").textContent).toContain("展开全部");
    fireEvent.click(screen.getByTestId("agent-thread-bounded-toggle"));
    expect(screen.getByTestId("agent-thread-bounded-toggle").textContent).toContain("收起");
  });

  // v4.64 Side Chat 式追问：sa_ 会话 tab 有追问输入框；发送即乐观上屏 +
  // 调用绑定；mt_ 运行不提供追问。
  it("追问输入框：发送派发绑定并乐观上屏；mt_ 运行隐藏输入框", async () => {
    mocks.SubagentFollowUp.mockResolvedValue("follow-up started");
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "再补充一下第二点" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));
    expect(mocks.SubagentFollowUp).toHaveBeenCalledWith("s1.jsonl", "sa_2_b2b2b2b2", "再补充一下第二点");
    // 乐观气泡可见；快照未变化时持续显示
    expect(screen.getByTestId("agent-follow-up-pending")).toBeTruthy();

    // mt_ 运行不提供追问
    cleanup();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="mt_9" task="摘要" status="completed" onBack={() => {}} />));
    await screen.findByText("摘要");
    expect(screen.queryByTestId("agent-follow-up-input")).toBeNull();
  });
});

// v4.64.x 追问失败诚实化：派发被守卫拒绝时错误条内联展示原因，乐观气泡
// 标失败态保留原文本并内联「重试/撤销」；重试复用 followUpBusy 守卫；
// 失败态气泡不被无关快照增长误清（清掉会让重试入口凭空消失）。
describe("SubagentThread 追问失败内联展示与重试（v4.64.x）", () => {
  it("追问派发失败：错误条内联展示原因，气泡标失败态并保留原文", async () => {
    mocks.SubagentFollowUp.mockRejectedValueOnce(new Error("主对话回合正在运行，请等回合结束再追问"));
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "再补充一下第二点" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));

    // 错误条内联展示失败原因（追问失败：{msg}）
    const errBar = await screen.findByTestId("agent-follow-up-error");
    expect(errBar.textContent).toContain("追问失败");
    expect(errBar.textContent).toContain("主对话回合正在运行，请等回合结束再追问");

    // 乐观气泡转失败态：文本保留（不丢用户输入），不再是无失败标注的等待态
    const bubble = screen.getByTestId("agent-follow-up-pending");
    expect(bubble.getAttribute("data-failed")).toBe("true");
    expect(bubble.textContent).toContain("再补充一下第二点");
    // 重试入口可见
    expect(screen.getByTestId("agent-follow-up-retry").textContent).toBe("重试");
  });

  it("失败气泡一键重试：重发同一文本、in-flight 期间 busy 守卫拦截重复派发，成功后快照接管清气泡", async () => {
    let resolveRetry!: (v: string) => void;
    mocks.SubagentFollowUp
      .mockRejectedValueOnce(new Error("该子代理已有追问正在运行"))
      .mockImplementationOnce(() => new Promise<string>((res) => { resolveRetry = res; }));
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "再补充一下第二点" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));
    await screen.findByTestId("agent-follow-up-error");

    // 快照将在重试成功后带回真实内容（6 条 → 8 条）
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      messages: [
        ...transcript.messages,
        { role: "user", content: "再补充一下第二点" },
        { role: "assistant", content: "第二点的补充如下…" },
      ],
    });
    fireEvent.click(screen.getByTestId("agent-follow-up-retry"));

    // 重发同一文本；重试点击即进入 busy：失败行退场、气泡回到等待态（非假象）
    expect(mocks.SubagentFollowUp).toHaveBeenCalledTimes(2);
    expect(mocks.SubagentFollowUp).toHaveBeenLastCalledWith("s1.jsonl", "sa_2_b2b2b2b2", "再补充一下第二点");
    expect(screen.queryByTestId("agent-follow-up-retry")).toBeNull();
    expect(screen.getByTestId("agent-follow-up-pending").getAttribute("data-failed")).toBeNull();

    // busy 守卫（与发送共用同一闸）：in-flight 期间再次派发被拦下，不重复调用
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "in-flight 抢跑" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));
    expect(mocks.SubagentFollowUp).toHaveBeenCalledTimes(2);
    // 乐观气泡不被抢跑文本覆盖，仍是重试中的那条
    expect(screen.getByTestId("agent-follow-up-pending").textContent).toContain("再补充一下第二点");

    resolveRetry("follow-up started");
    // 真实内容落盘 → 乐观气泡与错误条一并退场
    await screen.findByText("8 条");
    expect(screen.queryByTestId("agent-follow-up-pending")).toBeNull();
    expect(screen.queryByTestId("agent-follow-up-error")).toBeNull();
  });

  it("失败态气泡不被无关快照增长误清；撤销同时清掉气泡与错误条", async () => {
    mocks.SubagentFollowUp.mockRejectedValueOnce(new Error("子代理追问未接线（引擎尚未构建完成）"));
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "帮忙核对数据" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));
    await screen.findByTestId("agent-follow-up-error");

    // 无关快照增长（其他来源的新消息）不清失败气泡 —— 重试入口不能凭空消失
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      messages: [...transcript.messages, { role: "assistant", content: "别处落盘的新内容" }],
    });
    fireEvent.click(screen.getByRole("button", { name: "刷新对话" }));
    await screen.findByText("7 条");
    expect(screen.getByTestId("agent-follow-up-pending").getAttribute("data-failed")).toBe("true");
    expect(screen.getByTestId("agent-follow-up-pending").textContent).toContain("帮忙核对数据");

    // 撤销：气泡与错误条一并清掉
    fireEvent.click(screen.getByTestId("agent-follow-up-dismiss"));
    expect(screen.queryByTestId("agent-follow-up-pending")).toBeNull();
    expect(screen.queryByTestId("agent-follow-up-error")).toBeNull();
  });
});

// v4.66 追问后台失败可感知：受理成功但后台 runner 真正失败时，失败原因经
// meta（followUpError）随 transcript 快照轮询带回——等待态气泡转失败态
//（错误条文案 = 该原因，保留重试/撤销），不再永久「等待中」。失败判定优先
// 于「快照增长 = 成功」：失败前已落盘的部分内容不得把失败误判成成功。
describe("SubagentThread 追问后台失败经 meta 带回（v4.66）", () => {
  it("轮询快照带 followUpError：气泡转失败态并展示原因；重试成功后一并退场", async () => {
    mocks.SubagentFollowUp.mockResolvedValue("follow-up started");
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "再补充一下第二点" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));
    await screen.findByTestId("agent-follow-up-pending");
    await act(async () => { await Promise.resolve(); }); // 派发后首轮 load 落定

    // 后台失败：meta 写回 followUpError，消息数未变（该次无内容落盘）
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      followUpError: "provider 掉线：context deadline exceeded",
    });
    fireEvent.click(screen.getByRole("button", { name: "刷新对话" }));

    const errBar = await screen.findByTestId("agent-follow-up-error");
    expect(errBar.textContent).toContain("追问失败");
    expect(errBar.textContent).toContain("provider 掉线");
    const bubble = screen.getByTestId("agent-follow-up-pending");
    expect(bubble.getAttribute("data-failed")).toBe("true");
    expect(bubble.textContent).toContain("再补充一下第二点");
    expect(screen.getByTestId("agent-follow-up-retry")).toBeTruthy();
    expect(screen.getByTestId("agent-follow-up-dismiss")).toBeTruthy();

    // 重试成功：新一轮派发清旧摘要，快照带回真实内容 → 气泡与错误条退场
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      messages: [
        ...transcript.messages,
        { role: "user", content: "再补充一下第二点" },
        { role: "assistant", content: "第二点补充如下…" },
      ],
    });
    fireEvent.click(screen.getByTestId("agent-follow-up-retry"));
    expect(mocks.SubagentFollowUp).toHaveBeenCalledTimes(2);
    expect(mocks.SubagentFollowUp).toHaveBeenLastCalledWith("s1.jsonl", "sa_2_b2b2b2b2", "再补充一下第二点");
    await screen.findByText("8 条");
    expect(screen.queryByTestId("agent-follow-up-pending")).toBeNull();
    expect(screen.queryByTestId("agent-follow-up-error")).toBeNull();
  });

  it("失败判定优先于快照增长：失败前已落部分内容不被误判成功", async () => {
    mocks.SubagentFollowUp.mockResolvedValue("follow-up started");
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("开始检索公开信息。");
    fireEvent.change(screen.getByTestId("agent-follow-up-input"), { target: { value: "帮忙核对数据" } });
    fireEvent.click(screen.getByTestId("agent-follow-up-send"));
    await screen.findByTestId("agent-follow-up-pending");
    await act(async () => { await Promise.resolve(); });

    // 同一份快照：既带了失败摘要、又比基线多了部分内容 → 必须按失败处理
    //（气泡保留失败态 + 重试入口），不能按「内容已落盘」清成成功。
    mocks.SubagentTranscript.mockResolvedValue({
      ...transcript,
      followUpError: "生成中断：上下文超限",
      messages: [
        ...transcript.messages,
        { role: "user", content: "帮忙核对数据" },
        { role: "assistant", content: "部分输出…" },
      ],
    });
    fireEvent.click(screen.getByRole("button", { name: "刷新对话" }));

    const errBar = await screen.findByTestId("agent-follow-up-error");
    expect(errBar.textContent).toContain("生成中断");
    const bubble = screen.getByTestId("agent-follow-up-pending");
    expect(bubble.getAttribute("data-failed")).toBe("true");
    expect(bubble.textContent).toContain("帮忙核对数据");
    expect(screen.getByTestId("agent-follow-up-retry")).toBeTruthy();
  });
});

describe("SubagentThread 失败恢复入口（v4.93）", () => {
  it("status=failed 显示续跑提示条；completed/running 不显示", async () => {
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="failed" onBack={() => {}} />));
    const hint = await screen.findByTestId("agent-recover-hint");
    expect(hint.textContent).toContain("续跑");
    expect(hint.textContent).toContain("失败");
  });

  it("status=completed 无恢复提示", async () => {
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await screen.findByText("任务");
    expect(screen.queryByTestId("agent-recover-hint")).toBeNull();
  });
});
