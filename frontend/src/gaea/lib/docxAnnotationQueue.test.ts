import { describe, expect, it, vi } from "vitest";
import {
  FAIL_EMPTY_REPLACEMENT,
  SKIP_NOT_LOCATED,
  addToQueue,
  applyReplacementToText,
  clearDocxQueue,
  createQueueItem,
  locateExcerpt,
  nextQueueItemId,
  normalizeExcerpt,
  queueStats,
  removeFromQueue,
  resetItemForRetry,
  runQueue,
  runnableItems,
  transitionStatus,
  updateQueueItem,
  type DocxQueueItem,
  type DocxQueueItemStatus,
  type QueueRunDeps,
} from "./docxAnnotationQueue";

function itemOf(excerpt: string, instruction: string, status: DocxQueueItemStatus = "pending"): DocxQueueItem {
  return { ...createQueueItem(excerpt, instruction, nextQueueItemId()), status };
}

// ── 状态机：增删改查 / 去重 / 迁移 ────────────────────────────────

describe("docxAnnotationQueue 状态机", () => {
  it("addToQueue 新增条目并归一化摘录；队列原数组不变", () => {
    const q0: DocxQueueItem[] = [];
    const r = addToQueue(q0, "  第一段  摘录内容 ", "润色此段");
    expect(r.merged).toBe(false);
    expect(r.queue).toHaveLength(1);
    expect(r.queue[0].norm).toBe("第一段 摘录内容");
    expect(r.queue[0].status).toBe("pending");
    expect(r.queue[0].error).toBeUndefined();
    expect(q0).toHaveLength(0); // 原数组不被原地修改
  });

  it("去重：归一化后同摘录同指令合并进既有条目（空白差异不算新条目）", () => {
    const { queue } = addToQueue([], "预算合计\n一百二十万", "精简\n这段文字");
    const r = addToQueue(queue, "预算合计  一百二十万", "精简 这段文字");
    expect(r.merged).toBe(true);
    expect(r.queue).toHaveLength(1);
    expect(r.item.id).toBe(queue[0].id);
  });

  it("同摘录不同指令（或反之）不去重，各自成条", () => {
    const { queue } = addToQueue([], "同一段摘录", "润色");
    const r2 = addToQueue(queue, "同一段摘录", "翻译成中文");
    const r3 = addToQueue(r2.queue, "另一段摘录", "润色");
    expect(r2.merged).toBe(false);
    expect(r3.merged).toBe(false);
    expect(r3.queue).toHaveLength(3);
  });

  it("空摘录或空指令（归一化后）拒绝入队", () => {
    expect(() => addToQueue([], "   ", "润色")).toThrow();
    expect(() => addToQueue([], "摘录", "  ")).toThrow();
  });

  it("removeFromQueue / clearDocxQueue / updateQueueItem", () => {
    const { queue } = addToQueue([], "甲", "指令一");
    const { queue: q2 } = addToQueue(queue, "乙", "指令二");
    expect(removeFromQueue(q2, q2[0].id).map((q) => q.excerpt)).toEqual(["乙"]);
    expect(clearDocxQueue()).toEqual([]);
    const patched = updateQueueItem(q2, q2[1].id, { status: "done" });
    expect(patched[1].status).toBe("done");
    expect(patched[0].status).toBe("pending");
  });

  it("状态迁移白名单：pending→running→done/failed/skipped；failed/skipped 可重试；done 终态", () => {
    const p = itemOf("摘录", "指令", "pending");
    expect(transitionStatus(p, "running").status).toBe("running");
    const r = transitionStatus(transitionStatus(p, "running"), "failed");
    expect(transitionStatus(r, "pending").status).toBe("pending");
    const s = transitionStatus(transitionStatus(p, "running"), "skipped");
    expect(transitionStatus(s, "running").status).toBe("running");
    const d = transitionStatus(transitionStatus(p, "running"), "done");
    expect(() => transitionStatus(d, "pending")).toThrow(/非法的队列状态迁移/);
    expect(() => transitionStatus(d, "running")).toThrow();
    expect(() => transitionStatus(p, "done")).toThrow(); // pending 不能跳到 done
  });

  it("resetItemForRetry 仅对 failed/skipped 生效并清错误与歧义标记", () => {
    const { queue } = addToQueue([], "摘录", "指令");
    const failed = updateQueueItem(queue, queue[0].id, { status: "failed", error: "boom", ambiguous: true });
    const retried = resetItemForRetry(failed, queue[0].id);
    expect(retried[0].status).toBe("pending");
    expect(retried[0].error).toBeUndefined();
    expect(retried[0].ambiguous).toBe(false);
    const done = updateQueueItem(failed, queue[0].id, { status: "done" });
    expect(() => resetItemForRetry(done, queue[0].id)).toThrow(/不可重试/);
  });

  it("runnableItems 只含 pending/failed/skipped；queueStats 分状态计数", () => {
    const queue: DocxQueueItem[] = [
      itemOf("一", "a", "pending"),
      itemOf("二", "b", "running"),
      itemOf("三", "c", "done"),
      itemOf("四", "d", "failed"),
      itemOf("五", "e", "skipped"),
    ];
    expect(runnableItems(queue).map((q) => q.excerpt)).toEqual(["一", "四", "五"]);
    expect(queueStats(queue)).toEqual({ total: 5, pending: 1, running: 1, done: 1, failed: 1, skipped: 1 });
  });
});

// ── 归一化与定位 ────────────────────────────────────────────────

describe("normalizeExcerpt", () => {
  it("换行/制表/连续空格/全角空格收敛为单空格并去首尾", () => {
    expect(normalizeExcerpt("  第一段\n第二段\t\t第三段   末尾  ")).toBe("第一段 第二段 第三段 末尾");
    expect(normalizeExcerpt("全角\u3000空格\u00a0混排")).toBe("全角 空格 混排");
    expect(normalizeExcerpt(" \n\t ")).toBe("");
  });
});

describe("locateExcerpt 定位", () => {
  const fullText = "项目背景：为提升交付效率。预算合计一百二十万元。项目背景另有重复。";

  it("唯一命中：返回原文精确坐标且 unique=true", () => {
    const loc = locateExcerpt(fullText, "预算合计一百二十万元");
    expect(loc).not.toBeNull();
    expect(loc!.unique).toBe(true);
    expect(loc!.occurrences).toBe(1);
    expect(loc!.start).toBe(fullText.indexOf("预算合计"));
    expect(loc!.end).toBe(loc!.start + "预算合计一百二十万元".length);
    expect(applyReplacementToText(fullText, loc!, "预算 95 万元")).toBe(
      "项目背景：为提升交付效率。预算 95 万元。项目背景另有重复。",
    );
  });

  it("找不到：返回 null（不定位、绝不猜位）", () => {
    expect(locateExcerpt(fullText, "文中不存在的句子")).toBeNull();
    expect(locateExcerpt(fullText, "")).toBeNull();
    expect(locateExcerpt("", "预算")).toBeNull();
  });

  it("非唯一命中：取第一处但 unique=false 如实上报次数", () => {
    const doc = "重复标记在前。目标句子在中间。重复标记在后，目标句子又出现。";
    const loc = locateExcerpt(doc, "目标句子");
    expect(loc).not.toBeNull();
    expect(loc!.unique).toBe(false);
    expect(loc!.occurrences).toBe(2);
    expect(loc!.start).toBe(doc.indexOf("目标句子")); // 第一处
  });

  it("摘录与全文空白形态不同（换行/多空格）仍能命中，坐标映射回原文", () => {
    const doc = "第一段开头。\n  跨行  的摘录内容，收尾。\n后续段落。";
    const loc = locateExcerpt(doc, "跨行\n的摘录内容，收尾。");
    expect(loc).not.toBeNull();
    expect(loc!.unique).toBe(true);
    // 原文坐标从「跨」字起（跳过前导空白），到「。」收
    expect(doc.slice(loc!.start, loc!.end)).toMatch(/^跨行/);
    expect(doc.slice(loc!.start, loc!.end)).toMatch(/收尾。$/);
  });

  it("摘录比全文还长 / 全文只有前缀：不命中返回 null", () => {
    expect(locateExcerpt("短文", "这是一段远远超过全文长度的摘录文本")).toBeNull();
  });
});

// ── 批量执行编排 ────────────────────────────────────────────────

function makeDeps(over: Partial<QueueRunDeps> = {}): QueueRunDeps & {
  updates: Array<{ id: string; patch: Record<string, unknown> }>;
} {
  const updates: Array<{ id: string; patch: Record<string, unknown> }> = [];
  const deps: QueueRunDeps = {
    generate: vi.fn(async (excerpt: string) => `【改】${excerpt}`),
    apply: vi.fn(async () => {}),
    readText: vi.fn(async () => ""),
    onUpdate: (id, patch) => updates.push({ id, patch: patch as Record<string, unknown> }),
    ...over,
  };
  return { ...deps, updates };
}

describe("runQueue 批量执行编排", () => {
  it("串行逐条执行：状态经 running 落到 done，汇总计数正确", async () => {
    const doc = "第一处待改内容，第二处待改内容在后。";
    const { queue } = addToQueue([], "第一处待改内容", "润色");
    const withTwo = addToQueue(queue, "第二处待改内容", "精简").queue;
    const deps = makeDeps({ readText: async () => doc });
    const summary = await runQueue(withTwo, deps);
    expect(summary).toEqual({ total: 2, done: 2, failed: 0, skipped: 0 });
    // 串行：每条先 running 再 done，第一条完成前第二条不出发
    const first = deps.updates.filter((u) => u.id === withTwo[0].id);
    const second = deps.updates.filter((u) => u.id === withTwo[1].id);
    expect(first.map((u) => u.patch.status)).toEqual(["running", "done"]);
    expect(deps.updates.findIndex((u) => u.id === withTwo[1].id)).toBeGreaterThan(
      deps.updates.findIndex((u) => u.id === withTwo[0].id && u.patch.status === "done"),
    );
    expect(second.map((u) => u.patch.status)).toEqual(["running", "done"]);
    expect(deps.generate).toHaveBeenCalledWith("第一处待改内容", "润色");
    expect(deps.apply).toHaveBeenNthCalledWith(2, "第二处待改内容", "【改】第二处待改内容");
  });

  it("条目间重新定位：前一条替换后全文位移，后一条以刷新后的文本定位成功", async () => {
    // 第一条替换后文本变长（位移 +N），第二条的摘录在位移之后的位置。
    let current = "甲段。乙段目标句子。丙段。";
    const deps = makeDeps({
      generate: async (_excerpt, instruction) =>
        instruction === "改甲" ? "甲段落改写为一段长了很多很多很多的全新文字" : "乙段句子已按要求改好",
      apply: async (excerpt, replacement) => {
        const loc = locateExcerpt(current, excerpt);
        expect(loc).not.toBeNull();
        current = applyReplacementToText(current, loc!, replacement);
      },
      readText: vi.fn(async () => current), // 每次读最新全文（写回后位移可见）
    });
    const { queue } = addToQueue([], "甲段", "改甲");
    const q2 = addToQueue(queue, "乙段目标句子", "改乙").queue;
    const summary = await runQueue(q2, deps);
    expect(summary).toEqual({ total: 2, done: 2, failed: 0, skipped: 0 });
    expect(current).toBe("甲段落改写为一段长了很多很多很多的全新文字。乙段句子已按要求改好。丙段。");
    // readText 在每条执行前都会被调用（首条一次 + 每次写回后刷新）
    expect(deps.readText).toHaveBeenCalledTimes(2);
  });

  it("再定位失败（前一条替换覆盖了后一条摘录）→ skipped，绝不错位替换", async () => {
    let current = "前半段旧文字。将被吞噬的摘录。";
    const deps = makeDeps({
      generate: vi.fn(async () => "前半段加后句的整体重写文案"),
      apply: async (excerpt, replacement) => {
        const loc = locateExcerpt(current, excerpt);
        if (!loc) throw new Error(`定位失败：${excerpt}`);
        current = applyReplacementToText(current, loc!, replacement);
      },
      readText: async () => current,
    });
    const { queue } = addToQueue([], "前半段旧文字。将被吞噬的摘录。", "合并改写");
    const q2 = addToQueue(queue, "将被吞噬的摘录", "改后句").queue;
    const summary = await runQueue(q2, deps);
    // 第二条的摘录已被第一条的重写覆盖：诚实 skipped，不静默错位替换
    expect(summary).toEqual({ total: 2, done: 1, failed: 0, skipped: 1 });
    const item2 = q2[1];
    const skipPatch = deps.updates.find((u) => u.id === item2.id && u.patch.status === "skipped");
    expect(skipPatch).toBeTruthy();
    expect(skipPatch!.patch.error).toBe(SKIP_NOT_LOCATED);
    expect(current).not.toContain("改后句"); // 从未对第二条发起过生成
    expect(deps.generate).toHaveBeenCalledTimes(1);
  });

  it("单条生成/写回失败 → failed 携带错误信息，继续下一条", async () => {
    const doc = "第一条内容。第二条内容。第三条内容。";
    let nth = 0;
    const deps = makeDeps({
      generate: vi.fn(async () => {
        nth += 1;
        if (nth === 2) throw new Error("AI 服务超时");
        return `【改】第 ${nth} 条`;
      }),
      readText: async () => doc,
    });
    const { queue } = addToQueue([], "第一条内容", "a");
    const q2 = addToQueue(queue, "第二条内容", "b").queue;
    const q3 = addToQueue(q2, "第三条内容", "c").queue;
    const summary = await runQueue(q3, deps);
    expect(summary).toEqual({ total: 3, done: 2, failed: 1, skipped: 0 });
    const failPatch = deps.updates.find((u) => u.id === q3[1].id && u.patch.status === "failed");
    expect(failPatch!.patch.error).toBe("AI 服务超时");
    // 失败不中断：第三条照常 done
    expect(deps.updates.find((u) => u.id === q3[2].id && u.patch.status === "done")).toBeTruthy();
    // done 条目无错误残留
    expect(deps.updates.find((u) => u.id === q3[0].id && u.patch.status === "done")!.patch.error).toBeUndefined();
  });

  it("AI 返回空替换 → failed（哨兵 empty-replacement），不算 done", async () => {
    const deps = makeDeps({ generate: async () => "  ", readText: async () => "唯一内容。" });
    const { queue } = addToQueue([], "唯一内容", "润色");
    const summary = await runQueue(queue, deps);
    expect(summary).toEqual({ total: 1, done: 0, failed: 1, skipped: 0 });
    expect(deps.updates[0].patch.status).toBe("running");
    expect(deps.updates[1].patch.error).toBe(FAIL_EMPTY_REPLACEMENT);
    expect(deps.apply).not.toHaveBeenCalled();
  });

  it("非唯一命中：照常执行第一处，ambiguous=true 如实暴露", async () => {
    const deps = makeDeps({ readText: async () => "重复文本在前，重复文本在后。" });
    const { queue } = addToQueue([], "重复文本", "统一改写");
    const summary = await runQueue(queue, deps);
    expect(summary).toEqual({ total: 1, done: 1, failed: 0, skipped: 0 });
    const donePatch = deps.updates.find((u) => u.patch.status === "done")!;
    expect(donePatch.patch.ambiguous).toBe(true);
    expect(deps.apply).toHaveBeenCalledWith("重复文本", "【改】重复文本");
  });

  it("readText 失败 → failed（环境故障与摘录跳过区分），下一条重试读取", async () => {
    let calls = 0;
    const deps = makeDeps({
      readText: vi.fn(async () => {
        calls += 1;
        if (calls === 1) throw new Error("文档提取失败");
        return "恢复正常。";
      }),
    });
    const { queue } = addToQueue([], "恢复正常", "a");
    const q2 = addToQueue(queue, "永不出现", "b").queue;
    const summary = await runQueue(q2, deps);
    expect(summary).toEqual({ total: 2, done: 0, failed: 1, skipped: 1 });
    expect(deps.updates.find((u) => u.id === q2[0].id && u.patch.status === "failed")!.patch.error).toBe("文档提取失败");
    expect(deps.updates.find((u) => u.id === q2[1].id && u.patch.status === "skipped")!.patch.error).toBe(SKIP_NOT_LOCATED);
  });

  it("已 done 的条目不重跑：执行全部只处理 pending/failed/skipped", async () => {
    const queue: DocxQueueItem[] = [
      { ...itemOf("已完成", "a"), status: "done" },
      itemOf("待执行", "b", "pending"),
      { ...itemOf("曾失败", "c"), status: "failed", error: "boom" },
    ];
    const deps = makeDeps({ readText: async () => "已完成待执行曾失败" });
    const summary = await runQueue(queue, deps);
    expect(summary).toEqual({ total: 2, done: 2, failed: 0, skipped: 0 });
    expect(deps.generate).not.toHaveBeenCalledWith("已完成", "a");
  });
});
