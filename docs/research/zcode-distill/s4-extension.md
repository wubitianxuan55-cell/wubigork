# ZCode 扩展与知识注入模型调研（s4-extension）

> 调研对象：`clones/zcode`（只读检出）。
> 结论面向：gaea「办公板块」的复用判断。
> 纪律：只调研、不改代码；每条结论都给 `file:line` 证据；未验证的地方显式标注。

---

## 0. 摘要

ZCode 的扩展面由**五个可插拔能力域**构成，共同点是：**一律通过"声明 + capability + schema + 权限 + trace"进入 runtime，业务层不直接读盘/联网**（`apps/zcode-cli/docs/design/v2/skill.md:66-99` 的分层边界是这套写法最完整的样板）。

1. **Skills**：目录 + `SKILL.md` + frontmatter，**两层加载**（启动只注入 name/description/路径索引，正文由 `Skill` 工具按需拉取），索引有预算和降级，正文有字节上限和截断标注。
2. **子代理**：`agents/*.md`（名字/描述/模型/档位/工具白名单/权限模式/最大轮次/记忆域/mcpServers），可被项目、用户、插件三处声明；插件里的 agent 以 `<plugin>:<agent>` 命名空间注册。
3. **Plugins**：**插件 = 包**（manifest + skills + commands + agents + MCP + 可选 `ui.surfaces` 页面），带安装/启用/更新/卸载/孤立态的全生命周期；官方内置插件用 seed 缓存 + 不可变版本目录。
4. **MCP**：配置多源合并（plugin > project > user > CLI），**状态查询（快照）与主动探测（健康检查）分离**，project MCP 默认 untrusted 不自动连接。
5. **Hooks**：7 个生命周期事件、`process`/`command` 两种持久化类型、Claude 兼容 stdin、stdout 决策协议；project hook 走**声明 digest 粒度的精确信任 + 软门禁（pending 一律跳过，绝不劫持对话）**。

另有两块"知识注入"：**提示词分层**（system 稳定段 / system 动态段 / meta_user 上下文，各带 cacheHint）与**记忆**（一个事实一个 `.md` 文件 + `MEMORY.md` 短索引 + 后台抽取 + Dream 整理 + 双检索分支单一开关）。

对办公板块的判断见第 10、11 节：**能抄的是"声明式"和"门禁"两类机制**（子代理卡、插件组件投影、技能索引/正文分离、hook trust 粒度、MCP 状态口径）；**不该抄的是"生态"**（插件市场与安装链、UI Plugin 的 MCP Apps 宿主、把技能变成权限载体的那条路）。

---

## 1. 证据面与方法

- 源码：`clones/zcode/apps/zcode-cli/packages/{core,adapters,bootstrap,cli,contracts}`、`clones/zcode/packages/{shared,services}`。
- 规范：`apps/zcode-cli/docs/design/v2/*.md`（skill / plugin-compat / custom-commands / hooks / mcp / extension-discovery-priority）、`docs/specs/*`、`docs/runtime-tools/*`、`docs/usage/*`、`docs/superpowers/plans/*`。
- 官方插件源码：`apps/zcode-cli/packages/{documents,pdf,presentations,spreadsheets,office-runtime,visualize}-plugin`（这批就是 ZCode 的"办公能力"，证据价值最高）。
- gaea 侧落点：全部用 `grep` 确认过真实存在（见第 12 节列出的文件清单），没有凭印象写路径。

---

## 2. Skills

### 2.1 目录结构与发现

**机制**：一个技能 = 一个含 `SKILL.md` 的目录，或平铺的 `<name>.md`；根目录本身可含 `SKILL.md`（此时根即技能，同时再扫一层子目录），**只扫一层**。

- 证据：`clones/zcode/apps/zcode-cli/packages/adapters/src/skills/scan.ts:5-9`（"manifest 的 skills 项既可指向单个技能目录，也可指向一层技能集合"）、`:43-74`（先判根 `SKILL.md`，再 `readdir` 只遍历一层）、`:101-117`（候选必须是真实文件）。
- 错误契约：**只吞 ENOENT**，EACCES/EMFILE 必须上抛并转成 `skill_scan_failed` 诊断（`scan.ts:11-14`、`:144-155`、`adapters/src/skills/index.ts:144-157`）。
- 发现根与优先级（从高到低）：配置 `skills.roots` → 用户 `~/.zcode/skills` → 用户 `~/.agents/skills` → 项目 `<dir>/.zcode/skills`（从 cwd 逐级向上到 git worktree root）→ 项目 `.agents/skills` → 启用的插件根（`adapters/src/skills/roots.ts:21-72`、`:107-118`；规范 `docs/design/v2/extension-discovery-priority.md:17-28`）。
- 去重口径：**路径是安装身份**（`index.ts:80-85`），同名不同路径都可见；按 name 加载时**先发现者胜**（用户级遮蔽项目级，`extension-discovery-priority.md:26-28`）。

**办公场景价值**：中高。"一个技能一个目录 + 只扫一层"意味着技能包可以带 `references/`、`scripts/`、`assets/` 而不被误当技能；`.agents/skills` 兼容让同事按通用约定写的技能可直接用。

**gaea 落点**：`internal/gaea/skill/skill.go:157-179`（roots：项目 convention dirs → custom → home）、`:205-241`（`List` 按 **name** 去重、内置最后合并）、`:245-267`（`Read` 只在 name 命中时逐 root 找 `SKILL.md` 或 `<name>.md`）、`:273-306`（平铺 `.md` 与目录两种形态，跟随 symlink）。**缺口**：gaea 按 name 去重（后发现的同名直接丢），没有"路径身份 + 诊断重复"；也没有 worktree 逐级向上。

**成本**：S（gaea 已有 80%），补"重复名诊断 + 祖先目录扫描"约 1 天。

**风险**：向上扫祖先目录会读到用户没预期的技能（信任面扩大）；必须保留"到 repo 根为止"的硬边界（ZCode 用 `.git` marker 止步，`roots.ts:74-96`）。

### 2.2 frontmatter

**机制**：`SKILL.md` 开头 `---` 包住的平铺 YAML（**不是完整 YAML**，只解析顶层 `key: value`，支持 `>` / `|` 块标量）。

- 必填：`name`（缺失则回退目录名，仅在**完全没有 frontmatter** 时回退，`adapters/src/skills/index.ts:178-193`）、`description`（缺失是 error 并整条丢弃，`:194-204`）；长度上限 `1024`（`:20`、`:206-215`）。
- 可选：`when_to_use`（追加进描述预算，`:224`）、`license`、`metadata`；安全键白名单 `SAFE_FRONTMATTER_KEYS`（`:21-27`）。
- 解析细节：`parseFlatYaml` 跳过缩进行、支持块标量（`:300-377`），引号包裹去引号（`:404-415`）。
- `safeToAutoLoad` = frontmatter 只含安全键（`:232`），供权限判断使用；`policy.allowImplicitInvocation` 目前恒 true（`:234-236`）。
- 正文读取上限 `100_000` 字节，超出**从尾部截断**并标 `truncated`（`:28`、`:113-129`、`core/src/tool/handlers/skill.ts:18`）。

**规范里登记但未落地/P1**：`allowed-tools`、`model`、`effort`、`context: fork`、`disable-model-invocation`、`dependencies.tools`、`interface`（`docs/design/v2/skill.md:26`、`:134-145`、`:263-275`）。

**办公场景价值**：高。`description` 是**唯一进入模型视野的字段**，必须写成"何时用"而不是"是什么"——这是技能被正确触发的主要杠杆。

**gaea 落点**：`internal/gaea/skill/skill.go:311-337`（`name` 覆盖文件名、`description` 缺失只 warn、`allowed-tools` / `runas` / `context` / `agent` / `model` 五个字段）、`internal/gaea/frontmatter/frontmatter.go`。**缺口**：gaea 的 `description` 缺失只 warn（技能会进索引但显示占位符，`index.go:14`），而 ZCode 是 error 丢弃——两种都合理，但 gaea 应把它升级为**索引期诊断**，否则办公板块技能质量会静默劣化。

**成本**：S。
**风险**：gaea 的 frontmatter 是"平铺 key: value"，一旦办公技能要写多行 `description` 或 list，就会失真；ZCode 也只支持平铺，两边一致，暂不需要 YAML 库。

### 2.3 渐进式加载（索引 vs 全文）

**机制**（ZCode 的核心设计，`docs/design/v2/skill.md:59-64` 明确叫"两层加载"）：

1. **metadata discovery**：启动/初始化时把 `name / description / source / scope / path / policy` 注入上下文；
2. **content load**：模型显式调用 `Skill` 工具才读正文。

索引注入位与预算：

- 注入目标 `injectionTarget: "meta_user"`、`cacheHint: "dynamic"`（`core/src/context/sections/skills.ts:27-36`）——**不进 system prompt**，而是作为"上下文附件"塞在首条 user 消息里（`core/src/context/builder.ts:288-316`）。
- 默认预算 `20_000` 字符，单条描述截到 `250` 字符（`skills.ts:9-10`、`:60-69`）。
- 超预算**降级为纯名行**（`- name (file: path)`），不是丢掉尾部技能（`skills.ts:50-57`）。
- 每个条目带**绝对路径** `(file: /abs/.../SKILL.md)` 和"裸别名"提示 `(also loadable as <name>)`（`skills.ts:54-78`）。
- 只有 `Skill` 工具真的注册时才注入该段（`builder.ts:189-197`、`:235-238`）——避免告诉子代理一个它没有的工具。

正文加载：

- 输出包在 `<skill_content name="...">` 里，附 `Base directory for this skill: ...` 和相对路径说明（`core/src/tool/handlers/skill.ts:78-91`）。
- 变量替换只在这里做：`${CLAUDE_SKILL_DIR}` / `${ZCODE_SKILL_DIR}` → 技能目录（`skill.ts:93-96`）。
- 工具声明：`readOnly: true / destructive: false / concurrentSafe: true / riskLevel: low / needsApproval: false / sideEffectScope: session`（`skill.ts:120-144`），result budget `truncate`（`:145-153`），timeout 30s（`:154-158`）。
- `dynamic-workflows` 技能单独放宽到 `200_000` 字节（`skill.ts:25`、`:33-35`）。

**"已加载"的判定**：不另建会话级 Set，而是**扫 provider 可见历史**里是否有成功的 `Skill` 调用；compaction 把那次调用挤出去后，技能正文也不在了，于是门重新关上（`core/src/agent/loaded-skills.ts:1-46`）。这个判据非常值得抄。

**办公场景价值**：最高。办公技能正文往往很长（ZCode 官方 `docx/SKILL.md` 19.8KB、`pdf/SKILL.md` 72.8KB、`pptx/SKILL.md` 54KB，见 `packages/*-plugin/skills/*/SKILL.md` 文件大小），全量常驻会吃掉整个上下文预算。

**gaea 落点**：
- 索引：`internal/gaea/skill/index.go:12`（`IndexMaxChars = 4000`）、`:40-84`（超预算时**二分压缩描述宽度**，比 ZCode 的"降级纯名行"更平滑）、`:97-126`（渲染 `- name [🧬 subagent] — desc`）。
- 正文：`internal/gaea/skill/tools.go:67-101`（`run_skill` 分派 inline / subagent / pipeline）、`internal/gaea/tool/builtin/readskill.go`（`read_skill` 工具）、`internal/gaea/boot/sysprompt.go:132-147`（装配索引 + 注入 `read_skill` 解析器 + 计数回调）。
- **真缺口 1**：gaea 的索引**进 L1 缓存稳定前缀**（`boot/sysprompt.go:134` → `cache.Compiler`/`IdentityLayer`，`internal/gaea/cache/identity.go:14-42`），而 ZCode 把它放 meta_user 动态段。往 L1 塞意味着**加一个技能就整段前缀失效**；但放 meta_user 又会每轮重发。gaea 的四域（L1/L2/L3 profile/L4 spawn，`cache/compiler.go:1-11`）其实可以给技能索引单独开一个"低变动率"域。
- **真缺口 2**：gaea 目录布局技能会把 `references/*.md` **一次性拼进正文**（`skill.go:384-422`），等于取消了"渐进"的第二层——ZCode 的做法是正文只给 `routes/*.md`、`scenes/*.md` 的**路由表**，让模型自己按需 `Read`（`documents-plugin/skills/docx/SKILL.md:33-45`、`:70-94`）。
- **真缺口 3**：没有"已加载技能"判据；`main_test`/agent 侧靠工具结果留在历史里，compact 之后模型可能重复调用或误以为还在。

**成本**：M（缺口 2、3 各约 2-3 天；缺口 1 需要动 cache 域，属 L，但可以先用"技能索引改放 L2 且 Lock 前注入"折中）。
**风险**：把 references 改成按需 Read 会增加模型往返次数（办公场景常见"一次要 3 个 references"）；建议保留"小技能拼正文、大技能给路由表"的阈值（ZCode 的做法就是大技能才拆 routes/scenes）。

### 2.4 内置 skill 清单与分层

**ZCode 的三层**：

| 层 | 位置 | 可否卸载 | 证据 |
| --- | --- | --- | --- |
| bundled 技能包 | `packages/bundled-skills/skills/` | **不可**（不进市场、无开关、不能卸载） | `bootstrap/src/app/bundled-skills.ts:16-29`、`:87-95`（`scope: system, source: bundled, priority 1_000_000`） |
| 官方内置插件携带的 skill | `packages/*-plugin/skills/<name>/` | 可（随插件启停/卸载） | `bootstrap/src/app/official-plugin-definitions.ts:100-430` |
| 用户/项目 skill | `~/.agents/skills`、`<repo>/.zcode/skills` | 是文件，删了就没 | `adapters/src/skills/roots.ts:107-118` |

**bundled 的设计理由是一起事故**：`dynamic-workflows` 技能和 `/workflow` 命令原来住在用户可卸载的 `zcode-guide` 插件里，卸载后 `/workflow` 消失；修法是把技能挪进不可移除的 bundled 包（`bundled-skills.ts:19-29` 注释直接写了这次事故）。**"产品功能所依赖的技能必须与功能同生命周期"**——这条对办公板块极其重要。

另外 bundled 包有 **必需资产清单**，缺任何一个文件就**拒绝整包**并告警，而不是装出残缺技能（`bundled-skills.ts:36-41`、`:97-116`）；SEA 形态按内容 hash 解压到 `<storage>/bundled-skills/<hash>/`，目录名即内容身份，重复启动幂等，并发只有一个赢家（`:118-163`）。

官方内置插件清单（`official-plugin-definitions.ts:100-430`，逐个都在）：

- `visualize`（`0.1.0`，交互视图，带完整 assets/vendor 清单）
- `node-repl-host`（`0.6.1`，无 listing、不进市场，只带 `dist/mcp/server.js` 宿主产物）
- `android-emulator`（`0.1.0`）、`ios-simulator`（`0.1.0`）
- `browser-use`（`0.5.1`，带 2 个 skill + `scripts/browser-client.mjs`）
- **`documents`（0.2.0）/ `pdf`（0.2.1）/ `presentations`（0.2.0）/ `spreadsheets`（0.2.0）**：四个办公插件，各自 `componentTypes: ["ui"]`、`defaultEnabled: true`、必需资产 = `agents/visual-judge.md` + `skills/<fmt>/SKILL.md` + `skills/<fmt>-viewer/SKILL.md` + 查看器运行时三件（`:210-248`）
- `image-search`（`0.1.1`）、`restore-legacy-sessions`（`0.1.0`）
- `plugin-creator`（`0.1.1`）、`skill-creator`（`0.1.0`）、`zcode-guide`（`0.3.0`）
- `computer-use`（`0.6.5`，**默认关闭**，因为带 MCP + 系统 Helper）

`defaultEnabled` 的纪律写得很清楚：**只有"内容型插件"可以默认开**；带 MCP server 和系统依赖的必须用户显式开（`official-plugin-definitions.ts:43-46`、`:390-398`）。

**办公场景价值**：高。"office 四件套就是四个可插拔插件，每个带 skill + viewer + 一个共享的 visual-judge 子代理"——这正好是 gaea 办公板块该有的形状。

**gaea 落点**：`internal/gaea/skill/builtins.go:18-104`（`office-gongwen` / `genui` / `cad-takeoff` / `office-edit`，都是 `ScopeBuiltin` + `RunInline` + `Path: "(builtin)"`）、`:106-183`（`schedule-edit`）、`:184-272`（`format-convert` / `chart-builder` / `doc-assemble`，前两个是 `RunSubagent`）。**gaea 的内置技能已经是 bundled 语义**（`builtins.go:7-9`：同名文件可覆盖内置，但内置永远在），缺的是"必需资产清单"和"内容 hash 身份"。

**成本**：S（补一张 `builtinRequiredAnchors` 清单即可；gaea 已有等价物，只是以**测试锚点**形式存在）。
**风险**：内置技能正文写在 Go 字符串里（`builtins.go:47-104` 是一大段 raw string），改动成本低但**不可被非工程角色编辑**；办公技能的领域知识如果由业务同事维护，应放 `.gaea/skills/`（项目级）而不是 builtins。

### 2.5 skills-lock

**机制**：仓库根和 `apps/zcode-cli` 各有一个 `skills-lock.json`，记录**外部来源技能**的 `source` / `sourceType` / `computedHash`：

```json
{"version":1,"skills":{"agent-browser":{"source":"vercel-labs/agent-browser","sourceType":"github","computedHash":"939c..."}}}
```

（`clones/zcode/skills-lock.json:1-25`，另有 `apps/zcode-cli/skills-lock.json`）

**关键事实**：`computedHash` 在整个仓库里**只出现在这个文件本身**（全仓 grep 仅命中 `apps/zcode-cli/skills-lock.json`），runtime 不读它。它的作用是**供应链留痕**：`.agents/skills/<name>` 下的技能是从哪个 GitHub 仓库 vendored 进来的、内容哈希是多少（来源与安装位置的对应见 `docs/vercel-react-best-practices-skill.md:5-22`）。

**办公场景价值**：中。办公技能会引用**国标/企业模板**，一旦"技能内容被谁改过"不可追溯，责任就无法界定。但 hash 锁只解决"来源可追溯"，不解决"内容正确"。

**gaea 落点**：无对应物。可落在 `internal/gaea/skill/`（新文件 `lock.go`）+ `.gaea/skills.lock.json`；gaea 更需要的其实是**内容锚点测试**——`skill/office_skill_test.go:29-46` 与 `skill/novel_agents_skill_test.go:27-37` 就是在做这件事（"先读后写"、"逐 run"、"soffice" 等字符串必须在正文里）。这比 hash 更有价值：hash 只能证明"没变"，锚点能证明"该有的条款还在"。

**成本**：S（锚点已有）；hash 锁 S（一个 JSON + 一个校验函数）但**收益低**。
**风险**：hash 锁容易变成"改了就报红、于是大家改 hash"的形式主义；且对内置 Go 字符串技能无效。

### 2.6 多语言

**机制**：**只做 UI 层本地化，不动模型可见内容**。

- `/` 面板与 `$` mention 面板里的官方内置技能，按 `name` 查 `zh-CN`/`en-US` 文案表；只对 `scope === "plugin"` 且属于官方内置插件（或 path 落在官方缓存路径）的技能生效；未命中回退服务端 `description`（`docs/superpowers/specs/2026-06-27-bundled-skill-english-i18n.md:26-33`）。
- 明确非目标：不改 `SKILL.md` 本体、不改 frontmatter 解析、不影响模型可见 selection metadata（`:19-24`）。

**办公场景价值**：中。办公板块是中文主场，但 UI 双语是基本盘；关键是**模型可见的技能描述不要做语言切换**（否则同一技能在两种语言下行为漂移，无法复现）。

**gaea 落点**：`internal/gaea/i18n/`（包存在）+ 前端。gaea 的 `Description` 直接进索引（`index.go:107`），所以模型可见文案与 UI 文案目前是同一份——若要双语，正确做法是在 UI 侧做 name→文案表（照 ZCode），而不是给 frontmatter 加 `description_en`。

**成本**：S。
**风险**：给技能加 `description_zh/en` 会让索引随语言变化，破坏前缀缓存与可复现性（ZCode 正是因此把它列为非目标）。

---

## 3. 子代理（agents/*.md）

### 3.1 定义格式

**机制**：Markdown + frontmatter，正文即子代理的 system prompt 主体。

frontmatter 字段（`clones/zcode/packages/shared/src/subagent-profile-parser.ts:33-106`）：

| 字段 | 语义 | 证据 |
| --- | --- | --- |
| `name` / `description` | 必填；缺失则诊断并丢弃 | `:50-57` |
| `model` | 模型选择；`inherit`/`main`/`sonnet`/`opus`/`haiku` 视为继承 | `subagent-markdown-selection.ts:8-32` |
| `thoughtLevel` | 推理档位（→ `options.reasoningLevel`） | `subagent-markdown-selection.ts:29-31` |
| `color` | 8 种之一，UI 标识 | `subagent-profile-parser.ts:11-20` |
| `permissionMode` | `acceptEdits/auto/bypassPermissions/default/dontAsk/plan`；**project 来源被强制丢弃** | `:22-29`、`:61-64` |
| `maxTurns` | 正整数轮次上限 | `:65`、`:206-213` |
| `memory` | `user`/`project`/`local` | `:66-74`、`:108-112` |
| `tools` / `disallowedTools` | 工具投影（支持 `Bash(git *)` 这类带参写法，取括号前名字） | `:98-99`、`:215-226` |
| `skills` | 该子代理可用的技能名 | `:100` |
| `background` / `injectAgentsMd` | 是否后台、是否注入 AGENTS.md | `:101-102` |
| `mcpServers` | 只接受"父级 server 名"列表，不允许内联定义 | `:49`、`:75-83`、`:103` |

**安全红线（很值得抄）**：项目级 Markdown 属于**仓库输入**，不能通过 frontmatter 把子 runtime 提升到 `bypassPermissions`；只有 user 级或受信插件配置才接受 `permissionMode`（`subagent-profile-parser.ts:61-64` 注释："项目级 subagent markdown 属于仓库输入，不能通过 frontmatter 把 child runtime 提升到 bypass/yolo"）。

### 3.2 内置子代理与工具投影

**机制**：

- 内置只有两个：`general-purpose`（`tools: ["*"]`、`injectAgentsMd: true`、蓝色）与 `explore`（只读搜索，`tools: ["Bash","Glob","Grep","Read","WebFetch","WebSearch","TodoWrite"]`、`injectAgentsMd: false`、青色）（`core/src/subagent/profile.ts:23-37`、`:72-87`）。
- 内置可被同名 profile 覆盖，但**行为不能只按名字套**：`isBuiltInExploreAgentProfile` 要求 `name === "explore" && source === "built-in"`（`profile.ts:39-44`）。
- 工具投影在**两处**生效：① 给模型的"可用 agent 类型"清单里列出工具（`profile.ts:102-113`）；② 真正启动子代理时过滤工具表，并**强制剔除 `EnterPlanMode`/`ExitPlanMode`**——因为子代理没有独立的 plan 审批面，暴露它会让 `ExitPlanMode` 等用户确认而卡死父 turn（`core/src/subagent/tool-policy.ts:7-30`）。
- 子代理上下文是**独立装配**的：CLI 前缀 → agent prompt（stable）→ subagent notes（stable）→ 环境（dynamic）→ 用户指令 → 日期 → 技能索引（`core/src/subagent/context-builder.ts:111-168`）。

### 3.3 结果回传

**机制**：

- `Agent` 工具的 provider 描述里写死："**agent 的最终消息作为 tool result 返回给你，不会展示给用户 —— 由你转述要点**"（`core/src/tool/handlers/agent.ts:106`）。
- 输出 schema 是三态 `oneOf`：`completed`（含 `content[]`、`totalToolUseCount`、`totalDurationMs`、`totalTokens`、`usage`）/ `async_launched`（含 `agentId`、`outputFile`、`canReadOutputFile`）（`agent.ts:22-81`）。
- 结果进模型的字节上限 `120_000`（`agent.ts:20`）。
- 工作流 actor 走另一条更严的通道：`submit_result` 提交结构化终态，port 裁决 `accept`（挂 `turnControl` 终止本 turn）或 `reject`（变成 error tool_result，违规列表可见，模型在同会话内**修复重试**）（`core/src/tool/handlers/submit-result.ts:1-67`），模型可见预算 `16_000` 字节（`:29`）。
- 插件里的 agent 以 `<plugin>:<agent>` 注册，无冲突时同时给裸名别名（`docs/design/v2/plugin-compat.md:360-365`）。

### 3.4 现成的办公样板：`visual-judge`

ZCode 办公四件套**共用一个**插件内置子代理 `agents/visual-judge.md`，它是"办公子代理"最好的参照：

```yaml
name: visual-judge
description: "THE single visual acceptance pass ... it is read-only and edits nothing ... returns one JSON verdict line per page"
color: yellow
model: account:bigmodel-individual-coding-plan/GLM-5.3-Flash
thoughtLevel: max
tools: [Read, Bash]
```

（`apps/zcode-cli/packages/documents-plugin/agents/visual-judge.md:1-8`）

正文的关键约束（`:9-47`）：

- **只审分给它的页**，只读，绝不写交付物或工作区；
- 明确"**用我替代你自己看图，不要两者都做**"——避免重复一次全量渲染；
- 按格式给专门判据：pptx 看投影距离（文字压卡片、单卡半空、投影字号、跨页一致性）、docx 看阅读距离（分页artifact、孤行标题、目录无页码、图空白、页眉页脚连续性）、xlsx 看列被 `####` 截断、错误值、图表类型与数据不符、宽表被切页、pdf 看压边距/分栏断裂/孤行；
- 输出**每页一行 JSON**：`{"page":3,"verdict":"pass"}` 或 `{"page":4,"verdict":"fail","issues":[{"category":"Design","problem":"...","evidence":"..."}]}`，`category ∈ Spec|Visual|Design|Unverified`，无法确认必须 `Unverified` 而不是猜。

**办公场景价值**：极高。这是"AI 交付办公文档"最缺的一环：**渲染后有人（另一个模型）按用户验收标准逐页签字**。它同时示范了三件事：子代理可以**带独立模型与档位**（主模型写、便宜模型审）、可以**只读投影**、可以**返回结构化判决而不是散文**。

**gaea 落点**：
- 声明式 agent 定义：**没有**。gaea 子代理是 Go 写死的 5 个模板（`internal/gaea/cache/spawn.go:51-100`：explore / research / review / security / writing），通过 `RegisterSpawnTemplate` 注册（`:24-45`），模板前缀进 `TemplatePrefix` 参与前缀缓存（`internal/gaea/agent/agent_config.go:59-62`）。
- 已有一个**雏形约定**：网文七角色卡住在 `.gaea/skills/novel-agents/agents/<role>.md`，由 `novel-agents` 技能派发时先 `read_file` 读卡再执行（`internal/gaea/skill/novel_agents_skill_test.go:13-48`）。也就是说 gaea 已经有"agent 卡片"这个东西，只是**没有强制契约、没有模型/工具投影、没有命名空间**。
- 工具投影：gaea 有 `tool.AccessPolicy`（`agent/agent_config.go:26-28`）、`ActiveSchemas`（`:63-66`）与 `cache/skill.go` 的 L3 任务档位（`kind → tools/promptHint/temperature/maxSteps`，`internal/gaea/cache/skill.go:20-60`）。**声明式 per-agent 工具白名单是缺口**。
- 模型覆盖：`internal/gaea/config/subagent_model_test.go` 证明配置层已有按子代理选模型的入口；`skill.Skill.Model`（`skill.go:65`）已支持技能级模型覆盖。

**成本**：M（新增 `internal/gaea/agent/profile.go`：解析 `agents/*.md` + 合入内置 5 模板 + 提供 tools/model/maxTurns 投影；约 3-5 天，含测试）。
**风险**：① 项目级 agent 文件是仓库输入，必须比照 ZCode 丢弃 `permissionMode`（否则 PR 就能提权）；② 声明式 agent 会与 `cache/spawn.go` 的模板前缀缓存冲突——模板变了前缀就变，需把"卡片正文"与"稳定模板前缀"分开（ZCode 的做法是把卡片正文放 subagent_agent_prompt 段，`subagent/context-builder.ts:113-125`）。

---

## 4. Plugins

### 4.1 插件 = 包（而不是"一个 MCP 连接"）

**机制**：插件 id 是 `<name>@<marketplace>`；包内可投影五类组件：

| 组件 | manifest 字段 | 目录约定 | 证据 |
| --- | --- | --- | --- |
| skills | `skills`（路径或路径数组） | `skills/**/SKILL.md` | `plugin-compat.md:339-344` |
| commands | `commands` | `commands/**.md`（文件名即命令名，`_`/`-`/`:` 与嵌套目录都保留） | `:346-358` |
| agents | （P0 只诊断 manifest 的 `agents` 字段） | **约定目录 `agents/*.md` 直接生效**，以 `<plugin>:<agent>` 加载 | `:360-365` |
| MCP | `mcpServers`（内联或路径）+ `.mcp.json` | 两者都读，**manifest 优先** | `:367-392` |
| UI 页面 | `ui.surfaces[]` | `ui://<plugin>/<page>.html` 资源 | `UI_PLUGIN.md:300-331` |

manifest 发现顺序：`.zcode-plugin/plugin.json` → `.claude-plugin/plugin.json`（`plugin-compat.md:258-264`），字段名与 Claude 兼容（`:265-277`）；兼容级别分 `run` / `install-time only` / `diagnostic only` / `unsupported`（`:283-289`），`hooks`（第三方）/`agents`（manifest 形式）/`outputStyles`/`lspServers`/`channels`/`settings` 都是 diagnostic only（`:291-299`）。

变量展开：`${ZCODE_PLUGIN_ROOT}` / `${ZCODE_PLUGIN_DATA}` / `${ZCODE_PROJECT_DIR}` / `${user_config.KEY}` 等，Claude 与 ZCode 变量名同义；**任意进程环境变量只在 MCP secret sink（stdio.env / headers / OAuth clientSecret）里展开**，不得进 `command`/`args`/`cwd`/`url` 这类进程列表可见字段（`plugin-compat.md:414-445`）。

### 4.2 安装 / 更新 / 卸载

- 存储布局：`~/.zcode/cli/plugins/{installed_plugins.json, marketplaces/<m>/marketplace.json, cache/<m>/<name>/<version>/, data/<sanitized-id>/}`；`cache` 是可被替换的版本目录，`data` 是跨版本持久的插件数据目录（`plugin-compat.md:61-115`）。
- 官方内置插件用 **seed**：启动时把 `packages/<plugin>` seed 到官方缓存路径，MCP 命令改写为通过隐藏入口 `__zcode-plugin-host` import 缓存里的 server 并调 `main()`（`plugin-compat.md:155-203`）。
- 卸载是**彻底拆除**：删 record、删版本缓存目录、删 per-plugin data（含 ZCode 物化的 `generated-commands/`）、删 user config 里的 `plugins.enabledPlugins[id]` 与 `plugins.options[id]`；拆除集中在 adapter，bootstrap 只编排；卸载幂等（`plugin-compat.md:117-146`）。
- 组件注册不另设"注销"步骤：每次发现都从 `installed_plugins.json` 重算，删掉 record + 目录即自然去注册（`plugin-compat.md:139-142`）。
- 市场源被删除时**不级联卸载**，安装进入 `Orphaned Installed Plugin` 态：仍可用/可配置/可启停/可显式卸载，但**不能更新**，重新添加同一源后自动重新关联（`docs/adr/0003-marketplace-source-removal-keeps-installed-plugins.md:12-36`）。
- 状态所有权：安装/配置/enabled/来源/抑制态持久化在 `~/.zcode/cli/plugins` 与 CLI config，**UI store 是投影不是权威**（`docs/plugin-management-lifecycle-case-catalog.md:6-14`）。

### 4.3 UI Plugin（`UI_PLUGIN.md` 的宿主 API 与页面协议）

**机制**：插件页面用官方 `@modelcontextprotocol/ext-apps`（MCP Apps）与宿主通信，**不需要 ZCode 专属 SDK**；注册/安装/启用沿用普通插件流程（`UI_PLUGIN.md:18-20`）。

两种页面形态：

1. **工具结果内联页**：MCP 工具 `_meta.ui.resourceUri` + HTML 资源；
2. **会话侧栏面板**：manifest `ui.surfaces[]`，工具用 `_meta.ui.surface` 指到该面板 id（`UI_PLUGIN.md:22-30`、`:317-331`）。

页面能力（宿主支持范围内的官方 API，`UI_PLUGIN.md:339-363`）：`connect()` / `getHostCapabilities()` / `ontoolinput|ontoolresult|ontoolcancelled` / `onhostcontextchanged` / **`callServerTool()`**（沿用 Agent 权限与审批）/ `readServerResource()` / `listServerResources()` / **`sendMessage()`**（发主对话消息）/ **`updateModelContext()`**（把选区加进下一轮，输入区可见可删，不是持久存储）/ `createSamplingMessage()`（App 内调当前任务模型）/ `registerTool()` / `requestDisplayMode()` / `sendSizeChanged()` / `openLink()`（仅 http/https）/ `downloadFile()` / `sendLog()` / `onteardown`。

ZCode 扩展（需先查 `experimental[...]`）：`zcode/widgetState`（`ui/set-widget-state`，**只存视图态，不承诺持久**）、`zcode/resourceSubscribe`、`zcode/surface`、`zcode/csp`（`UI_PLUGIN.md:365-376`）。

边界与配额：**只在 ZCode Desktop 本地工作区支持**（Web/手机/远程只保留普通 MCP 工具记录）；HTML 资源 ≤16 MiB，单资源读取 ≤8 MiB，初始 `structuredContent` ≤64 KiB、页面元数据 ≤16 KiB、`content` ≤32 KiB；页面**没有 Node / 任意文件访问 / 网络（受 CSP）**；camera/mic/geolocation/clipboardWrite 需声明 + 授权（`UI_PLUGIN.md:32`、`:430-445`）。

数据归属建议：业务文档/数据库归**插件服务**，`widgetState` 只是临时视图快照，`localStorage/IndexedDB` 是浏览器侧存储——"清浏览器数据 ≠ 清插件文档"（`UI_PLUGIN.md:432-439`）。

**办公场景价值**：高但**不必重造**。ZCode 的 `office-runtime` 已经把四个查看器（`ui/viewers/{DocxView,XlsxView,PptxView,PdfView}.tsx`，17-27KB 各）与宿主桥（`ui/host.ts`、`ui/rpc.ts`、`ui/panel.html`）一起打进 `dist/server.mjs` + `dist/ui/panel.html`（`official-plugin-definitions.ts:67-72` 的 `OFFICIAL_OFFICE_VIEWER_RUNTIME_PATHS`）。gaea 已有 genui 与办公预览，缺的是"**插件服务拥有业务数据、页面只管视图**"这条归属纪律。

### 4.4 插件市场边界

- 市场源类型：`url` / `github` / `git` / `file` / `directory` / `settings` 可安装；`npm` / `hostPattern` / `pathPattern` 只诊断（`plugin-compat.md:310-316`）。
- 市场源校验刻意比插件校验**轻**：不得为了校验一个源去 clone 数百个条目，远端条目根只在"安装或单插件校验"时才 fetch（`plugin-compat.md:328-335`）。
- `publisherType`（official/community）**不授予信任、不改变安装身份**（`:48-53`）。
- Personal Source 是"可发现目录 + 未来更新来源"双重身份，删除它只删目录投影（ADR 0003）。

**gaea 落点（关键结论）**：`internal/gaea/plugin/` **只是一个 MCP 客户端宿主**，不是包管理器：

- `Spec`（server 名/传输/json）、`transport` 接口、stdio 与 http(含 SSE) 两种传输（`internal/gaea/plugin/plugin.go:26-62`、`transport_stdio.go`、`transport_http.go`）；
- `Host` 管多 server：`Add`/`Remove` 热插拔（`plugin.go:414-462`）、`ServerNames`、`ServerStatus`、`Failures`（`:304-380`）、`StartAll` 全成或全败 / `StartAvailable` 保留可用（`:125-267`）、连接超时 30s（`:268`）；
- 工具/prompts/resources 三件套（`prompts.go`、`resources.go`、`plugin.go:583-616`）、工具名规范化 `mcp__<server>__<tool>` 风格 + 冲突哈希（`:616-640`）、SSRF 防护客户端（`ssrf.go:16-60`）。
- **没有**：manifest、skills/commands/agents 投影、安装/更新/卸载、版本目录、市场、secrets sink 口径。

**成本**：L（要抄的是"插件包"形态，等于新写一层包管理 + 组件投影 + 生命周期）。
**风险**：办公板块是**单机桌面产品**，没有第三方生态诉求；引入包管理会带来"来源可信 / 版本升级 / 卸载残留 / 变量与密钥展开"四个新攻击面。**建议只抄投影层（manifest → skills/commands/agents），不抄市场与安装链**。

---

## 5. MCP

### 5.1 接入方式与权限

- 配置来源与合并优先级（**MCP 独有，和全局配置优先级不同**）：plugin → project → user → runtime/CLI override，即**用户级覆盖项目级**（`docs/design/v2/extension-discovery-priority.md:43-52`）。
- **project origin 的 MCP 默认不可信**：出现在 `/mcp list` 但启动不自动连接；若被同名 user server 遮蔽，则以 user origin 对待，可自动连接（`extension-discovery-priority.md:53-54`）。
- 插件 MCP 从 `.mcp.json` 与 `manifest.mcpServers` 两处读，manifest 优先；有 `command` 无 `type` 时推断 `stdio`；server 名做标识符规范化，`ios-simulator` → 工具名 `mcp__ios_simulator__ios_preflight`（`plugin-compat.md:367-413`）。
- 启动 MCP server 本身是副作用，必须走既有 MCP trust + permission（`plugin-compat.md:447-451`）。
- 子代理的 `mcpServers` 只能引用父级 server 名，不能内联定义（`subagent-profile-parser.ts:75-83`）。

### 5.2 状态列表控制面（`docs/runtime-tools/mcp-status-list-control-plane.md`）

这是本次调研里**最值得整段抄**的一篇设计：

- **`mcp/list` = 状态快照，不是健康检查**。它发往**独立的 `mcp-status` lane control-plane 进程**，进程内用与 runtime 同款 MCP adapter 刷新可信配置的连接状态，再返回 `status()` 的本地快照；project MCP 未信任前只回 `untrusted`，**不会被连接**（`:14-16`）。
- **`mcp/probe` = 单 server 主动检查**，只在用户手动检查或"启用后检查"时触发；结果可以和 `mcp/list` 不同（`:21`）。
- **必须独立进程的理由**（真实事故）：ZCode Protocol 对同一 stdio 连接严格 FIFO 串行，`mcp/list` 会真连全部 server，任一握手不完成就吃满 30s 超时；它原先与 `plugins/uninstall` 共用进程，导致卸载被排到 30s 探测后面（线上复现：卸载等 37s 直到用户强退，`mcp/list` 出现 60.5s = 两个 30s 首尾相接）。拆 lane 后探测再慢也不阻塞插件管理（`:19`）。
- 空闲回收：mcp-status lane 挂 MCP 子进程约 45MB，故启用 `idleTimeoutMs` 默认 5 分钟回收整棵进程树（`terminationKind=expected`），下次 `mcp/list` 透明重拉（冷启约 1s）；chat/plugin lane 不启用（`:17`）。
- 状态语义映射：`failed → error`；`connected/connecting/disconnected` 保持可展示；`disabled/untrusted` 显示为"非成功配置态 + 安全说明"（`:42`）。
- 错误码归一：envelope `3001`/`1000`、JSON-RPC `1006/3101`、HTTP 429/5xx 分别映射到 `server_not_found`/`server_unavailable`/`not_authenticated`/`coding_plan_required`/`rate_limited`/`server_internal_error`/`protocol_error`，细分不了才 `connection_failed`；**只有可信官方 HTTP MCP 读响应头 `X-Request-Id`**，stdio/DNS/timeout 不伪造 id（`:43-45`）。
- UI 合并纪律：只合并 `source === "zcodeagentmcp"` 的行；不得用 name 覆盖 plugin-managed 只读行；请求带 epoch，迟到的批量结果不得覆盖用户编辑或更新的手动检查；同一时刻只允许一个 `mcp/list` 在飞（`:34-41`）。

**办公场景价值**：高。办公板块会挂造价数据库、知识库、企业 OA 之类的 MCP；一旦某个 server 握手卡住 30s，用户会看到"整个设置页假死"。**"状态查询与主动探测分离 + 独立 lane + 空闲回收 + 明确的 untrusted 态"是可直接照搬的产品级口径。**

**gaea 落点**：`internal/gaea/plugin/plugin.go:304-380`（`ServerStatus` / `Failures` / `RecordFailure` / `clearFailure`）、`:221-267`（`StartAvailable` 保留好 server）、`:268`（`mcpConnectTimeout = 30s`）。**缺口**：① 没有"快照 vs probe"的两种语义区分（现在 `Servers()` 就是唯一口径）；② 没有独立进程/lane，MCP 探测与 UI 请求同进程；③ 没有 idle 回收（`Host.Close` 只在整个宿主关闭时调用，`:271-281`）；④ 没有 `untrusted` 态。gaea 的 `config/mcpjson.go` 已有 MCP 配置读写。

**成本**：M（①②③④ 中，先在 `plugin.go` 上分 `Snapshot()`/`Probe()` 两个方法 + 记录 `lastProbeAt`，再把状态查询挪到独立 goroutine 超时隔离，成本可控；真正独立进程可后置）。
**风险**：gaea 是 Wails 单体进程，"独立进程"代价比 ZCode 的 Node CLI 高；建议先用**超时 + 并发隔离**替代，不要为对齐而引入子进程。

---

## 6. Hooks

### 6.1 生命周期事件

**ZCode 支持且只支持 7 个**：`SessionStart`、`UserPromptSubmit`、`PreToolUse`、`PermissionRequest`、`PostToolUse`、`PostToolUseFailure`、`Stop`；**本期不增加新事件**，且持久化类型只允许 `process`（argv 执行，不过 shell）与 `command`（完整 shell，兼容 Claude Marketplace 插件）；明确**不实现** `prompt`/`agent`/`http`/`callback` 与 `if`/`once`/`asyncRewake`（`docs/hooks-runtime-contract.md:9-30`）。

Matcher 语义：缺省/空/`*` 全匹配；**只含字母数字下划线和 `|`** 按精确名或精确名列表；其他字符串按 JS 正则（非法正则不执行该 matcher 并记诊断）；tool 事件匹配 `tool_name` 并兼容 `Agent`/`Task` 别名；`SessionStart` 匹配 `source`；`UserPromptSubmit`/`Stop` 无 matcher（`hooks-runtime-contract.md:109-118`）。

stdin：单行 JSON + 换行，**同时保留 ZCode camelCase 与 Claude snake_case**（`session_id`/`transcript_path`/`cwd`/`permission_mode`/`hook_event_name`/`agent_type` + 各事件字段，`hooks-runtime-contract.md:120-151`）；ZCode 没有 Claude JSONL transcript，所以**临时造一份兼容 transcript**，hook 完成后清理（`:149-151`）。

stdout / 退出码 / 事件效果：

- `0` 成功并解析 stdout；`2` 请求阻断（可阻断事件产生 block/deny，`Stop` 产生"继续一轮"反馈）；其他非零是可恢复失败，**不让整个 turn 崩**（`hooks-runtime-contract.md:163-168`）。
- 事件效果表（可阻断 / 可修改 / 模型可见上下文）见 `:169-179`：`PreToolUse` 可 `updatedInput` 并**重新做 tool schema 校验**；`PermissionRequest` allow 时可返回 `updatedInput` + permissions；`PostToolUse` 本期**不替换** tool output，只追加 `additionalContext`；`Stop` block 表示"不要结束"，最多连续 3 次。
- `async: true` 只对 `command` 有效：立即继续、async stdout **不能** block/改输入/注上下文、仍受 timeout 与取消管理、必须捕获错误（`:194-204`）。

### 6.2 Workspace hook trust

这是 ZCode 在 hook 安全上最有分量的设计，核心是**软门禁（soft gate）**：

第一性原理（`docs/specs/workspace-hook-trust-soft-gate.md:9-17`）：

1. **未信任代码绝不执行**（default-deny 是底线）；
2. **对话不可被安全决策劫持**——被卡住的用户会为了解锁而点"全部信任"，受胁迫的同意不是同意（consent fatigue 反模式），信任决策必须在**无压力的带外场景**做；
3. 策略改变了行为就必须可见；
4. **信任是异步的、作用于未来、不回溯解锁过去**；
5. 执行点复杂度留在准入层。

由此推出：**turn 永不停等审核** → `pending` 本轮直接跳过（blocked）→ 会话内常驻提示条 + 一键去审核 → 删除整条 parked-turn 协议机制（`soft-gate.md:17`）。旧硬门禁的形态是"turn 停在 `activate()` + 强制打开设置页 + 10min watchdog"，新形态是"立即执行 + `WorkspaceHookAdmissionUpdated{pendingCount}` + 聊天底部提示条 `[N 个 Hook 待审核] [去审核] [忽略]`"（`:21-39`）。时序图见 `:41-65`。

信任粒度是 **`workspaceIdentity + hookDeclarationDigest`**，不是"这个项目我信了"：

- 点击"信任"建立当前 `workspaceIdentity + hookDeclarationDigest` 的持久 Trust；**声明的命令、matcher 等语义变化时 digest 变化，新声明自动恢复未信任并重新显示按钮**；未变化声明继续命中原 Trust（`docs/workspace-hook-trust-user-guide.md:6-26`）。
- 配置开关只管"是否启用"，**不建立也不撤销 Trust**；页面不提供"本次授权/批量信任/行内撤销"；关闭设置、超时、断连都等于"没有决定"（`:17-20`）。
- **SessionStart 不补跑**：第一次真实 SessionStart 时未信任声明被跳过，该事件即结束；后续 prompt 不得把已结束的 SessionStart 伪装成 catch-up（`soft-gate.md:122-127`）。
- managed `deny` 始终高于用户 Trust；rollout gate 关闭时所有 project hook hard block，记录保留但不读；stale task/run/remoteSession/generation/snapshot 响应在写 Trust store 前被拒（`user-guide.md:30-36`）。
- Trust store 严格 schema 解析，任一不合规即判 corrupt 并 **fail-closed**（所有消费者同结论，corrupt 下不展示任何 `trusted_persistent`），首次读到损坏文件会改名 `...json.corrupt-<ts>`；`zcode hooks trust status` 显式返回 `workspace_hooks_trust_store_corrupt` 而非 `pending_trust`（`user-guide.md:39-52`）。

**办公场景价值**：高。办公板块的 hook 典型用例是"存盘前校验命名规范/生成台账/打水印"——这些都来自**项目目录里的配置**，而项目目录可能是同事发来的压缩包。

**gaea 落点**：
- 事件面：`internal/gaea/hook/hook.go:34-63`，**10 个事件**（比 ZCode 多 `PostLLMCall`、`SessionEnd`、`SubagentStop`、`Notification`、`PreCompact`），`:65-70` 定义可阻断事件（`PermissionRequest`/`PreToolUse`/`UserPromptSubmit`），`:72-81` 按事件给超时（tool/prompt 5s，其余 30s）。`PreCompact` 的 stdout 当"额外摘要指导"（`:46-50`）——这个点 ZCode 没有，是 gaea 的优势。
- 配置面：`HookConfig{Match, Command, Description, Timeout, Cwd}`（`hook.go:92-105`），`Payload` 有 `Cwd`（`:232-234`），由 `Run(ctx, payload, hooks, spawner)` 执行（`:343`），走 `proc` + `sandbox`（`:27-28`）。
- 信任面：`internal/gaea/hook/trust.go`，**粒度是"项目根 → bool"**（`:9-35`），存在 `~/.gaea/trust.json`（`:16-27`），注释已把理由写清（"attacker controls the project file, not the user-global trust file"，`:9-14`）。

**成本**：M（把 `trust.json` 的 `map[project]bool` 升级为 `map[project]{digest → grantedAt}`，加一个 declaration digest 计算 + 软门禁"跳过并上报 pendingCount"）。**这是 gaea 目前安全水位最薄弱的一环**：per-project bool 意味着**信任一次之后，任何人改 `.gaea/settings.json` 里的 hook 命令都会静默获得执行权**（比如从 `echo` 改成 `curl | sh`）。

**风险**：① gaea 的 hook 有 `PostLLMCall` 这种"可替换模型输出"的事件（`hook.go:40-45`），一旦被恶意项目配置利用，等于**改写模型回答**——这类事件必须比照 `PreToolUse` 纳入可阻断/需信任集合，或明确标注为"仅用户级可配"；② 软门禁要求 UI 有提示条，gaea 前端要跟着改，否则 pending 会被静默跳过、用户以为 hook 生效了。

---

## 7. 命令（slash commands）与提示词组装

### 7.1 自定义命令

**机制**：Markdown 文件即命令模板，文件名即命令名。

- 发现根与技能同序：配置 roots → 用户 `.zcode/commands` → 用户 `.agents/commands` → 项目 `.zcode/commands` → 项目 `.agents/commands` → 插件 roots（`docs/design/v2/extension-discovery-priority.md:30-41`）。
- frontmatter：`description`（缺失时回退正文首行）、`argument-hint`、`allowed-tools`（**只授予本次调用范围的许可，永不覆盖全局 deny**）、`model`（`main`/`lite`/`provider/model`，未知 ref 在启动模型前失败）、`disable-noninteractive`、插件兼容字段 `skills`（声明"执行本命令前必须先加载这些技能"，只渲染"去调用 Skill 工具"的指令，**不直接读正文**）（`docs/design/v2/custom-commands.md:68-96`）。
- 命名：`^[a-z0-9][a-z0-9:-]{0,63}$`；子目录用冒号命名空间 `.zcode/commands/frontend/review.md → /frontend:review`（`:100-112`）。
- **内置命令永远赢同名自定义命令**，自定义 `/help`、`/model`、`/skill` 会被忽略并报诊断（`:47-49`）。
- 正文支持 `$ARGUMENTS` / `$1..$N` / `@file` / `` !`cmd` ``，但动态展开必须走 adapter 与权限契约，不允许 ad-hoc 字符串替换（`:24-32`）。

**内置 prompt 命令**只有两个（`bootstrap/src/builtin-prompt-command.ts:27-66`）：`/init` 与 `/workflow`。`/init` 的做法值得抄：它不是"生成一个 README"，而是给模型一段**带步骤和禁止项的操作规程**——先查 `.zcode/AGENTS.md`/`.agents/AGENTS.md` 是否已存在（存在就停、只报告），再 Read/Glob/Grep 检查仓库，已存在就 `Edit` 而不是整体重写，最后给"推荐内容清单"（`builtin-prompt-command.ts:94-125`）。

### 7.2 提示词分层

**机制**（`core/src/context/builder.ts:83-233`）：

| 顺序 | 段 | injectionTarget | cacheHint |
| --- | --- | --- | --- |
| 1 | CLI/product 前缀（短身份块） | system | — |
| 2 | 稳定身份体：`customSystemPrompt` / `workflowActor` / 默认 identity（含 output style） | system | stable |
| 3 | 动态 system：desktop 上下文、Dynamic Behavior、Session Guidance、Memory、Env Info、Output Style、Context Management、Git 上下文 | system | dynamic |
| 4 | **Skills 索引** | **meta_user** | dynamic |
| 5 | 请求级用户上下文（AGENTS.md 指令 / 项目记忆索引）+ 当前日期 | meta_user | — |
| 6 | 自定义 section | 调用方指定 | — |

装配成消息时按 `injectionTarget × cacheHint` 分桶：system/stable → 一个 system 消息（带 `cacheControl: ephemeral`）；system/dynamic → 第二个 system 消息（正文前加 `\n\n` 左边界）；meta_user → 变成 user 消息附件（`context_prefix` / `skills_listing`），并统一加"以下上下文可能相关也可能不相关"的免责段（`builder.ts:240-354`）。

两个细节值得抄：

- **custom system prompt 不是"只替换稳定身份体"，而是跳过整套默认 system prompt 与动态 system 段**（`builder.ts:126-131` 注释："否则用户提供 custom prompt 后仍会混入 Session Guidance / output style 等动态 system 段"）。
- **workflowActor 与 customSystemPrompt 互斥**，同时出现直接抛错，而不是默默二选一（`builder.ts:90-98`）。

### 7.3 AGENTS.md / 项目规则注入

- 用户级默认指令文件是 `~/.zcode/AGENTS.md`（优先级文件列表含 `AGENTS.md` 时才启用该 fallback）（`adapters/src/context/index.ts:229-243`）。
- 项目级从 cwd **逐级向上**找 `AGENTS.md`，遇到 `projectRoot` 或盘根停（`context/index.ts:204-227`）。
- 合并：用户在前、项目在后；`primary` 取 workspace 来源（`context/index.ts:190-201`）。
- 渲染成 `Contents of <path> (workspace instructions):` / `(user default instructions):` 两个具名段（`core/tests/context-builder.test.ts:393`、`:923-924`；`bootstrap/tests/session-persistence.test.ts:1562`）。
- 记忆索引 `MEMORY.md` 与 AGENTS 指令**共用同一条 request-user-context 注入链路**，只要任一非空就渲染一次 `# agentsMd` 标题（`docs/design/v2/memory/README.md:296-311`）。

**办公场景价值**：高。办公板块的"项目"往往不是代码仓库而是**一个季度的工作目录**；`.gaea/AGENTS.md` 里放"本季度口径/客户名/模板路径"，比塞进技能更合适（技能是能力，AGENTS 是项目事实）。

**gaea 落点**：
- 命令：`internal/gaea/command/command.go:20-49`（`Command{Name,Description,ArgHint,Body,Source}`、`$ARGUMENTS`/`$1..$N`/`$$` 替换）、`:56-80`（多目录加载、后者覆盖前者、symlink 跟随 + 环检测）、`internal/gaea/command/slashtool.go`（把 slash 命令暴露成工具）。
- 提示词：`internal/gaea/boot/sysprompt.go:43-134` 把身份 + output style + 语言策略 + genui 指针 + 记忆画像/索引/固化事实/常用资料 + 技能索引**全部拼成一个 L1 字符串**，再交给 `cache.Compiler`（`internal/gaea/cache/compiler.go:11-26`）。L2 是 `cache.RuntimeLayer`（`internal/gaea/cache/runtime.go:11-43`），有 `Lock()` 保缓存稳定，`SetPromptHint` 必须在 Lock 前调用（`:192-215`）。
- **结论**：gaea 的分层（L1/L2/L3/L4）**比 ZCode 更工程化**，但缺"meta_user 注入位"这一档。gaea 的 L1 里塞了**记忆索引、固化事实、常用资料、技能索引**四类会变的东西，任何一次 `remember` 都会让整个 L1 前缀失效（`cache/identity.go:98-118` 还有 `identity.hash` 落盘对比机制，说明他们自己也知道这件事贵）。
- 建议落点：给 `cache` 增加一个 **L2b "低变动率附件"域**（技能索引 + 固化事实），或至少把 `BuildPinnedFactsBlock`/`BuildMorningPreloadBlock`（`sysprompt.go:81-103`）挪出 L1。

**成本**：S/M（挪块 S；新增 L2b 域 M）。
**风险**：动 L1 装配会改变所有既有会话的前缀字节，`identity.hash` 对比会报"变化"——需要一次性的迁移说明，否则会被当成回归。

---

## 8. 记忆

### 8.1 文件协议

**机制**：一个事实一个**顶层小写 `.md`** 文件，文件名 3-4 个描述性词、不带 type 前缀；同一事实更新原文件、失效可删（`docs/design/v2/memory/README.md:248-256`）。

```yaml
---
name: concise-memory-name
description: one-line recall description
metadata:
  node_type: memory
  type: user            # user | feedback | project | reference
  originSessionId: sess_0123456789abcdef
---
One durable fact.
```

（`memory/README.md:258-287`）

- `type` 语义：`user`（角色/背景/职责/目标/稳定偏好）、`feedback`（确认或纠正的协作方式，正文要含 `Why:` 与 `How to apply:`）、`project`（无法从代码推导的原因/约束/决策，同样要 Why/How）、`reference`（未来去哪查的外部指针）（`:272-279`）。
- `originSessionId` **不由模型生成**：Foreground / Extraction / Dream / Custom Agent **共用的写入边界**在缺失时补一次 frontmatter 序列化，同时补 `node_type: memory` 与当前 session id；已有非空值则原样保留（`:281-287`）。
- `MEMORY.md` 是**短索引、无 frontmatter**；每写一个事实文件就同步一行 `- [Title](file.md) — hook`；Dream 负责后续整理裁剪；Recall 永远排除 basename 等于 `MEMORY.md` 的文件（`:292-294`）。
- 索引读取限额：200 行 / 25_000 字符，超限给**精确 warning**（告诉模型"只加载了一部分，请把索引条目压到 ~200 字符以内，细节移到 topic 文件"）（`core/src/memory/index-content.ts:3-4`、`:8-38`）；读取前先剥 frontmatter 与**顶层 HTML comment**（Markdown lexer 判定，代码块/段落内的 comment 保持原样，`:49-66`）。

### 8.2 双检索分支（单一开关）

`PROJECT_MEMORY_SEMANTIC_RECALL_ENABLED` 是**唯一原始 boolean**，只在 `project-memory-retrieval-branch.ts` 映射成 `default-index | semantic-recall`，其他生产代码**只消费派生值**（`docs/design/v2/memory/README.md:351-368`；实施记录 `docs/superpowers/plans/2026-07-14-memory-active-path.md:14-30`）。

| branch | Main pointer | 加载 `MEMORY.md` | Selector | `relevant_memory` |
| --- | --- | --- | --- | --- |
| `default-index`（当前 active） | 是 | 是 | 否 | 否 |
| `semantic-recall` | 否 | 否 | 是 | 是 |

两分支**严格互斥**，不得出现"pointer/index + Selector/Recall 同时进同一 session"的混合态；切换只改 Core 内部常量，不新增 Settings/配置/协议/环境变量/测试 setter（`memory/README.md:362-365`）。验收要求是：**改一个 boolean 就能切换三处生产行为**（plan:65-67）。

### 8.3 后台抽取与 Dream

- Extraction 在 `memory.extractionEnabled !== false`、成功、非 remote、无 child `agentId` 的 Main turn 之后 **fire-and-forget**；headless 默认 `extractionEnabled=false`，且 gate 必须**早于** enabled-root 解析与 snapshot 获取（`memory/README.md:478-507`）。
- 调度只有"一个 running + 一个 latest pending"，不建 FIFO/job table/持久恢复（`:544-566`）。
- cursor 是内存里的 durable history message UUID：本段 history 出现 project Memory Write/Edit 就跳过模型并推进 cursor（不看 tool result 成功与否）；没有"独立的、非 meta、至少 3 words 的 user text block"就跳过并推进；error/abort 不推进（`:568-576`）。
- Extraction 的执行策略是**窄白名单**：只允许当前真实注册的 `Read`/`Grep`/`Glob`、只读 Bash、memory root 内的 `.md` Write/Edit、memory root 内非递归 `rm .md`；拒绝 MCP/Agent/网络/workspace mutation；拒绝文案本身是冻结合同（`:579-594`）；最多 5 个 model turn，第 5 次响应里的 tool call 仍执行但不再发第 6 次请求（`:595-597`）。
- 失败不重试；no-op 逐字为 `Nothing to save.`，scheduler 不解析该文本（`:597-599`）。
- session close 必须先终止 Extraction 生命周期再释放资源；drain 的 60s 只是非协作 I/O 的关闭上界，**不是**允许后台模型继续跑的宽限期（`:601-626`）。
- Dream：**当前产品链路不自动触发**，builtin command registry 里没有 `/dream`；handler/lock/prompt/`memory_update` 保留，等待 project transcript 隔离形态确定（`:628-637`）。触发条件保留在 handler 内：每 10 分钟最多扫一次、距上次成功 Dream ≥24h、当前 project 至少有 5 个排除当前 session 且 mtime 晚于上次的 transcript（`:636-637`）。

**办公场景价值**：极高。办公交付物高度依赖"上次那个客户的口径"——这正是 `feedback`/`project` 类型事实的用途；而"一个月报模板的口径"属于**项目事实**，应该进记忆（`project`）而不是进技能正文。

**gaea 落点**：
- 存储：`internal/gaea/memory/store.go:114-201`（`fileBackend` = 一事实一文件 + `MEMORY.md` 索引；`sqliteBackend` = gaea.db）、`internal/gaea/memory/file_backend.go:392-425`（`indexLinesExcept` / `flushIndex` / `reindex`，按文件名排序重写 MEMORY.md）。
- 预算：`internal/gaea/memory/memory.go:211-214`（`capMemoryIndex` 按**字节预算 4096 + 行边界**截断，不切半行、不产生悬空 markdown 链接），测试见 `memory/cap_memory_index_test.go:10-210`。**gaea 这里比 ZCode 更细**（ZCode 是 25_000 字符，gaea 是 4096 字节 + 链接完整性）。
- 注入：`boot/sysprompt.go:66-103`（画像 `Compose` / 晨报预载 `BuildMorningPreloadBlock` / 项目本体 `BuildProjectBrief` / 固化正文 `BuildPinnedFactsBlock`）。
- 已有更细的机制：双层激活（`memory/activation.go`：pinned 正文进快照、relevant 仅检索可达）、时态图（`memory/graph.go`、`graphstore.go`）、衰减（`decay.go`）、失效（`invalidate.go`）、引用（`citations.go`）、注入评测（`injection_eval.go`）、空间隔离（`space_ctx.go`、`space_isolation_test.go`）。
- **缺口**：① 没有"一个事实一个文件 + 写入边界补 originSessionId"的**写入面收口**（gaea 的 `remember.go` 是工具入口，但 Write/Edit 直接改 memory 目录时不补元数据）；② 没有"单一开关 → 派生分支"的互斥纪律（gaea 的检索有 keyword/graph/sqlite 多路，容易叠成混合态）；③ 没有 Extraction 的"窄白名单 + 5 轮上限 + cursor 语义"。

**成本**：M（②③ 各 2-3 天；① 需要在 `tool/builtin/editfile.go`/`writefile.go` 加一个 memory-root 专用的写入边界钩子）。
**风险**：gaea 已有 graph/decay/invalidate 等**比 ZCode 多**的能力，硬套 ZCode 的"两分支互斥"会限制既有功能；正确做法是只抄**"唯一开关 + 派生值 + 生产代码不再各自解释"**这条纪律，不抄具体分支集合。

---

## 9. 逐条打分表

> 价值 = 对"办公板块"的复用价值；成本 S≤1d，M≈3-5d，L≥2w。

| # | 机制 | 关键证据 | 办公价值 | gaea 落点 | 成本 | 风险 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 技能两层加载（索引/正文分离） | `context/sections/skills.ts:9-78`；`tool/handlers/skill.ts:78-91` | 高 | `skill/index.go:12,40-84`；`skill/tools.go:67-101` | M | 往返次数增加 |
| 2 | 索引预算 + 降级不丢技能 | `skills.ts:50-57` | 中 | `index.go:40-84`（二分压缩，已更好） | S | — |
| 3 | 已加载技能由 provider 历史判定 | `agent/loaded-skills.ts:13-46` | 高 | 缺；`internal/gaea/agent/` | M | compact 语义需对齐 |
| 4 | 目录只扫一层 + 路径即身份 + 重复诊断 | `adapters/skills/scan.ts:43-74`；`index.ts:80-85` | 中 | `skill.go:205-241`（按 name 去重） | S | 老技能重名行为变化 |
| 5 | 内置/插件/用户的技能分层与"不可卸载" | `bootstrap/app/bundled-skills.ts:16-29,87-95` | 高 | `skill/builtins.go:18-272`（已等价） | S | 内置技能难由业务维护 |
| 6 | 必需资产清单（缺一拒整包） | `bundled-skills.ts:36-41,97-116` | 中 | 缺；现有 `office_skill_test.go:29-46` 是锚点替代 | S | — |
| 7 | skills-lock（供应链留痕） | `skills-lock.json:1-25`；全仓 grep 仅此文件 | 低 | 缺 | S | 易形式主义 |
| 8 | 技能 UI 双语（不动模型可见文案） | `specs/2026-06-27-bundled-skill-english-i18n.md:26-33` | 中 | `internal/gaea/i18n/` | S | 给 frontmatter 加 lang 会破坏缓存 |
| 9 | `agents/*.md` 声明式子代理 | `shared/subagent-profile-parser.ts:33-106` | **最高** | 缺（`cache/spawn.go:51-100` 是写死模板） | M | 项目级提权 |
| 10 | 项目源剥离 `permissionMode` | `subagent-profile-parser.ts:61-64` | 高 | 需新建 | S | — |
| 11 | 子代理工具投影 + 强制剔除 plan 工具 | `subagent/tool-policy.ts:7-30` | 高 | `tool/access.go`；`cache/skill.go:20-60` | M | 与 L3 档位重复 |
| 12 | 结果回传：最终消息即 tool result + 不展示给用户 | `tool/handlers/agent.ts:106`；`submit-result.ts:1-67` | 高 | `agent/subagent_store.go`；`agent/task.go` | M | 大结果截断 |
| 13 | 插件 = 包（skills/commands/agents/MCP/UI） | `plugin-compat.md:337-460` | 高（形态） | `plugin/plugin.go` 只有 MCP | L | 包管理四类攻击面 |
| 14 | 插件安装/更新/卸载/孤立态 | `plugin-compat.md:117-146`；ADR 0003 | 低 | — | L | 数据误删 |
| 15 | UI Plugin（MCP Apps + ui.surfaces） | `UI_PLUGIN.md:18-32,300-393` | 中 | `office-runtime` 已有 viewer | L | 与现有预览冲突 |
| 16 | MCP 状态快照 vs 主动探测 | `runtime-tools/mcp-status-list-control-plane.md:14-23` | 高 | `plugin/plugin.go:304-380` | M | Wails 内独立进程代价 |
| 17 | MCP 状态错误码归一 | 同上 `:43-45` | 中 | 缺 | S | — |
| 18 | project MCP 默认 untrusted | `extension-discovery-priority.md:43-54` | 高 | 缺 | S | 可用性下降需 UI 配合 |
| 19 | 7 个 hook 事件 + process/command | `hooks-runtime-contract.md:9-30` | 中 | `hook/hook.go:34-63`（已有 10 个） | S | 抄成"只支持 7 个"是退步 |
| 20 | hook stdin 双命名兼容 | `hooks-runtime-contract.md:120-151` | 中 | `hook/hook.go:232-234` | M | 兼容层维护 |
| 21 | **workspace hook trust（digest 粒度 + 软门禁）** | `specs/workspace-hook-trust-soft-gate.md:9-65`；`user-guide.md:6-52` | **最高** | `hook/trust.go:9-35`（per-project bool） | M | 需 UI 提示条配合 |
| 22 | 提示词 injectionTarget × cacheHint 分桶 | `context/builder.ts:240-354` | 中高 | `boot/sysprompt.go:43-134`；`cache/` | M | 改 L1 会动所有前缀 |
| 23 | custom system prompt 跳过整套默认段 | `context/builder.ts:126-131` | 中 | 缺 | S | 语义确认 |
| 24 | AGENTS.md 用户+项目合并注入 | `adapters/context/index.ts:204-243` | 高 | `memory/`（项目记忆）+ 缺 AGENTS.md | S | 祖先目录信任面 |
| 25 | 记忆"一事实一文件 + 短索引" | `memory/README.md:248-294` | 高 | `memory/file_backend.go:392-425`（已等价） | S | — |
| 26 | 记忆写入边界补 originSessionId | `memory/README.md:281-287` | 中 | `memory/remember.go`；缺 Write/Edit 边界 | M | 模型看不到的隐式改写 |
| 27 | 单一开关 → 派生检索分支 | `memory/README.md:351-368`；plan:14-30 | 中 | `memory/recall.go`、`search.go` | M | 抑制既有能力 |
| 28 | Extraction 窄白名单 + 5 轮 + cursor | `memory/README.md:568-599` | 中 | 缺 | M | 后台成本 |
| 29 | `/init` 式规程化内置命令 | `bootstrap/builtin-prompt-command.ts:94-125` | 高 | `command/command.go`；缺内置 prompt 命令 | S | — |
| 30 | 命令 `skills:` 前置加载声明 | `custom-commands.md:89-94` | 中 | 缺 | S | 隐式加载 |

---

## 10. 办公板块最该沉淀的 5 个技能

判据（五个问题，全部"是"才值得沉淀成技能）：

1. **重复性**：每月/每客户/每项目都会再来一次吗？
2. **步骤稳定性**：步骤能写成有序清单，而不是每次现想？
3. **有验证闭环**：做完能被机器验证（回读字段、渲染成图、跑校验脚本）吗？
4. **不需要新工具**：现有工具（`bash`/`office 编辑`/`format_convert`/`chart_gen`/`knowledge_*`/`schedule_*`/`cost_*`）就够吗？
5. **知识而非权限**：沉淀下来的是"怎么判断/怎么排版/按什么口径"，而不是"要多大权限"？

形态判据：

- **skill（inline）**：主代理执行、需要连续推理和看中间结果、失败要现场改。
- **skill（subagent）**：要隔离上下文（大量读、大量图）、只需要结论。
- **agents/\*.md**：需要**独立模型/档位**、需要只读投影、需要返回结构化判决。
- **命令（/xxx）**：用户发起入口固定、参数简单、不需要模型先"想起"。
- **固定流程工具**：高风险写操作（必须人确认 + 引擎裁决 + 回执核对），流程要硬化在代码里，不能靠提示词自觉。

| # | 技能 | 为什么该沉淀（判据） | 建议形态 | gaea 落点 | 成本 |
| --- | --- | --- | --- | --- | --- |
| 1 | **monthly-report 月度报表** | 每月都来；步骤固定（取数→清洗→透视→图表→排版→导出→自检）；可机器验证（xlsx 回读 + 渲染 pdf 出页图）；只用现有 xlsx/chart/convert 工具 | **skill（inline）+ 路由表 + 模板 references** | `skill/builtins.go` 新增；复用 `office-edit` 的纪律段；`tool/builtin/{chart_gen,format_convert}.go` | M |
| 2 | **docx 公文/长文档排版** | 国标规则密集（GB/T 9704）；错误代价高（红头字号、版记）；可验证（`postcheck` 类脚本 + 渲染取证） | **skill（inline）+ routes/scenes 分层引用**（ZCode 式：SKILL.md 只给路由表，细节进 `references/`） | 已有 `office-gongwen`（`builtins.go:21-27`）+ `gongwen_body.go`；把它升级为"路由表 + references"，不再是一整段正文 | M |
| 3 | **visual-judge 交付前视觉验收官** | 每次交付都要；判据客观（逐页 pass/fail）；**必须隔离**（不能自己审自己）；量大（页图很吃 token）；要独立模型/档位 | **`agents/visual-judge.md`（声明式子代理，只读 + `tools: [Read, Bash]` + 独立 model/thoughtLevel）** | 新建 `internal/gaea/agent/profile.go` + `.gaea/agents/visual-judge.md`；渲染取证复用 `office-edit` 里已写的 `soffice → pdf → pdftoppm` 链路 | M |
| 4 | **cost-review 造价/清单复核** | 每次投标/结算都要；口径固定（定额匹配、系数明示、宁缺勿误）；**有确定性工具与数据库可裁决**；错误代价极高 | **skill（inline）+ 固定流程工具（已有 `cost_*`）+ 写操作走确认卡** | `skill/builtins.go` 新增 `cost-review`；工具在 `tool/builtin/cost_tools.go`、`cost_compose.go`；口径进 `references/` | S/M |
| 5 | **kb-ingest 知识库入库与引用** | 周期性（每次项目收尾/每季更新）；步骤固定（抽取→切分→打标→入库→回查验证）；可验证（入库后必须能检索回原文并给出引用）；只用 `knowledge_*` 工具 | **skill（inline）+ 一个 `/kb-ingest` 命令入口 + 可选定时任务** | `skill/builtins.go` 新增；`tool/builtin/knowledge_add.go`、`knowledge_search.go`、`internal/gaea/knowledge/`；命令走 `internal/gaea/command/` | S |

补充说明：

- **行程/排期不需要新技能**：`schedule-edit`（`skill/builtins.go:106-183`）已经覆盖"先读后写 / 回执核对 / 引擎裁决 / 确认卡"。它缺的不是技能，而是**流程硬化**——schedule 的写操作应当在工具层就要求 diff 确认卡（技能里已写"任何权限级别都须经 diff 确认卡"，`:119-120`），这与第 11 节"不该抄"第 3 条同源。
- **转换不需要新技能**：`format-convert`（`builtins.go:184-209`）已是 `RunSubagent`；但要补一条纪律：**转换后必须回读校验**（Markdown 转换最容易静默丢表格），这条应加到那里而不是新开技能。
- **五个里面只有 #3 不是 skill**，这是刻意的：**"验收"和"生产"必须是两个上下文、两个模型**，否则会出现"自己写完自己打勾"。这也正是 ZCode 把 `visual-judge` 做成 agents/*.md 而不是 skill 的原因（`documents-plugin/agents/visual-judge.md:3`：*"Use it instead of looking at the page images yourself, never in addition"*）。

---

## 11. 最值得抄的 5 条 / 不该抄的 3 条

### 5 条最值得抄

1. **`agents/*.md` 声明式子代理 + 项目源剥离 `permissionMode` + 工具投影（含强制剔除 plan 工具）**
   证据 `shared/subagent-profile-parser.ts:33-106`、`:61-64`、`core/subagent/tool-policy.ts:7-30`。
   gaea 现在只有 5 个 Go 写死模板；办公板块最需要的"独立模型 + 只读 + 结构化判决"子代理（visual-judge 类）无处安放。这是**投入产出比最高的一条**。

2. **workspace hook trust 精化到声明 digest + 软门禁（pending 即跳过、绝不劫持 turn）**
   证据 `specs/workspace-hook-trust-soft-gate.md:9-65`、`workspace-hook-trust-user-guide.md:6-52`、`hook/trust.go:9-35`。
   gaea 的 per-project bool 是真实漏洞：信任一次后，任何人改 `.gaea/settings.json` 的 hook 命令都能静默执行。

3. **MCP 状态 = 快照，探测 = 单独动作；错误码归一；project server 默认 untrusted**
   证据 `runtime-tools/mcp-status-list-control-plane.md:14-23,42-45`、`extension-discovery-priority.md:43-54`。
   办公板块会挂外部数据库/OA，一个卡 30s 的 server 会让整页假死；"绿色 connected ≠ 健康检查通过"这条口径必须先立。

4. **技能索引/正文分离 + 索引预算降级 + "已加载技能由 provider 历史判定"**
   证据 `context/sections/skills.ts:9-78`、`agent/loaded-skills.ts:13-46`、`tool/handlers/skill.ts:78-91`。
   gaea 已有索引和 `run_skill`，但**大技能的 references 被一次性拼进正文**（`skill/skill.go:384-422`），且 compact 后没有"是否已加载"的判据。这两点是办公长文档场景的真痛点。

5. **插件 manifest → 组件的投影层（skills/commands/agents/mcpServers，`agents/*.md` 目录约定，`<plugin>:<agent>` 命名空间）**
   证据 `plugin-compat.md:337-365`、`official-plugin-definitions.ts:210-248`（办公四件套的必需资产清单）。
   只抄**投影**，不抄市场与安装链：这样办公板块能有一个"能力包"概念（docx 包、造价包、月报包），而不背生态包袱。

### 3 条不该抄

1. **插件市场与安装/更新/卸载/CDN seed 全链**
   证据 `plugin-compat.md:61-146,464-513`、`docs/adr/0003`、`plugin-management-lifecycle-case-catalog.md`（20 个生命周期用例）。
   gaea 是单机 Wails 桌面产品，没有第三方分发诉求；抄进来会同时引入"来源可信 / 版本升级回滚 / 卸载残留 / 变量与密钥展开（`plugin-compat.md:414-445`）"四个新攻击面，而收益只是"我自己给自己的插件打版本号"。

2. **UI Plugin（MCP Apps + `ui.surfaces` + Electron guest 沙箱）**
   证据 `UI_PLUGIN.md:18-32,300-393,430-445`。
   gaea 已有 genui（`internal/gaea/genui` + `tool/builtin/genui_validate.go`）与办公预览（`apps/zcode-cli/packages/office-runtime/ui/viewers/*` 在 ZCode 侧；gaea 侧是 Wails 前端）。再造一套 MCP Apps 宿主意味着**两套页面协议、两套 CSP/配额、两套生命周期**，而办公场景真正缺的是"插件服务持有业务数据、页面只持视图"的归属纪律（`UI_PLUGIN.md:432-439`），这条纪律不需要新协议就能落地。

3. **把技能/命令当成可以改运行时（模型、工具白名单、hooks、fork 执行）的载荷**
   证据 `docs/design/v2/skill.md:26,263-275`（`allowed-tools`/`model`/`effort`/`context: fork` 全部列为 P1/P2，且"出现这些字段默认 ask"）；`custom-commands.md:73-75`（`allowed-tools` 只授予调用范围、永不覆盖全局 deny）。
   ZCode 自己在 spec 里就把这条路**主动降级**了。gaea 已有 `runas: subagent` + `model` + `allowed-tools`（`skill/skill.go:63-65`、`:426-437`），够用；再往前一步（让技能正文能声明"给我 bypass 权限"）就是把技能变成权限后门，而技能文件的信任级别远低于配置文件。

**次优先（可做可不做）**：`/init` 式规程化命令（S，可直接抄 `bootstrap/builtin-prompt-command.ts:94-125` 的写法）、AGENTS.md 用户+项目双源合并（S）、`skills-lock`（收益低）、Claude 兼容 hook stdin（M，gaea 事件面已经比 ZCode 宽，补兼容层的性价比一般）。

---

## 12. gaea 落点核实清单（grep 确认存在）

| 用途 | 文件 | 现状 |
| --- | --- | --- |
| 技能发现/解析/frontmatter | `internal/gaea/skill/skill.go` | 有（name/description/allowed-tools/runas/context/agent/model） |
| 技能索引与预算压缩 | `internal/gaea/skill/index.go` | 有（4000 字符 + 二分压缩） |
| `run_skill` 派发 | `internal/gaea/skill/tools.go`、`parallel.go`、`pipeline.go` | 有（inline/subagent/pipeline/parallel） |
| 内置技能 | `internal/gaea/skill/builtins.go`、`gongwen_body.go`、`cad_takeoff_body.go` | 有（8 个） |
| `read_skill` 工具 | `internal/gaea/tool/builtin/readskill.go` | 有 |
| 系统提示词装配 | `internal/gaea/boot/sysprompt.go` | 有（单 L1 串） |
| 四域前缀缓存 | `internal/gaea/cache/{compiler,identity,runtime,skill,spawn}.go` | 有（L1/L2/L3/L4） |
| 子代理模板 | `internal/gaea/cache/spawn.go` | 有（5 个写死模板，**无声明式**） |
| 子代理运行时 | `internal/gaea/agent/{task,subagent_store,agent_run}.go` | 有 |
| hook 执行 | `internal/gaea/hook/{hook,runner}.go` | 有（10 事件） |
| hook 信任 | `internal/gaea/hook/trust.go` | 有（**per-project bool**） |
| MCP 宿主 | `internal/gaea/plugin/{plugin,transport_stdio,transport_http,ssrf}.go` | 有（无包管理） |
| MCP 配置 | `internal/gaea/config/mcpjson.go` | 有 |
| 自定义命令 | `internal/gaea/command/{command,slashtool}.go` | 有（`$ARGUMENTS`/`$1..$N`/`$$`；`Command.Name` 形如 `git:commit`，即子目录冒号命名空间，见 `command.go:22`） |
| 记忆存储/索引 | `internal/gaea/memory/{store,file_backend,memory}.go` | 有（MEMORY.md + 4096 字节预算） |
| 记忆检索/图谱/衰减 | `internal/gaea/memory/{recall,search,graph,decay,invalidate,citations}.go` | 有（**比 ZCode 多**） |
| 记忆注入 | `internal/gaea/boot/sysprompt.go:66-103` | 有（画像/晨报/项目本体/固化事实） |
| 知识库 | `internal/gaea/knowledge/`、`tool/builtin/knowledge_{add,search}.go` | 有 |
| 办公转换/图表/图示 | `tool/builtin/{format_convert,chart_gen,diagram_gen}.go` | 有 |
| 进度计划/造价 | `tool/builtin/{schedule_tools,cost_tools,cost_compose}.go`、`internal/gaea/{tasks,cost*,schedule*}` | 有 |
| 权限/沙箱 | `internal/gaea/{permission,sandbox,tool/access}.go` | 有 |
| 前端 genui | `internal/gaea/genui/`、`tool/builtin/genui_validate.go` | 有 |

---

## 13. 建议的落地顺序（若采纳）

1. **S（1 周内，低风险）**：`agents/*.md` 解析器 + 项目源剥离 `permissionMode` + 强制剔除 plan 工具 → 先把 `visual-judge` 跑起来（第 10 节 #3）。
2. **S（1 周内）**：hook trust 从 `bool` 升级为 `{digest → grantedAt}`，pending 一律跳过并上报计数（先只做 runtime 侧，UI 提示条随后）。
3. **M（2-3 周）**：技能 references 改为"路由表 + 按需 Read"（保留小技能拼正文的阈值）；补"已加载技能由历史判定"。
4. **M（2-3 周）**：MCP 状态面拆 `Snapshot()` / `Probe()`，加超时隔离与 `untrusted` 态。
5. **M（可与 3 并行）**：把 `sysprompt.go` 里会变的四类块（技能索引、固化事实、晨报、常用资料）挪出 L1，落到 L2 或新增的"低变动率附件"域。
6. **不做**：插件市场/安装链、MCP Apps 页面宿主、技能提权字段。

---

*本报告只做调研与建议，未修改任何代码。所有 `file:line` 均取自当前检出；凡标注"缺/无对应物"的，均经全仓 grep 确认。*
