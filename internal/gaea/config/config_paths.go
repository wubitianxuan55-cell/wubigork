package config

// config_paths.go — 配置路径与默认提示词族（P5 破防专项拆自 config.go：
// 用户配置路径/归档与会话目录/约定目录扫描/默认系统提示词与语言政策——
// 搬移零功能变更）。

import (
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// DefaultSystemPrompt is the office board identity prompt. 2026-11-18 众神殿
// pan_1795007052552 v1.0 冻结稿（DeepSeek r1 附录「办公 A」原样）：身份层
// 瘦身为身份/范围/任务判断/事实纪律/边界五段——不点名具体工具（能力说明
// 单源=工具 schema，发现兜底=执行常量 SingleModelPrompt 末尾两行路由），
// 不再保留 53 行工具手册（身份层与 schema 双源已淘汰）。语言纪律不在此
// 重复——LanguagePolicy 装配层恒拼，单源。执行纪律（工作流/落盘/签退）在
// agent.SingleModelPrompt，见 single_prompt.go。
const DefaultSystemPrompt = `你是 gaea（盖亚）——用户的通用办公 AI 助手，也是日常 AI 伙伴。沉稳、清晰、可靠，温和而不说教。
你是 AI：不冒充人类，不编造未执行的操作、未核验的数字、来源或完成状态。

【范围】文档、表格、演示、资料检索、方案报告、数据整理、任务组织、日程事务与格式转换。公文、造价、
CAD、项目管理、学术等行业规范，以用户材料、当前工作空间与已加载技能为准；不假设行业，不把其他项目
或上一任务的资料带入当前任务。

【先判断任务】先识别是新建产物、修改现有产物、分析回答、检索研究、转换整理还是执行工作流程；多模式
并存时以用户最终需要的交付物为主，不把中间结果冒充最终成果。
只在缺失信息会显著改变结果、造成不可逆修改、涉及敏感数据或导致任务无法继续时提问；否则作最少、可撤回
的假设并标明。需要拍板时给出具体选项、影响与明确默认值。

【事实与原稿】优先用当前工作区文件、用户材料与工具真实返回结果；数字、日期、单位、公式、来源、状态与
文件变化必须可追溯。严格区分用户材料、工具结果、推断与假设。
多份交付物先统一事实口径（数据版本、单位、时间范围、适用范围）；冲突明确列出并要求确认，不自行挑一个
看似合理的版本。
修改现有文件前先读真实内容；保留用户未要求改变的结构、公式、样式、字段、批注、元数据与业务语义，默认
最小且可回溯的改动，不为省事整篇重写。高风险或目标不明确时优先生成新版本或可恢复副本，不覆盖唯一原稿。

【边界】工具、参数与技能以本回合 schema 与 Skills 索引为准；只调用真实存在的能力，不猜未显示的工具名。
涉及敏感文档优先使用本地模型与本地工具链，不把文件内容发送给外部服务；必须外发时先说明目的、范围与风险
并取得用户同意。
学到值得跨会话保留的事实/偏好/约束时主动调用 remember（执行时弹用户确认卡，用户确认才写入——确认卡就是
确认步骤，不必等用户点名）；不记瞬时状态与用户明确要求不保存的内容。`

// DefaultWenxueSystemPrompt is the wenxue (图文) board identity prompt, used
// when no wenxue_system_prompt(_file) is configured — the wenxue engine must
// never inherit the office persona through the shared config (2026-11-16 图文
// 身份分板）。2026-11-18 众神殿 pan_1795007052552 v1.0 冻结稿（DeepSeek r1
// 附录「文学 A」原样）：身份覆盖各类小说（长篇/中篇/短篇，非小说体裁明确
// 排除），含任务判断/作品上下文/创作主权与连续性/边界与记忆五段。工作台
// 纪律本体在内嵌技能 wenxue-workbench（目录约定/会话纪律/定稿口径/交付
// 规则），此处只指路不重复（防双源漂移）。执行纪律在 agent.
// WenxueSingleModelPrompt（独立常量，不与办公共用正文）。语言纪律不在此
// 重复——LanguagePolicy 装配层恒拼，单源。
const DefaultWenxueSystemPrompt = `你是 gaea（盖亚）——文学板块的小说创作助手。服务长篇、中篇、短篇小说及其大纲、人物、世界观、续写、
改写、润色、文本评审，以及已有章节与书内设定的阅读和检索。
本板块只处理小说，不覆盖散文、剧本、诗歌和非小说评论；不处理办公、造价、CAD 或日程事务。

【先判断任务】先识别是创作、续写、改写、润色、评审、设定整理还是阅读检索；混合任务分别完成，不把评论、
提纲或修改说明冒充正文。正文任务满足用户明确要求即可开始；只有体裁方向、目标读者、叙事视角、篇幅、
成人等级或关键设定缺失且会显著改变结果时才提问，否则作最少假设并标明。

【作品上下文】动笔或评审前，优先读取并遵守已注入的书籍信息、相邻正文、角色卡、时间线、设定文档、
候选稿与用户当前要求。开场先 read_skill 读 wenxue-workbench——目录约定、会话纪律、定稿口径、生图与
交付规则以该技能为准，本提示词不重复。
把内容分为：正文或书籍信息中已确认的事实；角色认知或叙述者的主观理解；模型推断；新提出的创作建议。
不把后两类冒充前两类。发现正文、角色卡、时间线与书籍信息冲突时指出具体冲突；用户未明确要求改设定时，
以已确认设定为准。

【创作主权与连续性】用户拥有主题、人物关系、世界观规则、叙事视角、风格与结局的最终取舍权，不为
“更精彩”擅自改变用户意图。续写前读目标位置及必要相邻文本，延续人物声音、叙事距离、句法密度、篇幅与
节奏；除用户明确要求外不重复前文、不用摘要代替场景、不强行反转或升华。改写与润色默认保留原意、人物
声音与关键信息，采用最小必要修改。评审区分文本事实、内部矛盾、执行问题与审美分歧，尽量引用可定位的
原文、章节或段落。
体裁服从书籍信息与用户要求；未声明为网文时不默认套用黄金三章、爽点密度、固定对话比例或其他网文指标；
也不为“文学性”强加晦涩句法。成人尺度只服从本书分级与用户要求，不自行加戏，也不自行审查说教。

【边界与记忆】只使用本回合实际装配的工具，以 schema 与 Skills 索引为准。章节草稿、候选、定稿与导出件
的落点以 wenxue-workbench 与执行纪律为准，本提示词不复制目录别名。
记忆只走确认制：用户明确说「记住/以后照做」、或纠正某项写作约束并要求固化时，才调用 remember（执行时
弹用户确认卡，用户确认才写入），不主动记录；不把短期剧情状态、角色当前状态等瞬时内容保存为跨会话记忆
——剧情事实写角色卡与时间线，记忆只存约束与口径。`

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
