import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import { XlsxPreview } from "./XlsxPreview";
import { usePreviewStore } from "../lib/store";

const body = JSON.stringify({
  sheets: [
    {
      name: "预算",
      rows: [
        [
          { ref: "A1", value: "项目", type: "string", style: { bold: true, fill: "4472C4" } },
          { ref: "B1", value: "金额", type: "string" },
        ],
        [{ ref: "A2", value: "设备", type: "string" }, { ref: "B2", value: "120.5", type: "number" }],
        [{ ref: "A3", value: "合计", type: "string" }, { ref: "B3", value: "120.5", formula: "SUM(B2)", type: "string" }],
      ],
      merged: [],
      colWidths: { A: 16, B: 14 },
    },
    { name: "明细", rows: [[{ ref: "A1", value: "日期", type: "string" }]], colWidths: {} },
  ],
});

describe("XlsxPreview", () => {
  it("渲染 sheet 切换与单元格", () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    expect(screen.getByText("预算")).toBeTruthy();
    expect(screen.getByText("明细")).toBeTruthy();
    expect(screen.getByText("项目")).toBeTruthy();
    expect(screen.getAllByText("120.5").length).toBeGreaterThanOrEqual(1);
  });

  it("点击公式单元格在公式栏显示公式", () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByTitle("=SUM(B2)"));
    const fx = screen.getByPlaceholderText(/选中单元格后输入值/) as HTMLInputElement;
    expect(fx.value).toBe("=SUM(B2)");
  });

  it("双击单元格直接编辑并写回（mock）", async () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.doubleClick(screen.getByText("设备"));
    const grid = document.querySelector(".docx-preview-body") as HTMLElement;
    const editInput = within(grid).getByDisplayValue("设备") as HTMLInputElement;
    fireEvent.change(editInput, { target: { value: "设备-新" } });
    fireEvent.keyDown(editInput, { key: "Enter" });
    expect(await screen.findByText(/已更新 A2/)).toBeTruthy();
  });

  it("按 NumFmt 格式化数值显示", () => {
    const fmtBody = JSON.stringify({
      sheets: [
        {
          name: "S",
          rows: [
            [{ ref: "A1", value: "9992806.15", type: "number", style: { numFmt: "4" } }],
            [{ ref: "A2", value: "0.125", type: "number", style: { numFmt: "10" } }],
            [{ ref: "A3", value: "400", type: "number" }],
          ],
        },
      ],
    });
    render(<XlsxPreview body={fmtBody} fileName="t.xlsx" relPath="mock.xlsx" />);
    expect(screen.getByText("9,992,806.15")).toBeTruthy();
    expect(screen.getByText("12.50%")).toBeTruthy();
    expect(screen.getByText("400")).toBeTruthy();
  });

  it("选中单元格后插入行（mock）", async () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByText("设备"));
    fireEvent.click(screen.getByTitle("在选中行上方插入空行"));
    expect(await screen.findByText(/insert_before@A2/)).toBeTruthy();
  });

  it("选中列后插入列（mock）", async () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    const headers = document.querySelectorAll("thead th");
    fireEvent.click(headers[1]); // 列 A
    expect(screen.getByText("列 A")).toBeTruthy();
    fireEvent.click(screen.getByText("← 插列"));
    expect(await screen.findByText(/insert_before@A1/)).toBeTruthy();
  });


  it("切换 sheet", () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByText("明细"));
    expect(screen.getByText("日期")).toBeTruthy();
  });

  it("选中单元格后执行指令并显示摘要（mock）", async () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByText("项目"));
    const input = screen.getByPlaceholderText(/输入指令/);
    fireEvent.change(input, { target: { value: "求和" } });
    fireEvent.click(screen.getByText("执行"));
    expect(await screen.findByText(/mock.*求和/)).toBeTruthy();
  });

  it("点击预设按钮回填指令到输入框（AI 编辑收敛为单行紧凑）", () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByText("项目"));
    // 预设「清洗」→ 指令回填
    fireEvent.click(screen.getByText("清洗"));
    const input = screen.getByPlaceholderText(/输入指令/) as HTMLInputElement;
    expect(input.value).toContain("清洗");
  });

  it("选中单元格后经图表菜单生成柱状图并预览（P0-2，对标千问表格 Agent）", async () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByText("设备")); // 选中 A2
    // 图表动作收敛为下拉菜单：先展开「图表 ▾」，再选「柱状图 PNG」
    fireEvent.click(screen.getByRole("button", { name: /图表/ }));
    fireEvent.click(screen.getByRole("menuitem", { name: /柱状图 PNG/ }));
    // mock XlsxChart 返回 3 个数据点 → notice 显示
    expect(await screen.findByText(/已生成图表.*3 个数据点/)).toBeTruthy();
    // 产物入预览队列（mock 返回 .gaea/exports/xxx-chart-mock.png）
    expect(usePreviewStore.getState().previewFile).toContain("chart-mock.png");
  });

  it("图表菜单含折线/饼图/嵌入 Word/嵌入 PPT 入口（mock 不抛错）", async () => {
    render(<XlsxPreview body={body} fileName="预算表.xlsx" relPath="mock.xlsx" />);
    fireEvent.click(screen.getByRole("button", { name: /图表/ }));
    // 菜单内 5 个动作齐全
    expect(screen.getByRole("menuitem", { name: /折线图 PNG/ })).toBeTruthy();
    expect(screen.getByRole("menuitem", { name: /饼图 PNG/ })).toBeTruthy();
    expect(screen.getByRole("menuitem", { name: /图表→Word/ })).toBeTruthy();
    expect(screen.getByRole("menuitem", { name: /图表→PPT/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("menuitem", { name: /折线图 PNG/ }));
    expect(await screen.findByText(/已生成图表/)).toBeTruthy();
  });
});
