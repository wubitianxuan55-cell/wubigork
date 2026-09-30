package config

// config_paths.go — 配置路径与默认提示词族（P5 破防专项拆自 config.go：
// 用户配置路径/归档与会话目录/约定目录扫描/默认系统提示词与语言政策——
// 搬移零功能变更）。

import (
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

const DefaultSystemPrompt = `你是 gaea（盖亚）——用户的通用办公 AI 助手，也是日常 AI 伙伴。你沉稳、清晰、可靠，温和而不说教。
坦诚：你是 AI，不冒充人类、不编造出身；但你认真对待每一次对话，把用户的事放在心上。
可靠：先理解再动手，条理清楚；答应的事会做到，做不到会直说。
有温度：说话自然亲切，不甜腻、不客套、不空夸；该提醒的风险会提醒，该追问的需求会追问。
知之为知之：不知道就明说，然后主动去查、去验证，绝不编造。
沟通：使用简洁的中文，结论先行；需要时用列表、表格让信息一目了然；赞美要具体、基于事实。
所有思考和输出必须使用中文。

你负责协助用户完成日常办公工作：文档撰写与编辑、格式转换、表格与数据处理、
图表制作、演示文稿、资料检索、方案与报告编写、任务跟踪等。
使用提供的工具读取和写入文件以及运行 shell 命令。

**原则：**
- 理解请求后再行动；用工具验证而非猜测；保持变更最小且正确；完成后简要总结。
- 执行后必验证：每完成一个步骤，在 complete_step 之前先用工具验证结果正确性（检查生成文件、核对数据、确认输出格式）。
- 遇到用户真正需要决策的问题时（方案选择、范围、影响重大的判断），使用 ask 工具列出 2-4 个具体选项，不要猜测或把问题埋在文字里。有明确默认值时直接选择，不要为了确认而提问。
- 多步骤任务使用 todo_write 跟踪进度：列出步骤，始终保持恰好一个 in_progress，每完成一步就标记为 completed。随时更新列表，不要等到最后。
- 所有独立操作必须在一个响应中完成：并行读取多个文件、编辑不同文件、运行 shell 命令。只有顺序操作（编辑+验证同一文件、任务子代理）才分开发送。工具系统支持非冲突工具的并行执行——积极利用。
- 输出风格强调结构化文档、表格和计算过程，保持清晰可追溯。

**办公规范：**
- 长文档/表格/PDF 的创建与编辑交给已安装的 docx / xlsx / pdf 技能（run_skill 调用），agent 不自造文档格式
- 不同来源的文档统一用 format_convert 转为可编辑 Markdown 后再处理；表格数据也可用 bash + python（openpyxl/pandas）提取
- 图表用 chart_gen 生成（bar/line/pie/scatter）
- 报告先列结构大纲，再逐步填充；多份文档拼装用 run_skill 调 doc-assemble 技能
- 需要最新资料时用 web_search / web_fetch 检索并注明来源

**本地工具：**
- vision：识别图片内容（布局/对象/图表含义）。本地视觉模型，通常几秒，冷启动（模型未加载）约 20 秒+
- ocr：提取图片/扫描件中的文字。本地 OCR 服务，通常 2-5 秒/页，冷启动可能更久
- semantic_search：在成本库/工程知识库/办公记忆中按语义检索；scope=file 时检索工作区已索引文件。本地向量检索，通常 1-3 秒
- format_convert：docx/xlsx/pptx/pdf → Markdown（扫描件走 OCR 回退）。按文档大小数秒到数十秒
- chart_gen：统计图表（bar/line/pie/scatter）；diagram：流程图/时序图/甘特图等 Mermaid 图
- screen_capture：截取屏幕；image_gen：生成图片
- routine_llm：通用文本处理（摘要、归一化、抽取、改写等），目标模型在模型中心「常规办公」绑定，默认本地，可绑定免费云端模型
以上工具均在本地/免费运行，不消耗主模型 token。是否使用、何时使用，由你自行判断。

**子代理：**
子代理入口只有两级：task（派发临时自包含任务）与 run_skill（按名调用技能）。
以下场景优先派发子代理：
- 需把 docx/xlsx/pdf 转成 Markdown：run_skill 调 format-convert 技能
- 需从数据生成统计图表：run_skill 调 chart-builder 技能
- 需把多份文档拼装成完整报告：run_skill 调 doc-assemble 技能
子代理在独立上下文中运行——它看不到你的对话，任务消息必须自包含；其工具调用
不会撑大你的上下文。犹豫时直接派发。内置子代理技能见下方 Skills 索引。

**记忆：**
用 remember/forget 跨会话持久化事实。**只在用户明确要求记住时才调用 remember**（remember 会弹用户确认卡），不要主动记录：
- 用户明确说「记住这个」「以后都按这个口径」等：用 remember 保存
- 用户纠正偏好或事实并要求记住：用 remember 保存，避免后续重复犯错
- 记忆被证明错误：用 forget 删除
不要记录瞬时状态、用户明确要求不保存的内容、或未经用户确认的信息。记忆是持久的——只保存跨会话不变的事实。`

// LanguagePolicy is the forced language directive appended to the system prompt.
// Always Chinese — the user is a native Chinese speaker and cannot read English.
// Static text, so it stays part of the cache-stable prefix.
const LanguagePolicy = `所有思考过程和输出必须使用中文。不要使用英文——用户看不懂英文。` +
	`代码标识符（变量名、函数名、API 路由、数据库字段名）保持英文，但注释、` +
	`解释、分析、回复全部使用中文。即使收到的消息是英文，也始终用中文回复。`

func userConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "config.toml")
}

// UserConfigPath is the user-global config file (~/.config/gaea/config.toml),
// or "" when the user config dir can't be resolved.
func UserConfigPath() string { return userConfigPath() }

// ArchiveDir is where compacted conversation history is archived for
// traceability (one timestamped .jsonl per compaction). Empty if the user config
// directory cannot be resolved, in which case archiving is skipped.
func ArchiveDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "archive")
}

// SessionDir is where chat sessions are persisted (one .jsonl per session).
// Used by `gaea chat --continue` / `--resume` to find the recent ones. Empty
// if the user config dir can't be resolved — sessions then aren't saved.
func SessionDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea", "sessions")
}

// WorkspaceSessionDir returns the workspace-scoped session directory.
// Sessions are isolated per workspace so switching projects shows only that
// workspace's history.
//
// S2 双空间：space 为 "work"/"play" 时返回分区目录 <cwd>/.gaea/sessions/<space>/；
// space 为 ""（space.mode=off 的回退形态）返回平铺目录 <cwd>/.gaea/sessions/
// （旧行为）。旧平铺会话恒按 work 兼容可读（读端各空间目录 + 平铺兜底）。
func WorkspaceSessionDir(cwd, space string) string {
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			return SessionDir()
		}
	}
	base := filepath.Join(cwd, ".gaea", "sessions")
	switch space {
	case spaces.SpaceWork:
		return filepath.Join(base, spaces.SpaceWork)
	case spaces.SpacePlay:
		return filepath.Join(base, spaces.SpacePlay)
	default:
		return base // "" = 平铺（space.mode=off 回退形态）
	}
}

// MemoryUserDir returns the gaea user config root (…/gaea), under which
// the user-global TIANXUAN.md and the per-project auto-memory store live. Empty
// when the user config dir can't be resolved, which disables user-scoped memory.
func MemoryUserDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gaea")
}

// ConventionDirs are the parent directories scanned for agent assets (skills,
// commands), in canonical-first order. .gaea is ours; .agents / .agent /
// .claude let users drop in assets authored for other agent tools without moving
// files. Shared so skills (internal/skill) and commands (CommandDirs) discover
// the same set. Note: hooks are NOT scanned across these — a .claude/settings.json
// uses a different hook schema that can't be parsed as ours, so hooks stay in
// .gaea/settings.json (see internal/hook).
var ConventionDirs = []string{".gaea", ".agents", ".agent", ".claude"}

// conventionSubdirsAsc joins sub under each ConventionDir of base, in ascending
// priority (reverse of ConventionDirs) so the canonical .gaea ends up the
// highest-priority entry — command.Load lets a later directory win on a clash.
func conventionSubdirsAsc(base, sub string) []string {
	out := make([]string, 0, len(ConventionDirs))
	for i := len(ConventionDirs) - 1; i >= 0; i-- {
		out = append(out, filepath.Join(base, ConventionDirs[i], sub))
	}
	return out
}

// CommandDirsAt returns the directories scanned for custom slash commands,
// lowest priority first, so a later (more specific) directory overrides an
// earlier one on a name clash. Order: home-dir convention dirs
// (~/.claude/commands … ~/.gaea/commands), the legacy XDG user dir
// (~/.config/gaea/commands), then the project's convention dirs under cwd
// (.claude/commands … .gaea/commands). Scanning the .claude / .agents / .agent
// dirs lets commands authored for other agent tools (same .md + frontmatter
// format) work here unchanged.
func CommandDirsAt(cwd string) []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, conventionSubdirsAsc(home, "commands")...)
	}
	if dir, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(dir, "gaea", "commands")) // legacy XDG user dir
	}
	dirs = append(dirs, conventionSubdirsAsc(cwd, "commands")...)
	return dirs
}

// CommandDirs 是基于进程工作目录的命令扫描目录（兼容 CLI 场景）。
func CommandDirs() []string {
	return CommandDirsAt(".")
}

// SourcePath returns the highest-priority config file that exists, or "" if none.
func SourcePath() string {
	if _, err := os.Stat("gaea.toml"); err == nil {
		return "gaea.toml"
	}
	if uc := userConfigPath(); uc != "" {
		if _, err := os.Stat(uc); err == nil {
			return uc
		}
	}
	return ""
}
