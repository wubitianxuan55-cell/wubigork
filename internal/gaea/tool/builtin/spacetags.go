// spacetags.go 是双空间装配（S1.3-B）的工具空间标签表：工具名 → work/play/shared。
// 过滤只发生在装配期（boot.addBuiltins / boot ExtraTools / MCP spec 层），
// 运行时 executeOne 行为不变——注册表是 per-Build 实例，子代理 ActiveSchemas
// 继承父注册表，装配期物理过滤天然保持对齐（设计 docs/gaea-space-assembly-design.md §0.2/§2）。
//
// 归类原则（以实际注册工具名核对）：
//   - work：全部办公/编辑/检索系（S0.6 edit 系、shell、文件、检索、文档/图表、
//     成本库/知识库/事实库、专业办公工具）；
//   - play：生图/轻语/小说/角色域工具（image_gen 等，多为桌面端 ExtraTools）；
//   - shared：通用编排/记忆/任务工具（ask/complete_step/todo_write/memory_search、
//     task/run_skill 等元工具——它们不经本表过滤路径注册，凡注册即两空间通用）；
//   - MCP 工具缺省 shared（动态名不进静态表；spec 层过滤为 no-op，热插拔走同一规则）。
package builtin

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// SpaceShared 是"两空间通用"标签值（非空间标识；work/play 复用 spaces 常量）。
const SpaceShared = "shared"

// spaceTags 是静态分类表（显式列出全部已归类工具，未列出的名字缺省 shared）。
// 键为模型可见工具名（MCP 工具为 "mcp__<server>__<tool>" 动态名，不入表）。
var spaceTags = map[string]string{
	// ── work：办公/编辑/检索系 ──────────────────────────────────
	"bash":             spaces.SpaceWork,
	"bash_output":      spaces.SpaceWork,
	"kill_shell":       spaces.SpaceWork,
	"wait":             spaces.SpaceWork,
	"read_file":        spaces.SpaceWork,
	"write_file":       spaces.SpaceWork,
	"edit_file":        spaces.SpaceWork,
	"edit_lines":       spaces.SpaceWork,
	"multi_edit":       spaces.SpaceWork,
	"move_file":        spaces.SpaceWork,
	"ls":               spaces.SpaceWork,
	"grep":             spaces.SpaceWork,
	"web_fetch":        spaces.SpaceWork,
	"web_search":       spaces.SpaceWork,
	"format_convert":   spaces.SpaceWork,
	"chart_gen":        spaces.SpaceWork,
	"diagram_gen":      spaces.SpaceWork,
	"diagram":          spaces.SpaceWork,
	"screen_capture":   spaces.SpaceWork,
	"vision":           spaces.SpaceWork,
	"ocr":              spaces.SpaceWork,
	"cost_search":      spaces.SpaceWork,
	"cost_save":        spaces.SpaceWork,
	"cost_indicators":  spaces.SpaceWork,
	"knowledge_add":    spaces.SpaceWork,
	"knowledge_search": spaces.SpaceWork,
	"semantic_search":  spaces.SpaceWork,
	"routine_llm":      spaces.SpaceWork,
	"translate_text":   spaces.SpaceWork,
	"fact_add":         spaces.SpaceWork,
	"fact_list":        spaces.SpaceWork,
	"fact_clear":       spaces.SpaceWork,
	// ── play：生图/轻语/小说/角色域 ──────────────────────────────
	"image_gen": spaces.SpacePlay,
	// ── shared：通用编排/记忆/任务（显式入表便于核对，缺省同为 shared）──
	"ask":            SpaceShared,
	"complete_step":  SpaceShared,
	"todo_write":     SpaceShared,
	"memory_search":  SpaceShared,
	"memory_get":     SpaceShared,
	"remember":       SpaceShared,
	"forget":         SpaceShared,
	"task":           SpaceShared,
	"run_skill":      SpaceShared,
	"read_skill":     SpaceShared,
	"install_skill":  SpaceShared,
	"slash_command":  SpaceShared,
	"summarize_file": SpaceShared,
}

// ToolSpace 返回工具名对应的空间标签："work" | "play" | "shared"。
// 不在表内的名字（含 MCP 动态名）缺省 shared（两空间通用，fail-open）。
func ToolSpace(name string) string {
	if tag, ok := spaceTags[name]; ok {
		return tag
	}
	return SpaceShared
}

// SpaceTagOf 返回工具的生效空间标签：SpaceTaggedTool 自声明优先（仿
// PersistWriteTool 模式），否则按名字查表，缺省 shared。
func SpaceTagOf(t tool.Tool) string {
	if tag := strings.TrimSpace(tool.SpaceTagOf(t)); tag != "" {
		return tag
	}
	return ToolSpace(t.Name())
}

// AllowsSpace 报告工具 t 是否应在 space 装配中注册。space 取装配空间
// （"work"/"play"）；""（space.mode=off 的整体回退形态）或其他值一律全注册
// （现状逐字节回退）。shared 两空间都注册；work/play 只在对应空间注册。
func AllowsSpace(t tool.Tool, space string) bool {
	if !spaces.Valid(space) {
		return true // mode=off / 异常值：不分区，现状全注册
	}
	tag := SpaceTagOf(t)
	if tag == spaces.SpaceWork || tag == spaces.SpacePlay {
		return tag == space
	}
	return true // shared
}
