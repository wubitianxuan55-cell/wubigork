// lingxiIcons.tsx — 灵犀卡片图标（SVG 路径逐字取自 re-lingxi/out/ui/showcase.html
// 效果图基准，viewBox 16 / stroke currentColor，与灵犀渲染层视觉一致）。
// 用途：办公对话流卡片头图标（batch3 UI「卡片图标换成灵犀的」）。
import type { SVGProps } from "react";

function base(props: SVGProps<SVGSVGElement>) {
  return {
    width: 16,
    height: 16,
    viewBox: "0 0 16 16",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.2,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
    ...props,
  };
}

/** 思考卡时钟（showcase 思考卡头图标） */
export function LxThought(props: SVGProps<SVGSVGElement>) {
  return (
    <svg {...base(props)}>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6l2.4 1.6" />
    </svg>
  );
}

/** 终端/命令执行（showcase 工具卡·代码执行） */
export function LxTerminal(props: SVGProps<SVGSVGElement>) {
  return (
    <svg {...base(props)}>
      <rect x="2.2" y="2.8" width="11.6" height="10.4" rx="2" />
      <path d="M4.6 6.4 6.6 8l-2 1.6M8.4 10h3" />
    </svg>
  );
}

/** 编辑/改写（灵犀铅笔同款圆+斜笔形） */
export function LxEdit(props: SVGProps<SVGSVGElement>) {
  return (
    <svg {...base(props)}>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M6.5 6.5 10.5 5.5 9.5 9.5 5.5 10.5z" />
    </svg>
  );
}

/** 读取/文档（showcase 画布头文档图标） */
export function LxDoc(props: SVGProps<SVGSVGElement>) {
  return (
    <svg {...base(props)}>
      <rect x="2.5" y="1.8" width="11" height="12.4" rx="1.6" />
      <path d="M5.2 5h5.6M5.2 8h5.6M5.2 11h3.2" />
    </svg>
  );
}

/** 网页获取 */
export function LxGlobe(props: SVGProps<SVGSVGElement>) {
  return (
    <svg {...base(props)}>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M1.8 8h12.4M8 1.8c2.6 2.2 2.6 10.2 0 12.4M8 1.8c-2.6 2.2-2.6 10.2 0 12.4" />
    </svg>
  );
}

/** 通用工具（未映射工具的兜底：灵犀爪形要点标） */
export function LxGeneric(props: SVGProps<SVGSVGElement>) {
  return (
    <svg {...base(props)}>
      <rect x="2.2" y="2.8" width="11.6" height="10.4" rx="2" />
      <path d="M5.5 8h5M8 5.5v5" />
    </svg>
  );
}

/** 按工具名映射灵犀图标（子串匹配；未命中=generic 兜底） */
export function lxToolIcon(name: string): (props: SVGProps<SVGSVGElement>) => JSX.Element {
  const n = name.toLowerCase();
  if (/bash|exec|command|run_command|jupyter|shell|terminal/.test(n)) return LxTerminal;
  if (/edit|write|replace|patch|multiedit|apply|insert/.test(n)) return LxEdit;
  if (/read|view|doc|spill|preview/.test(n)) return LxDoc;
  if (/web|search|fetch|url|grep|glob|ls\b|find/.test(n)) return LxGlobe;
  if (/office_/.test(n)) return LxEdit;
  return LxGeneric;
}
