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
    vi.restoreAllMocks();
  });

  it("Clipboard API 成功：返回 true，不走兜底", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    stubExecCommand(() => true);
    await expect(copyToClipboard("文本")).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith("文本");
    expect(document.execCommand).not.toHaveBeenCalled();
  });

  it("API 拒绝（失焦/权限）：execCommand 兜底成功 → true", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("denied")));
    stubExecCommand(() => true);
    await expect(copyToClipboard("文本")).resolves.toBe(true);
    expect(document.execCommand).toHaveBeenCalledWith("copy");
  });

  it("Clipboard API 缺失（旧环境）：直接走兜底", async () => {
    stubClipboard(undefined);
    stubExecCommand(() => true);
    await expect(copyToClipboard("文本")).resolves.toBe(true);
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
    await expect(copyToClipboard("整段内容")).resolves.toBe(true);
    expect(taValue).toBe("整段内容");
    // 兜底收尾即摘除临时 textarea，不留 DOM 残渣
    expect(document.querySelector("textarea")).toBeNull();
  });
});
