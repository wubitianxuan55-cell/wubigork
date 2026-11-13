# 灵犀工具契约（符号 × 模板簇 join · 修正版）

- 工具 **25** 个（`prompt.*ToolDesc`）；模板簇 **23** 个
- join 规则：先按签名 define 把工具锁定到**簇**，再在簇内找「参数 snake_case → `<arg>_arg`」。
- 标 `簇N` 的文案来自该簇；标 `—` 的表示簇内无对应文案（参数说明在别处拼）。

| 工具 | 参数 | 簇 | 文案 |
|---|---|---|---|
| `AskUserQuestion` | `header` | 簇10 | 显示为卡片标题的超短标签（最多 12 个字符）。 |
| `AskUserQuestion` | `multi_select` | 簇10 | 设为 true 以允许用户选择多个选项。当选项不互斥时使用。 |
| `AskUserQuestion` | `options` | 簇10 | 选项列表（2-4 个）。顺序即展示顺序，先排好序再填入。 |
| `AskUserQuestion` | `question` | 簇10 | 向用户提出的简短明确问句，如果过长适当换一行展示。 |
| `AskUserQuestion` | `questions` | 簇10 | 问题列表（{{ .Min }}-{{ .Max }} 个），按呈现顺序排列。 |
| `Await` | `brief` | 簇15 | `search` 时描述搜索目的；`fetch` 时可选，传入后具体说明轻量模型要从网页中提取凝练的字段、范围、时间或统计口径，不传或留空则返回经过可读性清理的网页正文。 |
| `Await` | `max_tokens` | 簇15 | 本次等待返回的输出 token 预算（上限）。省略或 0 表示不限制；sandbox 另有自身截断长度，实际返回量可能少于该值，未返回部分落盘由 file 工具读取。配合 terminate:true 时，输出达此上限即终止进程，无需等 timeout 到期 |
| `Await` | `task_id` | 簇15 | 后台任务 ID |
| `Await` | `terminate` | 簇15 | 仅在用户**明确要求**终止/停止/杀掉该后台进程时才传 true（例如「把它杀掉」「停止这个进程」「别让它再跑了」）。
其余所有情况（只想看输出、问进度、催促、等待完成、超时后想继续）都必须传 false 或省略。
默认 false：到点后仅返回已产生的输出 |
| `Await` | `timeout` | 簇15 | 最大等待秒数（1-{{ .max_timeout }}）。任务在此时间内未完成则返回 timeout 状态 |
| `Bash` | `brief` | 簇22 | 简要描述此次命令的目的（5-10 个字） |
| `Bash` | `command` | 簇22 | 要执行的 shell 命令（传递给 bash -c） |
| `Bash` | `justification` | 簇22 | 仅在 `require_escalated` 时填写面向用户的批准问题；其他情况请省略。 |
| `Bash` | `run_in_background` | 簇22 | 立即将进程切换到后台运行，不等待输出 |
| `Bash` | `sandbox_permissions` | 簇22 | 可选的单条命令沙箱覆盖选项。省略或设为 `use_default` 时沿用当前执行策略；设为 `require_escalated` 时需提供 `justification`，用于向用户发起非沙箱执行的授权请求。 |
| `ConnectorExec` | `action` | 簇9 | 执行动作：list=列出连接器的可用工具，call=调用指定工具 |
| `ConnectorExec` | `arguments` | 簇9 | 调用 MCP 工具时附带的参数，自由格式的 JSON 对象（key-value），具体字段名和类型由目标 MCP 工具的 inputSchema 决定。action=call 时有条件必填，取决于目标工具是否需传参 |
| `ConnectorExec` | `brief` | 簇9 | 一句话描述调用目的 |
| `ConnectorExec` | `id` | 簇9 | MCP 服务唯一标识 |
| `ConnectorExec` | `tool` | 簇9 | 要调用的 MCP 工具名称，action=call 时必填 |
| `ConnectorManage` | `action` | 簇11 | 操作类型：add=添加, remove=移除, list=列出, update=更新配置, get=查看详情 |
| `ConnectorManage` | `brief` | 簇11 | 一句话描述此次图片搜索操作的目的。 |
| `ConnectorManage` | `command` | 簇11 | 启动 mcp 服务的 bash 指令。mode=stdio 时必填。 
这是一条自包含的启动命令，所有配置（包括环境变量、认证凭据、参数等）都必须直接融入此命令中，不需要也不支持从外部注入任何配置。常见做法：
- 需要环境变量时，直接在命令中设置，如 API_ |
| `ConnectorManage` | `description` | 簇11 | MCP 服务描述 |
| `ConnectorManage` | `endpoint` | 簇11 | HTTP 模式 MCP Server 的 endpoint URL，如 https://api.example.com/mcp。mode=http 时必填 |
| `ConnectorManage` | `headers` | 簇11 | HTTP 模式自定义 HTTP 请求头键值对，如鉴权信息。格式为 {"请求头名称": "请求头值"}，如 {"Authorization": "Bearer xxx", "X-API-Key": "key123"} |
| `ConnectorManage` | `id` | 簇11 | MCP 服务唯一标识。get/remove/update 时必填 |
| `ConnectorManage` | `mode` | 簇11 | 连接模式。stdio=本地启动子进程（需指定 command），http=连接远程 endpoint（需指定 endpoint）。add 时必填 |
| `ConnectorManage` | `name` | 簇11 | 计划的短名称，约 3–4 个词。 |
| `CreatePlan` | `action` | 簇11 | 操作类型：add=添加, remove=移除, list=列出, update=更新配置, get=查看详情 |
| `CreatePlan` | `edit_all` | 簇11 | 是否替换所有匹配项（默认 false，仅首次）。 |
| `CreatePlan` | `edit_find` | 簇11 | 要查找的精确文本。 |
| `CreatePlan` | `edit_replace` | 簇11 | 替换后的新文本；空字符串表示删除该段。 |
| `CreatePlan` | `edits` | 簇11 | edit 时的替换列表，每项含 find（精确查找）与 replace（替换文本）；可选 all=true 替换全部匹配。按序执行。小改优先用 edits。 |
| `CreatePlan` | `name` | 簇11 | 计划的短名称，约 3–4 个词。 |
| `CreatePlan` | `overview` | 簇11 | 2-3 句概述，说明背景和核心思路。submit 必填；edit 时可选。 |
| `CreatePlan` | `plan` | 簇11 | 完整方案正文，Markdown 格式。submit 必填；edit 时可选（传入则整段覆盖正文）。 |
| `CreatePlan` | `todo_content` | 簇11 | 待办项的描述内容。 |
| `CreatePlan` | `todo_id` | 簇11 | 待办项唯一标识，如 impl-1、test-2。 |
| `CreatePlan` | `todos` | 簇11 | 可选待办项列表。edit 时若传入则整表覆盖。每项只包含唯一 id 和一句话 content。 |
| `ExecuteJupyter` | `brief` | 簇16 | 简要描述执行此操作的目的 |
| `ExecuteJupyter` | `code` | 簇16 | 延续上一次调用的执行环境状态编写增量代码。已导入的模块和已创建的变量均可直接使用，严禁写入任何重复 import 已存在模块的语句。 |
| `File` | `action` | 簇16 | 操作类型。read: 读取文件; write: 覆盖写入; append: 追加写入; edit: 精确替换文本 |
| `File` | `brief` | 簇16 | 简要描述执行此操作的目的 |
| `File` | `content` | 簇16 | write/append 时写入的内容（空字符串会被校验拒绝；需要清空文件时传一个空格） |
| `File` | `edit_all` | 簇16 | 是否替换所有匹配项（默认 false，仅替换首次匹配）；谨慎使用全量替换，可能导致大范围误修改，仅在你确认需要替换所有匹配项时才设为 true |
| `File` | `edit_find` | 簇16 | 要查找的精确文本 |
| `File` | `edit_replace` | 簇16 | 替换后的新文本 |
| `File` | `edits` | 簇16 | edit 时的替换列表，每项包含 find（要查找的精确文本）和 replace（替换后的文本，空字符串表示删除）；按序执行，单项未匹配时跳过，不影响其他项 |
| `File` | `end_line` | 簇16 | read 时的结束行号（从 1 开始，包含该行），不指定则读取到文件末尾 |
| `File` | `path` | 簇16 | 目标文件的路径，支持绝对路径或工作目录的相对路径 |
| `File` | `start_line` | 簇16 | read 时的起始行号（从 1 开始），不指定则从第一行开始读取 |
| `GenImage` | `aspect_ratio` | 全局唯一 | 图片宽高比，默认 "16:9" |
| `GenImage` | `brief` | 歧义 | — |
| `GenImage` | `input_images` | 全局唯一 | 图生图模式的参考输入图片列表，最多 {{ .max_input_images }} 张。每项须包含 type 字段（"path" 表示本地路径，"url" 表示网络地址）；type 为 path 时填 path（本地绝对路径），type 为 url 时填 ur |
| `GenImage` | `input_path` | 全局唯一 | 参考图片的本地绝对路径（当 type 为 path 时必填） |
| `GenImage` | `input_type` | 全局唯一 | 参考图来源：path 表示本地文件路径，url 表示网络地址 |
| `GenImage` | `input_url` | 全局唯一 | 参考图片的 URL（当 type 为 url 时必填） |
| `GenImage` | `input_weight` | 全局唯一 | 参考图片的权重（0.0~1.0），影响生成结果与参考图的相似度 |
| `GenImage` | `prompt` | 歧义 | — |
| `ImageSearch` | `brief` | 歧义 | — |
| `ImageSearch` | `image_count` | 全局唯一 | 每个 query 返回的图片数量，不传或传 0 时默认返回 5 张，最大 5。 |
| `ImageSearch` | `image_ratio` | 全局唯一 | 图片比例档位，按宽高比判定。**只能选一个**，不传表示不限制。
- `细长竖图`：9:16 及更窄，如手机壁纸
- `竖图`：介于 9:16 与正方形之间，如海报
- `方图`：1:1，如头像、图标
- `横图`：介于正方形与 16:9 之间，如常规配图
- |
| `ImageSearch` | `image_size` | 全局唯一 | 图片尺寸档位，按短边（宽高中较小的一边）判定。不传表示不限制。
- `小图`：短边 300px 以下
- `中图`：短边 300 ~ 600px
- `大图`：短边 600 ~ 1080px
- `超大图`：短边 1080px 以上 |
| `ImageSearch` | `queries` | 歧义 | — |
| `ManageSkills` | `action` | 簇6 | 要执行的操作。"list" 列出已安装技能；"install" 从指定路径安装技能；"uninstall" 卸载指定技能。 |
| `ManageSkills` | `names` | 簇6 | 要卸载的技能目录名列表（仅 uninstall 时必填）。传入技能目录下的子目录名，非完整路径。 |
| `ManageSkills` | `path` | 簇6 | 技能目录的完整路径（仅 install 时必填）。该目录下必须包含 SKILL.md 文件。 |
| `Members` | `action` | 歧义 | — |
| `ProjectInsight` | `action` | 歧义 | — |
| `ProjectInsight` | `brief` | 歧义 | — |
| `ProjectInsight` | `content` | 歧义 | — |
| `RenderWidget` | `code` | 簇10 | HTML 片段，要保持精简，HTML 为不含文档级标签的片段 |
| `RenderWidget` | `name` | 簇10 | 小部件名称，简短描述该小部件展示的内容，例如“销售趋势折线图” |
| `Search` | `brief` | 歧义 | — |
| `Search` | `queries` | 歧义 | — |
| `Search` | `time` | 全局唯一 | 可选的时间过滤器，将结果限制在最近的时间范围内；仅在 `action=search` 时使用。 |
| `Search` | `type` | 歧义 | — |
| `ShareFile` | `message` | 簇10 | 随文件一起发送的简短说明文字（可选）。用于向用户解释文件内容或用途。 |
| `ShareFile` | `paths` | 簇10 | 要发送的文件路径数组。每个路径必须是**绝对路径**（完整路径，如 /home/lingxi/workspace/output/report.pptx），禁止使用相对路径。支持图片、文档、代码等常见文件类型。 |
| `SwitchMode` | `reason` | 簇10 | 申请切换的简短原因，说明当前目标为什么更适合目标模式。 |
| `SwitchMode` | `target_mode` | 簇10 | 要申请切换到的目标模式。"plan" 用于探索和设计方案；"agent" 用于写入文件、运行代码等完整执行。 |
| `TimerTask` | `action` | 簇18 | 操作类型: add（创建）、list（查看列表）、update（修改）、delete（删除） |
| `TimerTask` | `enabled` | 簇18 | 任务启用状态。仅 update 时可选：true 表示启用任务，false 表示禁用任务。若只是切换启用状态，只传 enabled；若用户明确要求同时修改其他字段，可以和 title/prompt/schedule 一起传 |
| `TimerTask` | `prompt` | 簇18 | Agent 执行的任务描述。add 时必填，update 时可选。

重要：prompt 必须是直接的任务指令，绝对不能包含"创建定时任务"、"设置提醒"、"每天执行"等暗示定时任务的内容，否则会在执行时再次创建定时任务导致无限循环。

正确示例：
- "查询 |
| `TimerTask` | `schedule` | 簇18 | 调度时间，支持两种格式：1) 日期时间如 "2026/3/31 18:26" 表示一次性任务；2) cron 表达式如 "0 0 9 * * *" 表示周期性任务。add 时必填，update 时可选 |
| `TimerTask` | `task_id` | 簇18 | 任务 ID，update 和 delete 操作时必填。delete 前仍必须先用 list 查看当前最新任务列表；若用户未明确提供 task_id，而是通过任务标题、时间、状态等描述来指代任务，也应先用 list 定位 task_id |
| `TimerTask` | `title` | 簇18 | 定时任务标题，即前端展示的任务名称，如"每日天气提醒"、"周报生成"。add 时必填，update 时仅在用户明确要求修改标题时才传；如果只是启用/禁用或修改其他字段，不要传 title |
| `TodoWrite` | `merge` | 簇7 | 是否与已有 todos 合并。为 true 时，按 id 字段把新项合并进已有列表，未出现的属性可保持不变。为 false 时，用新列表整体替换已有 todos。 |
| `TodoWrite` | `todo_content` | 簇7 | 待办项描述。新增项必填；更新已有项时可省略。 |
| `TodoWrite` | `todo_id` | 簇7 | 待办项唯一标识，如 impl-1。 |
| `TodoWrite` | `todo_status` | 簇7 | 待办状态：pending、in_progress、completed、cancelled。 |
| `TodoWrite` | `todos` | 簇7 | 待办项列表。每项含唯一 id、status；content 在新增或 merge=false 时必填，更新已有项时merge可省略以保留原文。 |
| `ViewImage` | `brief` | 簇10 | 一句话描述本次创作目的（如"撰写年会邀请函"），会作为上下文传递给创作模型。 |
| `ViewImage` | `paths` | 簇10 | 要发送的文件路径数组。每个路径必须是**绝对路径**（完整路径，如 /home/lingxi/workspace/output/report.pptx），禁止使用相对路径。支持图片、文档、代码等常见文件类型。 |
| `ViewImage` | `question` | 簇10 | 向用户提出的简短明确问句，如果过长适当换一行展示。 |
| `WikiMemory` | `action` | 歧义 | — |
| `WikiMemory` | `brief` | 簇10 | 一句话描述本次创作目的（如"撰写年会邀请函"），会作为上下文传递给创作模型。 |
| `WikiMemory` | `new_content` | 簇10 | — |
| `WikiMemory` | `old_text` | 簇10 | — |
| `WikiMemory` | `path` | 歧义 | — |
| `WikiMemory` | `query` | 簇10 | — |

- 簇内命中 **86** / 未命中 **16**
