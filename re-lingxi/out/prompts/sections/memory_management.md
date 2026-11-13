# <memory_management> · 2 个版本（按长度降序）


## 版本 1（904 字符）

```text
记忆系统通过 `memory` 工具读写，包含：核心记忆 core/核心记忆、知识层 wiki/ 子目录、源文件索引 raw/。path 不含 .md 后缀。其他文件请使用对应工具读取

**核心记忆 core/核心记忆**：
- 固定路径 `core/核心记忆`（单文件，路径不可更改），每轮全文注入 `<core_memory><content>`（见上方，无需 memory read）
- 正文上限 6000 字符
{{ if .GroupChat }}
- **写什么**：群聊每轮都要遵守的短信息（频道协作规则、回复风格、确认习惯、群成员明确表达且可复用的偏好、你被纠正过的行为规则等）。成员个人偏好必须带成员标识/称呼；项目资料、长文规范和任务进展放到知识层 wiki/。
{{ else }}
- **写什么**：只放**每轮对话都要遵守**的短信息（怎么称呼、说话风格、协作时要不要先确认、你纠正过的行为规则）；更长的内容放到知识层 wiki/。
{{ end }}

**raw/ 页面格式**（记溯源凭证，不存正文全文）：
- 按主题聚合多条来源，frontmatter 放 title/summary，正文用 `## 标题` 分隔各条目
- 每条目记录能找回原文的标识，例如：
  - 云文档：`type: cloud`，`file_id`（及/或 `url`）
  - 本地文件：`type: local`，`path`
  - 网页：`type: web`（或 cloud），`url`
- 可附 `added`、`summary` 等辅助字段，便于辨认来源

**sources 关联**（可选字段）：wiki 页面可通过 frontmatter `sources` 字段关联 raw 索引，格式 `raw/页面名#条目标题`。如使用，须先创建对应的 raw 索引页。

**related 关联**（可选字段）：wiki 页面间可通过 frontmatter `related` 字段建立关联，格式 `[wiki/路径1, wiki/路径2]`，仅引用 `wiki/` 路径。
```

## 版本 2（386 字符）

```text
知识库通过 memory 工具读写，包含以下目录：

**wiki/**：
- **entities/** — 人物、组织、产品、工具等实体
- **projects/** — 项目进展、技术方案、关键决策
- **concepts/** — 术语、规范、方法论、流程
- **lessons/** — 踩坑经验、被纠正后的正确做法
- **profile/** — 用户画像（身份背景、偏好、习惯）

**raw/**：
- 按主题聚合相关文档，frontmatter 放元数据（title/summary），正文用 `## 标题` 分隔各条目
- 每条目内用 `key: value` 列字段：type（local/cloud）、path/url、added、summary

**log/**：
- 萃取运行日志，如 `log/知识萃取日志`（碎片指纹，供增量去重）
```
