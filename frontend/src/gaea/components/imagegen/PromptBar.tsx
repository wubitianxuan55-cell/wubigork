// PromptBar.tsx — 轻量创作输入区（全部标签展示，全宽生成按钮）
import { Send, Loader2, Sparkles } from "lucide-react";

interface PromptBarProps {
  prompt: string;
  onPromptChange: (v: string) => void;
  generating: boolean;
  elapsed: number;
  onGenerate: () => void;
}

/** 快捷风格标签 */
const QUICK_TAGS = [
  "电影级光影", "8K超高清", "概念艺术", "史诗场景",
  "黑暗奇幻", "赛博朋克", "水墨风", "油画质感",
];

/** PromptBar — 轻量创作输入区 */
export function PromptBar({ prompt, onPromptChange, generating, elapsed, onGenerate }: PromptBarProps) {
  const handleTagClick = (tag: string) => {
    if (prompt.includes(tag)) return;
    const sep = prompt.trim() ? "，" : "";
    onPromptChange(prompt + sep + tag);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey && !generating) {
      e.preventDefault();
      onGenerate();
    }
  };

  return (
    <div className="absolute bottom-3 left-3 right-3 z-10">
      <div className="border border-border/60 rounded-2xl bg-bg-elev/90 backdrop-blur-xl shadow-2xl p-4 transition-shadow duration-300">
        {/* 标题 */}
        <div className="flex items-center gap-1.5 mb-2 text-fg-faint text-sm">
          <Sparkles className="w-3.5 h-3.5" />
          描述你心中的画面…
        </div>

        {/* TextArea — 三行，字号统一 */}
        <textarea
          placeholder="悬浮云端的仙侠城市，琉璃瓦宫殿，瀑布倾泻而下，霞光万丈…"
          value={prompt}
          onChange={(e) => onPromptChange(e.target.value)}
          rows={3}
          onKeyDown={handleKeyDown}
          className="dream-input text-sm leading-relaxed bg-bg-soft/50"
        />

        {/* 快捷标签 — 全部显示，无折叠 */}
        <div className="flex flex-wrap items-center gap-1.5 mt-2.5">
          {QUICK_TAGS.map((tag) => {
            const active = prompt.includes(tag);
            return (
              <button
                key={tag}
                onClick={() => handleTagClick(tag)}
                className={active ? "dream-tag dream-tag--active" : "dream-tag"}
              >
                {tag}
              </button>
            );
          })}
        </div>

        {/* 分隔线 */}
        <div className="h-px my-2 bg-border" />

        {/* 底部栏：字符计数左对齐 + 全宽生成按钮 */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 shrink-0">
            <span className="text-xs text-fg-faint tabular-nums">
              ⌨ {prompt.length} 字符
            </span>
            {generating && (
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-accent-soft text-accent text-xs font-medium">
                <Loader2 className="w-2.5 h-2.5 animate-spin" />
                {elapsed}s
              </span>
            )}
          </div>
          <button
            onClick={onGenerate}
            disabled={generating || !prompt.trim()}
            className="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-xl text-sm font-semibold text-accent-fg bg-accent shadow-md hover:shadow-lg transition-all disabled:opacity-40 disabled:cursor-not-allowed active:scale-[0.97] h-[38px]"
          >
            {generating ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                生成中 {elapsed}s
              </>
            ) : (
              <>
                <Send className="w-4 h-4" />
                生成图像
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
