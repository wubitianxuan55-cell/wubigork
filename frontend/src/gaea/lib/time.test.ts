import { describe, it, expect } from "vitest";
import { relativeTime, startOfDay } from "./time";

const now = new Date(2026, 7, 13, 12, 0, 0).getTime(); // 2026-08-13 12:00

describe("relativeTime", () => {
  it("小于 1 分钟显示刚刚", () => {
    expect(relativeTime(now - 30_000, now)).toBe("刚刚");
  });

  it("分钟前", () => {
    expect(relativeTime(now - 5 * 60_000, now)).toBe("5 分钟前");
  });

  it("小时前（同一天）", () => {
    expect(relativeTime(now - 3 * 3_600_000, now)).toBe("3 小时前");
  });

  it("昨天", () => {
    expect(relativeTime(now - 24 * 3_600_000, now)).toBe("昨天");
  });

  it("更早显示 M-D", () => {
    const tenDaysAgo = new Date(2026, 7, 3, 12, 0, 0).getTime();
    expect(relativeTime(tenDaysAgo, now)).toBe("8-3");
  });
});

describe("startOfDay", () => {
  it("归零时分秒毫秒", () => {
    expect(startOfDay(new Date(2026, 7, 13, 23, 59, 59, 999))).toBe(new Date(2026, 7, 13).getTime());
  });
});
