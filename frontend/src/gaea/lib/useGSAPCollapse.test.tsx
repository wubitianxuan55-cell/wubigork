// useGSAPCollapse — rAF 冻结兜底回归（2026-11-20 实弹）：窗口隐藏/失焦态的
// open 翻转必须直接落终值；遮挡返回、焦点回收与 watchdog 都不能留下半开卡。
import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useRef, useState } from "react";
import gsap from "gsap";
import { useGSAPCollapse } from "./useGSAPCollapse";

function Harness({ initial = false }: { initial?: boolean }) {
  const [open, setOpen] = useState(initial);
  const ref = useRef<HTMLDivElement>(null);
  useGSAPCollapse(ref, open);
  return (
    <div>
      <div ref={ref} style={{ overflow: "hidden" }}>
        <p style={{ height: 100 }}>内容</p>
      </div>
      <button onClick={() => setOpen((v) => !v)}>toggle</button>
    </div>
  );
}

function setHidden(hidden: boolean) {
  Object.defineProperty(document, "hidden", { configurable: true, get: () => hidden });
}

const fireVisibility = (hidden: boolean) => {
  setHidden(hidden);
  document.dispatchEvent(new Event("visibilitychange"));
};

describe("useGSAPCollapse rAF 冻结兜底", () => {
  afterEach(() => {
    setHidden(false);
    vi.useRealTimers();
    vi.restoreAllMocks();
    cleanup();
  });

  it("隐藏态翻转：高度直接落终值（auto↔0px，无 tween 冻帧）", () => {
    setHidden(true);
    const { container } = render(<Harness />);
    const wrap = container.firstElementChild!.firstElementChild as HTMLElement;
    // 挂载初始 open=false → 0px；隐藏态翻 true → 直接 auto；再翻 false → 直接 0px
    expect(wrap.style.height).toBe("0px");
    fireEvent.click(container.querySelector("button")!);
    expect(wrap.style.height).toBe("auto");
    fireEvent.click(container.querySelector("button")!);
    expect(wrap.style.height).toBe("0px");
  });

  it("visibilitychange→hidden：容器高度吸附到目标态（半开残留清零）", () => {
    const { container } = render(<Harness initial={false} />);
    const wrap = container.firstElementChild!.firstElementChild as HTMLElement;
    expect(wrap.style.height).toBe("0px");
    // 人为制造「冻在中间高度」（实弹 112~380px 半开卡），再触发隐藏事件 → 吸附回 0px
    wrap.style.height = "237px";
    fireVisibility(true);
    expect(wrap.style.height).toBe("0px");
  });

  it("打开态隐藏吸附：高度落 auto", () => {
    const { container } = render(<Harness initial={true} />);
    const wrap = container.firstElementChild!.firstElementChild as HTMLElement;
    expect(wrap.style.height).toBe("auto");
    wrap.style.height = "150px"; // 冻结中间态
    fireVisibility(true);
    expect(wrap.style.height).toBe("auto");
  });

  it("WebView 仅失焦时翻转：不等待 rAF，直接落终值", () => {
    vi.spyOn(document, "hasFocus").mockReturnValue(false);
    const { container } = render(<Harness />);
    const wrap = container.firstElementChild!.firstElementChild as HTMLElement;
    const toggle = container.querySelector("button")!;
    expect(wrap.style.height).toBe("0px");
    fireEvent.click(toggle);
    expect(wrap.style.height).toBe("auto");
    fireEvent.click(toggle);
    expect(wrap.style.height).toBe("0px");
  });

  it("返回办公窗口 focus：在途半开 tween 吸附到关闭终值", () => {
    vi.spyOn(document, "hasFocus").mockReturnValue(true);
    const { container } = render(<Harness initial={true} />);
    const wrap = container.firstElementChild!.firstElementChild as HTMLElement;
    fireEvent.click(container.querySelector("button")!);
    wrap.style.height = "173px"; // rAF 冻结中间态
    fireEvent(window, new Event("focus"));
    expect(wrap.style.height).toBe("0px");
  });

  it("tween 永不推进：watchdog 在动画预算后强制落到关闭终值", () => {
    vi.useFakeTimers();
    vi.spyOn(document, "hasFocus").mockReturnValue(true);
    vi.spyOn(gsap, "fromTo").mockImplementation(
      () => ({}) as unknown as ReturnType<typeof gsap.fromTo>,
    );
    const { container } = render(<Harness initial={true} />);
    const wrap = container.firstElementChild!.firstElementChild as HTMLElement;
    fireEvent.click(container.querySelector("button")!);
    wrap.style.height = "211px";
    act(() => vi.advanceTimersByTime(400));
    expect(wrap.style.height).toBe("0px");
  });
});
