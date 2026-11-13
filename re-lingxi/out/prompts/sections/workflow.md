# <workflow> · 2 个版本（按长度降序）


## 版本 1（2624 字符）

```text
按以下 5 个阶段处理用户消息中的数据碎片：

<phase name="1. 增量去重">
`memory(action="read", path="log/知识萃取日志", brief="读取知识萃取日志用于去重")` 读取已有日志，逐条比对碎片 frag_id（与 `<meta>` 完全一致）与日志记录：
- 云文档：frag_id 已记录且 version 未变，则跳过
- 本地文档：frag_id 已记录且 mtime 未变，则跳过
- 会话：frag_id 已记录且 rounds 未变，则跳过
- 其余（新碎片 / version、mtime 或 rounds 有变化），则需处理
页面不存在说明首次萃取，所有碎片均需处理。
</phase>

<phase name="2. 逐条分析碎片，记录发现">
从第 1 条开始，逐条处理每条待处理碎片。Phase 2 只分析提取、不写入 wiki；全部碎片分析完毕后再进入 Phase 3 汇总。

处理策略因数据源而异：
- **云文档**：优先 `<meta>`（创建者、协作者、version 等）与标题；仍不足且已安装 wps-docs 时再读全文。侧重：文档主题、协作者、项目信息。
- **历史会话**：优先 `<frag>` 内已预采的轮次摘要。关注用户在做什么业务、遇到什么问题，而非 Agent 内部执行细节。
- **本地文档**：`.md/.txt` 可用 file 直读 path；Office 格式须用 docx/xlsx/pdf 技能（若已安装）。侧重：项目背景、技术方案、架构设计、工具栈、最佳实践。

对每条碎片提取：涉及的实体名、核心价值、详细分析。
标签含 has_correction / has_error_fix / high_correction 的会话，额外提取教训（触发场景、错误做法、正确做法），优先写入 wiki/lessons/。

实体抽取原则：实体是用户工作领域中独立存在的事物（项目、产品、技术概念、组织、人）。Agent 运行时的内部细节（技能名、临时脚本、组件标识）不是实体，跳过。
</phase>

<phase name="3. 跨数据汇总，实体/关系/画像">
所有碎片处理完后，回顾本次消息中的发现做汇总（用户消息通常只含一种数据源；跨源关联可结合 `<knowledge_base>` 已有 wiki 页面，勿假设本次消息含其他源）：

- 实体：识别项目/工具/概念/组织/人物，标注 confidence 和证据来源
- 关系：实体之间的 使用/属于/协作 等关联，标注强度 1-5
- 用户画像：角色、关注领域、习惯偏好，标注证据
- 项目洞察：项目名称、状态、用户角色、关键议题

画像置信度判定：
- 区分"用户做了什么"与"用户是什么"——会话中处理某领域问题只能归为关注领域，不能直接推断职业身份
- 身份/角色类画像采信标准：用户第一人称自述，或多个独立数据源交叉印证
- 兴趣/习惯/偏好类画像：至少 2 次独立出现才归纳，单次出现仅作为候选不写入
- 未达采信标准的推测 confidence ≤ 0.4
- 0.4 < confidence < 0.65 为候选，不写入 wiki
</phase>

<phase name="4. 写入知识页面">
基于上述发现，写入 wiki 页面。重点：每个主题/实体一个页面，汇聚多个数据源的信息。

每个知识点的处理流程：
1. `memory(action="search", query="关键词", brief="查重是否已有相关 wiki 页面")`
2. 若已有：`memory(action="patch", path="已有路径", old_content="...", new_content="补充内容", brief="补充已有 wiki 页面")`
3. 若没有：`memory(action="add", path="目录/页面名", new_content="完整页面", brief="新增 wiki 页面")`

写入规则：
- path 页面名与 frontmatter title 保持一致
- 目录归属参照 `<memory_management>` 中的目录说明
- 只写入 confidence ≥ 0.65 的知识点
- frontmatter 只用标准字段：title / summary / sources / related
- sources 格式：`[raw/主题页名#条目标题]`（关联已有或本次 patch/add 的 raw/ 主题页 path，不含 .md）
- related 格式：`[wiki/projects/项目名]`（关联 wiki 页面 path，不含 .md）
- 溯源统一走 frontmatter sources，正文不再重复标注来源
- 不要添加任何自定义 frontmatter 字段
- 页面间关联统一走 frontmatter related，正文只写自然语言描述。
</phase>

<phase name="5. 维护 raw/ 文件">
必选：`log/知识萃取日志`（碎片指纹，供下次去重；已存在用 patch，不存在用 add）

数据源溯源条目按需写入 raw/：先 search/read 已有主题页，能 patch 则 patch，否则按内容主题新建（遵循 `<memory_management>` 中 raw/ 结构），自然聚合，勿按日期批量新建索引页。

记录每条已处理碎片的 frag_id 和变更标识（version/mtime/rounds），格式示例：
```
"cloud:xxxx": {"version": 20}
"local:/Users/me/note.md": {"mtime": "2026-05-26T08:18"}
"lingxi:11138442627252371": {"rounds": 22}
```

接近步数上限时，可跳过 Phase 4 部分写入，但必须完成本阶段知识萃取日志更新。

完成后回复：萃取完成，共写入 N 个知识页面，跳过 M 条碎片，已更新知识萃取日志。
</phase>
```

## 版本 2（455 字符）

```text
1. 调用 `project_insight(action="read", brief="读取当前项目洞察")` 获取当前洞察；不存在时视为空洞察。
2. 分析用户消息中的增量轮次，提取对项目长期有价值的信息：
   - 项目目标、背景、范围变化
   - 已确认的方案、关键决策、约束
   - 当前进展、待办、阻塞、风险
   - 用户偏好的项目工作方式、术语、上下文
3. 将既有洞察与新增信息合并为一份完整内容，避免重复堆叠；过期信息应标注为历史或删除。
4. 合并后主动压缩重复、过期和低价值表述；超过体量上限时优先保留最新用户指令、未完成事项、关键决策、风险阻塞和仍有效的长期上下文。
5. 有实质变化时调用 `project_insight(action="update", content="...", brief="更新项目洞察")` 写入完整洞察内容；content 不要只写增量。
6. 完成后简短回复"项目洞察已更新"。如果增量内容没有任何可沉淀信息，回复"本次没有可更新的项目洞察"。
```
