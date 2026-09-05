// tool_icons.ts — 工具名→图标映射表，ToolCard.tsx 专用。
// 从 ToolCard.tsx 提取，减少主文件行数。
// 工具分组从编程分类改为办公分类：文档/计算/规范/项目/图表/通讯
import {
  BookOpen, Brain, Calculator, CheckCircle,
  FilePen, FileText, FolderOpen, Globe, Hourglass,
  Layers, List, ListTree, PlusCircle, Search, Sparkles,
  Table, Trash2, Users, Wrench, Zap, type Icon,
} from "../icons";

export const ICONS: Record<string, Icon> = {
  // 文档
  edit_file: FilePen, multi_edit: FilePen, write_file: FilePen, read_file: FileText,
  delete_range: Trash2, delete_symbol: Trash2, notebook_edit: FilePen,
  // 计算
  bash: Calculator, bash_output: Calculator, kill_shell: Calculator,
  // 规范
  ls: FolderOpen, glob: Search, grep: Search, check: CheckCircle,
  // 项目
  task: ListTree, run_skill: Zap, install_skill: PlusCircle, slash_command: Zap,
  // 图表
  stats: Table, chart: Table,
  chart_gen: Table, "chart-builder": Table,
  // 办公
  format_convert: FileText, "format-convert": FileText, "doc-assemble": Layers,
  // 通讯
  web_fetch: Globe, web_search: Globe, message: Users, chat: Users,
  // 通用
  memory_search: Brain, remember: Brain, memory_get: Brain, read_skill: BookOpen,
  knowledge_search: Search, knowledge_add: PlusCircle, promote_session_facts: Sparkles,
  genui_validate: CheckCircle,
  wait: Hourglass, complete_step: CheckCircle, ask: List, brainstorm: Sparkles,
};

export function mcpOr(name: string): Icon {
  return name.startsWith("mcp__") ? Wrench : Wrench;
}
