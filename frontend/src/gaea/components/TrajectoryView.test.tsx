import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Trajectory } from "../lib/types";

const trajectoryMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { Trajectory: trajectoryMock },
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
}));

const TRAJECTORY: Trajectory = {
  ok: true,
  turns: [
    {
      turn: 1,
      startedAt: 1750000000,
      end: { seq: 30, ts: 1750000100 },
      records: [
        { seq: 2, kind: "user", ts: 1750000001, user: { text: "调研 internal/gaea/config" } },
        {
          seq: 3, kind: "header", ts: 1750000002, step: 1,
          header: { system: "系统提示词预览", toolCount: 50, tokens: 12500, change: "initial" },
        },
        {
          seq: 4, kind: "assistant", ts: 1750000003, step: 1,
          assistant: { reasoning: "先搜索", text: "配置集中在 config.go", usage: { promptTokens: 350600, completionTokens: 93 } },
        },
        {
          seq: 8, kind: "tool", ts: 1750000004, step: 1, durationMs: 3200,
          tool: { id: "t1", name: "pwsh", args: "{\"command\":\"git status\"}", output: "M App.tsx", status: "ok" },
        },
        {
          seq: 12, kind: "tool", ts: 1750000008, step: 1, durationMs: 1500,
          tool: { id: "t2", name: "bash", args: "rm -rf /", output: "", err: "denied by policy", status: "error" },
        },
        { seq: 15, kind: "ask", ts: 1750000010, step: 1, ask: { question: "如何协调并行改动？" } },
      ],
    },
  ],
  betweenTurns: [
    { seq: 31, kind: "compact", ts: 1750000100, compact: { trigger: "manual", summary: "轮间压缩" } },
  ],
};

describe("TrajectoryView 轨迹事件账本", () => {
  // 全量套件高负载下 jsdom 调度会被饿死，RTL 默认 1s 超时出现过 flaky；
  // 显式放宽到 5s（仍有上界，不会掩盖真回归）。
  const LOAD = { timeout: 5000 };

  beforeEach(() => {
    trajectoryMock.mockReset();
    trajectoryMock.mockResolvedValue(TRAJECTORY);
  });

  it("渲染统计 chips、轮次与记录行", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/Turns 1/, undefined, LOAD)).toBeTruthy();
    expect(screen.getByText(/Calls 2/)).toBeTruthy();
    expect(screen.getByText(/Duration/)).toBeTruthy();
    expect(screen.getByText("第1轮")).toBeTruthy();
    expect(screen.getByText("ASSISTANT")).toBeTruthy();
    expect(screen.getAllByText("TOOL").length).toBe(2);
    expect(screen.getByText(/调研 internal\/gaea\/config/)).toBeTruthy();
  });

  it("展开工具记录显示参数/结果/错误", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    // 第一个 TOOL 行的按钮：文本含 pwsh
    const btn = await screen.findByRole("button", { name: /pwsh/ }, LOAD);
    fireEvent.click(btn);
    expect((await screen.findAllByText(/"command":"git status"/, undefined, LOAD)).length).toBeGreaterThan(0);
    expect((await screen.findAllByText("M App.tsx", undefined, LOAD)).length).toBeGreaterThan(0);
    expect(await screen.findByText(/denied by policy/, undefined, LOAD)).toBeTruthy();
  });

  it("渲染 ask 与 Between-turns 压缩记录", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/如何协调并行改动？/, undefined, LOAD)).toBeTruthy();
    expect(screen.getByText(/Between turns/)).toBeTruthy();
    expect(screen.getByText(/轮间压缩/)).toBeTruthy();
  });

  it("搜索过滤记录", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    const input = await screen.findByPlaceholderText("搜索", undefined, LOAD);
    fireEvent.change(input, { target: { value: "git status" } });
    expect(screen.getByText(/pwsh/)).toBeTruthy();
    expect(screen.queryByText(/配置集中在 config.go/)).toBeNull();
  });

  it("空态提示", async () => {
    trajectoryMock.mockResolvedValue({ ok: true, turns: [] });
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/暂无轨迹记录/, undefined, LOAD)).toBeTruthy();
  });

  it("轨迹概览：渲染投影柱与轮数", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText("轨迹概览", undefined, LOAD)).toBeTruthy();
    expect(screen.getByText(/1 轮 · 点击柱跳转/)).toBeTruthy();
    expect(screen.getByText(/柱高 = 记录密度/)).toBeTruthy();
  });

  it("收起全部 / 展开全部控制轮次区段", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    await screen.findByText(/Turns 1/, undefined, LOAD);
    expect(screen.getByText("ASSISTANT")).toBeTruthy();
    fireEvent.click(screen.getByText("收起全部"));
    expect(screen.queryByText("ASSISTANT")).toBeNull();
    expect(screen.getByText("第1轮")).toBeTruthy();
    fireEvent.click(screen.getByText("展开全部"));
    expect(screen.getByText("ASSISTANT")).toBeTruthy();
  });

  it("超长会话虚拟化：DOM 有界 + 滚动到末尾可见", async () => {
    // 300 条记录的会话：扁平行流 = 6 个轮次头 + 300 记录 = 306 行。
    const bigTurns: Trajectory["turns"] = Array.from({ length: 6 }, (_, ti) => ({
      turn: ti + 1,
      startedAt: 1750000000 + ti,
      end: { seq: 100 + ti, ts: 1750000100 + ti },
      records: Array.from({ length: 50 }, (_, ri) => ({
        seq: ti * 50 + ri + 1,
        kind: "user" as const,
        ts: 1750000000 + ti * 50 + ri,
        user: { text: `第${ti + 1}轮第${ri + 1}条` },
      })),
    }));
    trajectoryMock.mockResolvedValue({ ok: true, turns: bigTurns });
    const { TrajectoryView } = await import("./TrajectoryView");
    const { container } = render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/第1轮第1条/, undefined, LOAD)).toBeTruthy();
    // 虚拟化：306 行只渲染可见窗口（±overscan），DOM 行数远小于全量
    const listEl = container.querySelector('[role="list"]');
    if (!listEl) throw new Error("virtual list not found");
    const itemCount = container.querySelectorAll('[role="listitem"]').length;
    expect(itemCount).toBeLessThan(80);
    expect(screen.queryByText(/第6轮第50条/)).toBeNull(); // 视口外未渲染
    // 滚到底部（jsdom 无布局，直接覆写 scrollTop 再派发 scroll）
    Object.defineProperty(listEl, "scrollTop", { value: 8600, configurable: true, writable: true });
    fireEvent.scroll(listEl);
    expect(await screen.findByText(/第6轮第50条/, undefined, LOAD)).toBeTruthy();
  }, 15000);
});

// ── v4.28 子代理答复记录（kind="subagent"）：折叠弱存在感行 + 点击展开全文 ──
describe("TrajectoryView 子代理答复记录", () => {
  const LOAD = { timeout: 5000 };

  // 一条带 ref/parentId（task 派生的正式子代理）、一条无 ref（临时子代理容错）。
  const SUBAGENT_TRAJECTORY: Trajectory = {
    ok: true,
    turns: [
      {
        turn: 1,
        startedAt: 1750000000,
        end: { seq: 40, ts: 1750000100 },
        records: [
          { seq: 2, kind: "user", ts: 1750000001, user: { text: "帮我调研配置分布" } },
          {
            seq: 10, kind: "subagent", ts: 1750000005,
            subagent: { ref: "subagent:run-7f3a", text: "调研完成：配置集中在 internal/gaea/config，共 3 处入口。", parentId: "task-call-9" },
          },
          {
            seq: 11, kind: "subagent", ts: 1750000006,
            subagent: { text: "第二轮检查完毕：无遗漏文件。" },
          },
        ],
      },
    ],
  };

  beforeEach(() => {
    trajectoryMock.mockReset();
    trajectoryMock.mockResolvedValue(SUBAGENT_TRAJECTORY);
  });

  it("折叠行渲染「子代理」徽标 + 答复摘要 + ref，默认不展开详情", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    // 两条 subagent 记录各一个徽标
    expect((await screen.findAllByText("子代理", undefined, LOAD)).length).toBe(2);
    expect(screen.getByText(/调研完成：配置集中在 internal\/gaea\/config，共 3 处入口。 · subagent:run-7f3a/)).toBeTruthy();
    expect(screen.getByText(/第二轮检查完毕：无遗漏文件。/)).toBeTruthy(); // 无 ref 摘要后缀留空
    // 折叠态：详情区（ref:/parentId: 元信息）不渲染
    expect(screen.queryByText(/ref: subagent:run-7f3a/)).toBeNull();
    expect(screen.queryByText(/parentId: task-call-9/)).toBeNull();
  });

  it("点击展开显示完整答复 + ref/parentId", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    fireEvent.click(await screen.findByRole("button", { name: /调研完成/ }, LOAD));
    // 详情区元信息 + 全文 pre（与折叠摘要行各一份，共 2 处命中）
    expect(await screen.findByText("ref: subagent:run-7f3a", undefined, LOAD)).toBeTruthy();
    expect(screen.getByText("parentId: task-call-9")).toBeTruthy();
    expect(screen.getAllByText(/共 3 处入口/).length).toBe(2);
  });

  it("无 ref/parentId（临时子代理）容错：展开只显示全文不炸", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    fireEvent.click(await screen.findByRole("button", { name: /第二轮检查完毕/ }, LOAD));
    // 无 ref/parentId：详情区不渲染元信息行，全文与摘要同文（2 处命中）
    await screen.findAllByText(/第二轮检查完毕：无遗漏文件。/, undefined, LOAD);
    expect(screen.getAllByText(/第二轮检查完毕：无遗漏文件。/).length).toBe(2);
    expect(screen.queryByText(/^ref: /)).toBeNull();
    expect(screen.queryByText(/^parentId: /)).toBeNull();
  });

  it("搜索命中子代理答复文本", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    const input = await screen.findByPlaceholderText("搜索", undefined, LOAD);
    fireEvent.change(input, { target: { value: "共 3 处入口" } });
    expect(await screen.findByText(/调研完成/, undefined, LOAD)).toBeTruthy();
    expect(screen.queryByText(/帮我调研配置分布/)).toBeNull();
  });
});
