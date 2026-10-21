// clipboard.ts — 复制到剪贴板单源（v4.513 诚实反馈轮）
//
// Why：此前四处「假已复制」（fire-and-forget / catch-noop 后无条件置成功
// 态——Clipboard API 被拒或窗口失焦时失败仍显示「已复制」），另有三份
// 手写 execCommand 兜底互相重复且不检查兜底结果（二次失败仍假成功）。
// 单一出口：剪贴板 API 优先，不可用/失败走 execCommand 兜底；返回是否
// 真正写入，调用方按返回值给诚实反馈（成功才翻转「已复制」，失败提示
// 可见）。兜底 textarea 用 fixed+opacity:0 防滚动跳变，select 后检查
// execCommand 布尔返回。

export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    // 落入 execCommand 兜底
  }
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(ta);
    return ok;
  } catch {
    return false;
  }
}
