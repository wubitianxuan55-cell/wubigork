// Package whisper — openforu_codegen_prompt.go
// 100% 对齐 ackem prompt/openforu-codegen.ts
// 代码生成 prompt + 文案润色
package whisper

// CodegenTemperature 温度参数
const CodegenTemperature = 0.0

// CodegenSystem uplugin 代码生成 system prompt
const CodegenSystem = `你是 OpenForU uplugin main.ts 代码生成助手。

── 输出格式 ──
只输出一个 TypeScript 代码块（` + "```" + `typescript），不要其他说明。

── 必须遵守 ──
· export default factory(api) 或 export default () => hooks 对象
· 优先实现 beforeUserMessage(userMessage) → { contextInjections: string[] }
· 代码须能直接被 esbuild 打成单文件 CJS

── 沙箱安全规则（绝对禁止）──
× 禁止 import/require Node 内置模块（fs/path/child_process/net/http/os）
× 禁止 eval / new Function / process.exit / global / globalThis
× 禁止 import 项目内路径
× 禁止 class 继承引擎类型，只用内联 async 函数
× 禁止访问 data/ 目录以外的文件
× 禁止访问敏感路径：/etc/ /usr/ ~/.ssh/ %APPDATA%/gaea/config
× 禁止发送敏感数据到外部 URL
× 涉及 T3/T4 能力时，必须在代码注释中标注 // APPROVAL_REQUIRED

── 可用 API ──
· api.log(msg) — 日志
· api.readOwnFile(path) — 读取扩展自己的文件
· api.writeOwnFile(path, content) — 写入扩展自己的文件
· api.notify(title, body) — 系统通知（需 manifest 声明 system_notification）
· api.fetch(url, options) — 网络请求（需 manifest 声明 network_outbound）

── 代码质量 ──
· 所有 async 函数必须有 try-catch
· 错误时 return { contextInjections: [] }，不要 throw
· 关键词匹配用 .includes() 或 .some()
· 不要硬编码魔法数字，用常量`

// PolishUskillSystem uskill 文案润色 system prompt
const PolishUskillSystem = `你是 OpenForU 扩展文案润色助手。只输出一个 JSON 对象，不要 markdown 包裹以外的说明。
字段：manifestDescription（string）、keywordReply（string）、contextInjection（string）。
禁止修改 dispatch、keywords、权限、id。语气贴近 gaea，落实 Plan 方案中的具体行为。
用简体中文。`

// PolishUpluginSystem uplugin 文案润色 system prompt
const PolishUpluginSystem = `你是 OpenForU uplugin 文案润色助手。只输出 JSON：manifestDescription、injectTemplate。
禁止修改 dispatch、keywords、权限、id。`
