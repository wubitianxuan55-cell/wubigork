import { describe, expect, it } from "vitest";
import { repairGenuiSpec, repairSingleComponent } from "./guard";
import { GENUI_LIMITS } from "./spec";

describe("guard.repairGenuiSpec", () => {
  it("修复合法规格：保留已知节点并 clamp 数字", () => {
    const spec = repairGenuiSpec({
      title: "看板",
      items: [
        { type: "stat", label: "营收", value: "¥128k", delta: "+12%" },
        { type: "progress", label: "完成", value: 150, valueLabel: "150%" },
        { type: "button", label: "刷新", action: "refresh" },
      ],
    });
    expect(spec).not.toBeNull();
    expect(spec?.title).toBe("看板");
    expect(spec?.items).toHaveLength(3);
    const progress = spec?.items[1];
    if (progress?.type !== "progress") throw new Error("progress expected");
    expect(progress.value).toBe(100);
  });

  it("丢弃未知 type 与缺必填字段的节点", () => {
    const spec = repairGenuiSpec({
      items: [
        { type: "evil", html: "<script>" },
        { type: "text" },
        { type: "stat", label: "ok", value: "1" },
      ],
    });
    expect(spec).not.toBeNull();
    expect(spec?.items).toHaveLength(1);
    expect(spec?.items[0]).toMatchObject({ type: "stat" });
  });

  it("根规格不是对象/无 items → null；单组件 root 自动包装", () => {
    expect(repairGenuiSpec("oops")).toBeNull();
    expect(repairGenuiSpec({ items: [] })).toBeNull();
    const wrapped = repairGenuiSpec({ type: "callout", content: "hi", tone: "info" });
    expect(wrapped?.items).toHaveLength(1);
    expect(wrapped?.items[0]).toMatchObject({ type: "callout" });
  });

  it("节点预算：超 200 节点时截断", () => {
    const items = Array.from({ length: 210 }, (_, i) => ({
      type: "text" as const,
      content: `n${i}`,
    }));
    const spec = repairGenuiSpec({ items });
    expect(spec?.items).toHaveLength(GENUI_LIMITS.maxNodes);
  });

  it("字符串按 cap 截断、颜色只收 hex/单词色", () => {
    const spec = repairGenuiSpec({
      items: [
        {
          type: "text",
          content: "x".repeat(3000),
        },
        {
          type: "avatar",
          name: "盖亚",
          color: "url(javascript:alert(1))",
        },
        { type: "avatar", name: "盖亚", color: "#2dd4bf" }, // hex-exempt 单测输入色值
      ],
    });
    expect(spec?.items[0]).toMatchObject({ type: "text" });
    const text = spec?.items[0];
    if (text?.type !== "text") throw new Error("text expected");
    expect(text.content).toHaveLength(GENUI_LIMITS.maxString);
    expect(spec?.items[1]).toMatchObject({ type: "avatar" });
    const bad = spec?.items[1];
    if (bad?.type !== "avatar") throw new Error("avatar expected");
    expect(bad.color).toBeUndefined();
    const good = spec?.items[2];
    if (good?.type !== "avatar") throw new Error("avatar expected");
    expect(good.color).toBe("#2dd4bf"); // hex-exempt 单测输入色值
  });

  it("表格行/列超限截断；quiz 选项不足丢弃", () => {
    const table = repairGenuiSpec({
      items: [
        {
          type: "table",
          columns: ["a", "b"],
          rows: Array.from({ length: 60 }, () => ["1", "2"]),
        },
        { type: "quiz", question: "q", options: [{ label: "only" }] },
      ],
    });
    const t = table?.items[0];
    if (t?.type !== "table") throw new Error("table expected");
    expect(t.rows).toHaveLength(GENUI_LIMITS.maxTableRows);
    expect(table?.items).toHaveLength(1);
  });
});

describe("guard.repairSingleComponent", () => {
  it("修复单个组件", () => {
    expect(repairSingleComponent({ type: "divider" })).toMatchObject({ type: "divider" });
    expect(repairSingleComponent({ type: "unknown" })).toBeNull();
  });
});

describe("guard.单组件根兜底与字段方言（2026-11-11 聊天板块 GenUI 不可用根修）", () => {
  // 形状取自真实聊天史（plain 模式本地模型输出，字段名做了脱敏替换）：
  // 模型把单个组件对象直接当围栏根，且 items 条目用 {label,content} 方言。

  it("list 根 + {label,content} 方言条目 → title 进 banner、条目归一为 {title,desc} 记录", () => {
    const spec = repairGenuiSpec({
      type: "list",
      title: "四个理解维度",
      items: [
        { label: "无常", content: "梦醒即逝。" },
        { label: "惜时" },
        { content: "只有正文的条目" },
        "普通字符串条目",
      ],
    });
    expect(spec).not.toBeNull();
    expect(spec?.title).toBe("四个理解维度");
    expect(spec?.items).toHaveLength(1);
    const list = spec?.items[0];
    if (list?.type !== "list") throw new Error("list expected");
    expect(list.items).toHaveLength(4);
    // 上游 SKILL 归一口径：label/name→title，description/content/text/body/
    // detail→desc；只有正文时正文用作 title。
    expect(list.items[0]).toEqual({ title: "无常", desc: "梦醒即逝。" });
    expect(list.items[1]).toEqual({ title: "惜时" });
    expect(list.items[2]).toEqual({ title: "只有正文的条目" });
    expect(list.items[3]).toBe("普通字符串条目");
  });

  it("timeline 根 + {time,label,content} 方言条目 → label/content/date 认领", () => {
    const spec = repairGenuiSpec({
      type: "timeline",
      title: "工作日日程",
      items: [
        { time: "07:00", label: "晨间唤醒", content: "起床、洗漱。" },
        { date: "12月", label: "上映", content: "第三部。" },
        { time: "08:00", title: "正主优先", desc: "正主描述", label: "方言标签", content: "方言描述" },
      ],
    });
    expect(spec).not.toBeNull();
    expect(spec?.title).toBe("工作日日程");
    const timeline = spec?.items[0];
    if (timeline?.type !== "timeline") throw new Error("timeline expected");
    expect(timeline.items[0]).toEqual({ title: "晨间唤醒", desc: "起床、洗漱。", time: "07:00" });
    expect(timeline.items[1]).toEqual({ title: "上映", desc: "第三部。", time: "12月" });
    // 正主字段存在时方言不覆盖。
    expect(timeline.items[2]).toEqual({ title: "正主优先", desc: "正主描述", time: "08:00" });
  });

  it("steps 根 + {label,content} 方言条目 → title/desc 认领", () => {
    const spec = repairGenuiSpec({
      type: "steps",
      items: [
        { label: "第一步", content: "准备。" },
        { title: "第二步", desc: "执行。" },
      ],
    });
    const steps = spec?.items[0];
    if (steps?.type !== "steps") throw new Error("steps expected");
    expect(steps.steps[0]).toEqual({ title: "第一步", desc: "准备。" });
    expect(steps.steps[1]).toEqual({ title: "第二步", desc: "执行。" });
  });

  it("上游文档化别名：card.body/card.content→items、grid.columns→cols", () => {
    const card = repairGenuiSpec({
      type: "card",
      title: "卡",
      body: [{ type: "text", content: "正文" }],
    });
    const cardNode = card?.items[0];
    if (cardNode?.type !== "card") throw new Error("card expected");
    expect(cardNode.items).toHaveLength(1);

    const grid = repairGenuiSpec({
      type: "grid",
      columns: 2,
      items: [{ type: "divider" }],
    });
    const gridNode = grid?.items[0];
    if (gridNode?.type !== "grid") throw new Error("grid expected");
    expect(gridNode.cols).toBe(2);
  });

  it("card 根不双挂 banner（card 自渲染标题）", () => {
    // items 条目用方言（无 type）逼迫走单组件根路径：spec.title 不重复挂，
    // 标题留在 card 节点自身（卡体方言条目不在本刀范围，照白名单口径丢弃）。
    const spec = repairGenuiSpec({
      type: "card",
      title: "卡片标题",
      items: [{ label: "方言条目", content: "正文" }],
    });
    expect(spec?.title).toBeUndefined();
    expect(spec?.items).toHaveLength(1);
    expect(spec?.items[0]).toMatchObject({ type: "card", title: "卡片标题" });
  });

  it("items 全空且根无白名单 type → 仍 null（走降级面）", () => {
    expect(repairGenuiSpec({ title: "无名", items: [{ label: "x", content: "y" }] })).toBeNull();
    expect(repairGenuiSpec({ items: [] })).toBeNull();
  });
});

