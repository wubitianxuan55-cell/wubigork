// copyToClipboard 单测（v4.513 诚实反馈轮）：四条路径——API 成功 / API
// 失败走兜底 / API 缺失走兜底 / 双路全败。execCommand 在 jsdom 未实现，
// 显式 stub 并在 afterEach 恢复，防污染同 worker 的其它套件。
import { afterEach, describe, expect, it, vi } from "vitest";
import { copyToClipboard } from "./clipboard";

const stubClipboard = (writeText?: (t: string) => Promise<void>) => {
  Object.defineProperty(navigator, "clipboard", {
    value: writeText ? { writeText } : undefined,
    configurable: true,
  });
};
const stubExecCommand = (impl: () => boolean) => {
  Object.defineProperty(document, "execCommand", {
    value: vi.fn(impl),
    configurable: true,
  });
};

describe("copyToClipboard（v4.513 单源）", () => {
  afterEach(() => {
    // 恢复 jsdom 原状（clipboard undefined / execCommand 未实现）
    Object.defineProperty(navigator, "clipboard", { value: undefined, configurable: true });
    delete (document as Partial<Document>).execCommand;
    delete (globalThis as { ClipboardItem?: unknown }).ClipboardItem;
    vi.restoreAllMocks();
  });

  it("Clipboard API 成功：返回 true，不走兜底", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    stubExecCommand(() => true);
    await expect(copyToClipboard("文本")).resolves.toBe("plain");
    expect(writeText).toHaveBeenCalledWith("文本");
    expect(document.execCommand).not.toHaveBeenCalled();
  });

  it("API 拒绝（失焦/权限）：execCommand 兜底成功 → true", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("denied")));
    stubExecCommand(() => true);
    await expect(copyToClipboard("文本")).resolves.toBe("plain");
    expect(document.execCommand).toHaveBeenCalledWith("copy");
  });

  it("Clipboard API 缺失（旧环境）：直接走兜底", async () => {
    stubClipboard(undefined);
    stubExecCommand(() => true);
    await expect(copyToClipboard("文本")).resolves.toBe("plain");
    expect(document.execCommand).toHaveBeenCalledWith("copy");
  });

  it("双路全败：返回 false（调用方不得显示「已复制」）", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("denied")));
    stubExecCommand(() => false);
    await expect(copyToClipboard("文本")).resolves.toBe(false);
  });

  it("兜底路径把全文写入临时 textarea（内容经 execCommand 复制且即挂即摘）", async () => {
    stubClipboard(undefined);
    let taValue = "";
    stubExecCommand(() => {
      const ta = document.querySelector("textarea");
      taValue = ta ? (ta as HTMLTextAreaElement).value : "";
      return true;
    });
    await expect(copyToClipboard("整段内容")).resolves.toBe("plain");
    expect(taValue).toBe("整段内容");
    // 兜底收尾即摘除临时 textarea，不留 DOM 残渣
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("rich copy writes text/plain + text/html through ClipboardItem", async () => {
    class ClipboardItemStub {
      constructor(public data: Record<string, Blob>) {}
    }
    Object.defineProperty(globalThis, "ClipboardItem", {
      value: ClipboardItemStub,
      configurable: true,
      writable: true,
    });
    const write = vi.fn().mockResolvedValue(undefined);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { write, writeText },
      configurable: true,
    });

    await expect(copyToClipboard("纯文本", { html: "<table><tr><td>x</td></tr></table>" }))
      .resolves.toBe("rich");
    expect(writeText).not.toHaveBeenCalled();
    const item = write.mock.calls[0][0][0] as ClipboardItemStub;
    expect(Object.keys(item.data)).toEqual(["text/plain", "text/html"]);
  });

  it("empty content does not modify the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    await expect(copyToClipboard("")).resolves.toBe(false);
    expect(writeText).not.toHaveBeenCalled();
  });

  it("rich write rejection falls back to plain text", async () => {
    class ClipboardItemStub {
      constructor(public data: Record<string, Blob>) {}
    }
    Object.defineProperty(globalThis, "ClipboardItem", {
      value: ClipboardItemStub,
      configurable: true,
      writable: true,
    });
    const write = vi.fn().mockRejectedValue(new Error("rich denied"));
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { write, writeText },
      configurable: true,
    });
    await expect(copyToClipboard("文本", { html: "<b>x</b>" })).resolves.toBe("plain");
    expect(writeText).toHaveBeenCalledWith("文本");
  });

  it("execCommand throw still removes the temporary textarea", async () => {
    stubClipboard(undefined);
    stubExecCommand(() => { throw new Error("copy threw"); });
    await expect(copyToClipboard("文本")).resolves.toBe(false);
    expect(document.querySelector("textarea")).toBeNull();
  });
});
