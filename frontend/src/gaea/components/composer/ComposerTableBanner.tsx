// Composer 拆分产物：粘贴表格即数据提示条（行为零变化，T6-10.1）
import { Table } from "../../icons";

export interface ComposerTableBannerProps {
  rows: number
  cols: number
  tableMode: boolean
  onTableModeChange: (v: boolean) => void
}

export function ComposerTableBanner({ rows, cols, tableMode, onTableModeChange }: ComposerTableBannerProps) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border-soft/60 bg-accent/5 text-[11px]">
      <Table size={12} className="text-accent shrink-0" />
      <span className="text-fg-dim">
        已识别表格数据：{rows} 行 × {cols} 列
      </span>
      <label className="flex items-center gap-1 ml-auto cursor-pointer select-none text-fg-faint hover:text-fg transition-colors">
        <input
          type="checkbox"
          checked={tableMode}
          onChange={(e) => onTableModeChange(e.target.checked)}
          className="accent-accent"
        />
        发送时转为 Markdown 表格
      </label>
    </div>
  )
}
