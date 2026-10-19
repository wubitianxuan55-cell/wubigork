// v4.504 ToolGroup 行内主体串截断：bash 组的 subject 是整条命令，三条拼一行
// 又成「一长串」——单条截 48 字符（与单卡折叠行截断口径一致，完整命令在
// 子卡 $ 块里）。顺带钉 scanGroups 的分组语义（≥2 连续同名才成组）。
import { describe, expect, it, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import { ToolGroup, scanGroups } from "./ToolGroup";
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

  it("超长命令截 48 字符加省略号，不再整条上屏", () => {
    renderT(<ToolGroup tools={[bashItem("b1", longCmd), bashItem("b2", "ls -la")]} />);
    const row = screen.getByText(/bash × 2/);
    expect(row).toBeTruthy();
    // 期望串与渲染同口径构造（slice(0,48)+…），不手算中英混排下标
    expect(screen.getByText(`${longCmd.slice(0, 48)}…, ls -la`)).toBeTruthy();
    expect(screen.queryByText(longCmd)).toBeNull();
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
});
