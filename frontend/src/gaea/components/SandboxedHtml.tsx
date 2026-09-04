// 沙箱 HTML 渲染（蒸馏规划 1c，对标 dsh html viewer）：产物/报告类 .html
// 文件不再当纯文本展示，也不注入宿主 DOM——放独立 iframe 里跑。
//
// 双保险：
//  1. sandbox="allow-scripts"（刻意不给 allow-same-origin）→ 文档运行在
//     不透明源，无法触碰宿主 DOM/localStorage/cookie，也无法向上导航；
//  2. Chromium csp 属性（WebView2 支持）default-src 'none'：禁一切网络与
//     外链资源，只放内联样式/脚本与 data:/blob: 图——README 级独立报告
//     可看，外链探针/追踪脚本无网可用。
//
// 顶条如实标注沙箱语义（简化不改功能红线：新增能力，无既有行为删除）。
import { t } from "../lib/i18n";
import { Shield } from "../icons";

const SANDBOX_CSP =
  "default-src 'none'; img-src data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:; media-src data: blob:";

export function SandboxedHtml({ html, title }: { html: string; title: string }) {
  // i18n 说明：用非响应式 t() 而非 useT()——本组件被 FilePreview/
  // FilePreviewModal 挂载，其测试（jsdom）不包 LocaleProvider，useT 会抛错
  // （DocxPreview 同款先例）；非响应式取词结果与 useT() 一致。
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div
        data-testid="sandbox-html-note"
        className="flex shrink-0 items-center gap-1.5 border-b border-border-soft bg-bg-soft px-3 py-1.5 text-[10.5px] text-fg-faint"
      >
        <Shield size={12} className="text-accent" />
        <span>{t("preview.sandboxNote")}</span>
      </div>
      <iframe
        data-testid="sandboxed-html"
        title={title}
        sandbox="allow-scripts"
        {...({ csp: SANDBOX_CSP } as Record<string, string>)}
        referrerPolicy="no-referrer"
        srcDoc={html}
        className="min-h-0 w-full flex-1 border-0 bg-bg"
      />
    </div>
  );
}
