// tool_icons.ts — 工具名→图标映射表，ToolCard.tsx 专用。
// 从 ToolCard.tsx 提取，减少主文件行数。
// 工具分组从编程分类改为办公分类：文档/计算/规范/项目/图表/通讯
import {
  BookOpen, Brain, Calculator, CheckCircle,
  FilePen, FileText, FolderOpen, Globe, Hourglass,
  Layers, List, ListTree, PlusCircle, Search, Sparkles,
  Table, Trash2, Users, Wrench, Zap, type LucideIcon,
} from "lucide-react";

export const ICONS: Record<string, LucideIcon> = {
  // 文档
  edit_file: FilePen, multi_edit: FilePen, write_file: FilePen, read_file: FileText,
  delete_range: Trash2, delete_symbol: Trash2, notebook_edit: FilePen,
  // 计算
  bash: Calculator, bash_output: Calculator, kill_shell: Calculator,
  // 规范
  ls: FolderOpen, glob: Search, grep: Search, check: CheckCircle,
  // 项目
  task: ListTree, run_skill: Zap, parallel_skills: Layers, install_skill: PlusCircle,
  // 图表
  stats: Table, chart: Table,
  // 通讯
  web_fetch: Globe, web_search: Globe, message: Users, chat: Users,
  // 通用
  memory_search: Brain, remember: Brain, read_skill: BookOpen,
  wait: Hourglass, complete_step: CheckCircle, ask: List, brainstorm: Sparkles,
};

export function mcpOr(name: string): LucideIcon {
  return name.startsWith("mcp__") ? Wrench : Wrench;
}
