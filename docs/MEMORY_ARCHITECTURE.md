# gaea 记忆体系三脑架构

> 状态：实施中（2026-08，阶段 1-5）
> 依据：用户决策 + 行业调研（ChatGPT/Claude Memory、SillyTavern 四轨、Character Card V2）

## 一、命名体系（希腊神话宇宙观）

软件品牌与各 AI 伙伴构成一个神话家族，所有伙伴都是 gaea（大地母神）的子女：

| 名称 | 定位 | 说明 |
|---|---|---|
| **gaea** | 软件品牌 / 大地母神 | 整个应用，所有 AI 的母亲 |
| **AIgaea** | 产品类型 | 所有 AI 伙伴的统称（轻语先例："你是'Hermes'——一个AIgaea"） |
| **Hephaestus** | 办公 agent（左脑） | 火与工匠之神 → 工程/方案/办公；原名与品牌重名 "gaea"，已改 |
| **Hermes** | 轻语 agent（右脑） | 信使之神 → 交流/陪伴/角色扮演；原名 "轻语"，已改 |

**规则**：
- 板块/产品名保留"轻语"（launcher 标签、设置面板、路由 key `whisper`、CSS 主题）
- agent 名字语境一律用 Hermes/Hephaestus（提示词自称、记忆三元组、生日、dossier）
- 新 agent 提示词措辞对齐轻语先例：`你是"<名字>"——用户的AIgaea` + "你就是这个应用里的 gaea（AIgaea），不是底层大模型品牌"

## 二、三脑架构

```
┌──────────── 主脑（记忆中枢 · gaea.db SQLite）────────────┐
│  统一记忆 API（读/写/检索/冲突裁决）                      │
│  全局共享层：跨板块用户画像 + 领域知识库（向量 RAG）       │
│  调度路由：人格类→右脑，工作类→左脑，通用事实→主脑         │
└───────┬─────────────────────────┬───────────────────┘
        │ 统一 API                  │ 统一 API
┌───────▼──────────┐        ┌──────▼───────────────┐
│ 左脑：办公记忆     │        │ 右脑：轻语人格记忆     │
│ (Hephaestus)      │        │ (Hermes)             │
│ gaea.db facts 表  │        │ whisper.db（不动）    │
│ Type×Kind 分类    │        │ 6 领域×25 子类        │
│ remember/dream    │        │ 多路检索+向量+生命周期 │
└──────────────────┘        └──────────────────────┘
```

### 数据库布局

| 存储 | 归属 | 内容 |
|---|---|---|
| `Hephaestus.db`（新建） | 主脑 + 左脑 | facts（办公记忆）、profile（全局画像）、knowledge（领域知识库）、migrations |
| `hermes.db`（现有，原 whisper.db） | 右脑 | 轻语全部人格记忆（memory_facts/episodes/三元组/画像/diary 等） |

左右脑物理隔离（人格数据与工作数据信任模型不同），主脑用统一 API 统筹。

### 记忆分类

- 办公记忆沿用 Type × Kind：Type = user / feedback / project / reference；Kind = semantic / episodic / procedural
- 轻语记忆沿用 6 领域 × 25 子类（IDENTITY/SOCIAL/DAILY_LIFE/PURSUITS/INNER_WORLD/TEMPORAL）

### 路由规则（阶段 4）

| 事实类型 | 目标 |
|---|---|
| 人格/关系/情绪/角色互动 | 右脑（轻语 hermes.db） |
| 工作/方案/工程/领域 | 左脑（办公 Hephaestus.db facts） |
| 通用用户偏好/身份/习惯 | 主脑全局画像（Hephaestus.db profile） |

冲突裁决：同一事实跨板块矛盾时，按时间 + 置信度裁决，标记待用户确认。

## 三、实施路线

1. **阶段 1 命名落地**：Hephaestus/Hermes 改名（提示词 + Meta + 记忆内容）✅
2. **阶段 2 主脑底座**：Hephaestus.db 建库 + 统一记忆 API 骨架
3. **阶段 3 左脑接通**：办公记忆从 Markdown 迁 Hephaestus.db（含迁移工具）
4. **阶段 4 调度+画像**：路由规则 + 全局画像 + 冲突裁决
5. **阶段 5 知识库 RAG**：知识迁 Hephaestus.db + 复用轻语向量能力升级检索

## 四、设计原则

- **复用不重建**：向量检索（TF-IDF/稠密 embedding/FTS5）从轻语提取为共享层，办公/知识库直接复用
- **API 统一、存储自治**：统一记忆 API 收口三板块读写；底层存储各自为政不强制合并
- **显式优先**：办公记忆偏显式（用户可管），轻语延续自动提取 + 自我编辑混合
- **不造第四套存储**：新增记忆一律走 Hephaestus.db 或 hermes.db 两库之一
