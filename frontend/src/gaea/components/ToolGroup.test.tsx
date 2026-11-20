// v4.504 ToolGroup 行内主体串截断：bash 组的 subject 是整条命令，三条拼一行
// 又成「一长串」。v4.507 收口单源：subjectOf 里 bash 先经 commandHeadline
// 首行化+词界截断，组行 cap 幂等跳过已截串（… 结尾）不再二次截。
// 顺带钉 scanGroups 的分组语义（≥2 连续同名才成组）。
import { describe, expect, it, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import { ToolGroup, scanGroups } from "./ToolGroup";
import { commandHeadline } from "../lib/tools";
import { LocaleProvider } from "../lib/i18n";
import type { Item } from "../lib/store";

const renderT = (ui: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const longCmd = `python tools/settlement_adjust.py --network "data/2026Q3/三期联测.xlsx" --datum J1 --method indirect`;
// 返回窄类型（工具卡面），需要 Item 联合的用例处再放宽——工具卡族测试既有口径
const bashItem = (id: string, cmd: string): Extract<Item, { kind: "tool" }> => ({
  kind: "tool",
  id,
  name: "bash",
  args: JSON.stringify({ command: cmd }),
  readOnly: false,
  status: "done",
  output: "ok\n",
});

describe("ToolGroup 行内主体截断", () => {
  afterEach(cleanup);

  it("超长命令经 commandHeadline 词界截断，组行不再二次截", () => {
    renderT(<ToolGroup tools={[bashItem("b1", longCmd), bashItem("b2", "ls -la")]} />);
    const row = screen.getByText(/bash × 2/);
    expect(row).toBeTruthy();
    // 期望串与渲染同口径构造（subjectOf→commandHeadline），不手算中英混排下标
    expect(screen.getByText(`${commandHeadline(longCmd)}, ls -la`)).toBeTruthy();
    expect(screen.queryByText(longCmd)).toBeNull();
    // 词界截断回归钉：截断点必须落在原串的空白处（硬 slice 会切在词中间）
    const h = commandHeadline(longCmd);
    expect(h.endsWith("…")).toBe(true);
    expect(longCmd[h.length - 1]).toMatch(/\s/);
  });

  it("多行命令只显首行（heredoc/python -c 不再压平上屏）", () => {
    const multi = `python -c "\nimport json\nprint(json.dumps({"a": 1}))\n"`;
    renderT(<ToolGroup tools={[bashItem("b1", multi), bashItem("b2", "ls -la")]} />);
    // getByText 精确匹配：若整条命令压平上屏，该精确文本节点即不存在
    const row = screen.getByText(`${commandHeadline(multi)}, ls -la`);
    expect(row.textContent).not.toContain("import json");
  });

  it("短主体原样显示不受影响", () => {
    renderT(<ToolGroup tools={[bashItem("b1", "ls -la"), bashItem("b2", "pwd")]} />);
    expect(screen.getByText("ls -la, pwd")).toBeTruthy();
  });
});

describe("scanGroups 分组语义（既有口径回归钉）", () => {
  it("≥2 连续同名工具成组；异名/单条不成组", () => {
    const items = [
      bashItem("a", "x"),
      bashItem("b", "y"),
      { kind: "tool", id: "c", name: "read_file", args: "{}", readOnly: true, status: "done" } as Item,
    ];
    const gs = scanGroups(items);
    expect(gs.filter((g) => g.kind === "group").length).toBe(1);
    expect(gs.filter((g) => g.kind === "item").length).toBe(1);
  });

  // 2026-11-18 委派家族不折叠（ce5fc790 补钉）：并行派发=连续同名 task 调用，
  // 折成「task × N」组行会吞掉委派卡渲染。skipRunning 两态都必须不折。
  // 2026-11-20 家族收窄：task 恒委派不折；run_skill 走调用级证据——无回执
  //（inline/运行中）按普通工具可折，任一调用带子代理引用行整组不折。
  it("task 连续同名不折叠（每调用独立成卡）；run_skill 无回执可折、带引用行不折", () => {
    const mk = (id: string, name: string, output?: string, status: "done" | "running" = "done") =>
      ({ kind: "tool", id, name, args: "{}", readOnly: false, status, output } as Item);
    for (const opts of [undefined, { skipRunning: true }]) {
      const gs = scanGroups([mk("t1", "task"), mk("t2", "task"), mk("t3", "task")], opts);
      expect(gs.every((g) => g.kind === "item"), `task ${JSON.stringify(opts)}`).toBe(true);
      expect(gs.length).toBe(3);
    }
    // run_skill 无回执（inline 形态）：普通折叠语义
    const inline = scanGroups([mk("s1", "run_skill"), mk("s2", "run_skill")]);
    expect(inline).toHaveLength(1);
    expect(inline[0].kind).toBe("group");
    // run_skill 带引用行（subagent 形态回执）：委派卡不折
    const sub = scanGroups([mk("s1", "run_skill", "ok\nSubagent reference: sa_1"), mk("s2", "run_skill")]);
    expect(sub.every((g) => g.kind === "item")).toBe(true);
    expect(sub).toHaveLength(2);
  });
});
