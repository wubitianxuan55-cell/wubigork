import { describe, expect, it } from "vitest";
import {
  dropSending,
  firstPending,
  insertAt,
  isQueueBusy,
  makeQueueItem,
  markSending,
  pendingItems,
  removeAt,
  revertSending,
} from "./composerQueue";

describe("composerQueue 纯函数", () => {
  it("makeQueueItem 生成 pending，id 递增不重复", () => {
    const a = makeQueueItem("甲");
    const b = makeQueueItem("乙");
    expect(a.status).toBe("pending");
    expect(b.status).toBe("pending");
    expect(a.id).not.toBe(b.id);
    expect(a.text).toBe("甲");
  });

  it("markSending / isQueueBusy / firstPending / pendingItems", () => {
    const a = makeQueueItem("一");
    const b = makeQueueItem("二");
    expect(isQueueBusy([a, b])).toBe(false);
    expect(firstPending([a, b])?.id).toBe(a.id);
    const next = markSending([a, b], a.id);
    expect(isQueueBusy(next)).toBe(true);
    expect(next[0].status).toBe("sending");
    expect(firstPending(next)?.id).toBe(b.id);
    expect(pendingItems(next).map((q) => q.text)).toEqual(["二"]);
  });

  it("revertSending 只把指定 sending 收回 pending；dropSending 丢掉发送中", () => {
    const a = makeQueueItem("一");
    const b = makeQueueItem("二");
    const sending = markSending([a, b], a.id);
    const reverted = revertSending(sending, a.id);
    expect(reverted[0].status).toBe("pending");
    expect(dropSending(sending).map((q) => q.text)).toEqual(["二"]);
    expect(revertSending(sending, b.id)[0].status).toBe("sending");
  });

  it("removeAt / insertAt 保序", () => {
    const items = [makeQueueItem("一"), makeQueueItem("二"), makeQueueItem("三")];
    expect(removeAt(items, 1).map((q) => q.text)).toEqual(["一", "三"]);
    expect(removeAt(items, 9)).toHaveLength(3);
    const extra = makeQueueItem("插");
    expect(insertAt(items, 1, extra).map((q) => q.text)).toEqual(["一", "插", "二", "三"]);
    expect(insertAt(items, 99, extra).at(-1)?.text).toBe("插");
  });
});
