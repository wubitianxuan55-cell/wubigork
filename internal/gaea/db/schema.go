// Package db — gaea 主脑数据库网关（嵌入式 SQLite，单例 per userDir）
//
// 三脑架构中的「主脑 + 左脑」存储：facts（办公记忆）、profile（全局画像）、
// knowledge（领域知识库）。右脑轻语记忆保持独立 hermes.db。
package db

// SchemaV1 主脑底座：facts/profile/knowledge 三表 + 版本元数据。
// 迁移链模式对齐 internal/whisper/db（schema_meta.user_version 递增）。
const SchemaV1 = `
CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

-- 办公记忆（左脑）：Type×Kind 分类保留，per-project 隔离
CREATE TABLE IF NOT EXISTS facts (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  project     TEXT NOT NULL,
  name        TEXT NOT NULL,
  title       TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  type        TEXT NOT NULL DEFAULT 'project',
  kind        TEXT NOT NULL DEFAULT 'semantic',
  tags        TEXT NOT NULL DEFAULT '[]',
  body        TEXT NOT NULL DEFAULT '',
  archived    INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT '',
  updated_at  TEXT NOT NULL DEFAULT '',
  UNIQUE(project, name)
);
CREATE INDEX IF NOT EXISTS idx_facts_project ON facts(project);
CREATE INDEX IF NOT EXISTS idx_facts_archived ON facts(project, archived);

-- 全局共享层（主脑）：跨板块用户画像
CREATE TABLE IF NOT EXISTS profile (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  source     TEXT NOT NULL DEFAULT '',
  confidence REAL NOT NULL DEFAULT 1.0,
  updated_at TEXT NOT NULL DEFAULT ''
);

-- 领域知识库（主脑）：从 ~/.gaea/knowledge Markdown 迁入，升级 RAG 用
CREATE TABLE IF NOT EXISTS knowledge (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL UNIQUE,
  title      TEXT NOT NULL DEFAULT '',
  category   TEXT NOT NULL DEFAULT '',
  phase      TEXT NOT NULL DEFAULT '',
  discipline TEXT NOT NULL DEFAULT '',
  tags       TEXT NOT NULL DEFAULT '[]',
  status     TEXT NOT NULL DEFAULT '',
  version    TEXT NOT NULL DEFAULT '',
  author     TEXT NOT NULL DEFAULT '',
  reviewer   TEXT NOT NULL DEFAULT '',
  source     TEXT NOT NULL DEFAULT '',
  body       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_knowledge_category ON knowledge(category);
`

// SchemaV2 成本库（记忆中枢扩展库）：cost_entries 表。
// 成本条目：单价/单位/规格/来源，供方案测算与预结算复用。
const SchemaV2 = `
CREATE TABLE IF NOT EXISTS cost_entries (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL UNIQUE,
  title      TEXT NOT NULL DEFAULT '',
  category   TEXT NOT NULL DEFAULT '',
  unit       TEXT NOT NULL DEFAULT '',
  price      REAL NOT NULL DEFAULT 0,
  spec       TEXT NOT NULL DEFAULT '',
  source     TEXT NOT NULL DEFAULT '',
  tags       TEXT NOT NULL DEFAULT '[]',
  status     TEXT NOT NULL DEFAULT '草稿',
  body       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_cost_category ON cost_entries(category);
`

// SchemaV3 办公记忆生命周期（P1-⑤/⑥ 记忆生命周期 + 溯源）：
// facts 增加最近使用时间（高频排序注入）、来源会话/消息（记忆可控与溯源）。
const SchemaV3 = `
ALTER TABLE facts ADD COLUMN last_used_at TEXT NOT NULL DEFAULT '';
ALTER TABLE facts ADD COLUMN source_session TEXT NOT NULL DEFAULT '';
ALTER TABLE facts ADD COLUMN source_message TEXT NOT NULL DEFAULT '';
`

// SchemaV4 成本库价格更新（P1-⑤/⑥/⑦）：价格源订阅、抓取记录、价格历史。
// price_sources 存订阅源配置（URL/解析器/频率/自定义头）；price_fetch 存
// 每次抓取的待确认结果（无确认不写回 cost_entries）；cost_price_history 存
// 每次发布的价格快照（旧价保留、可回看环比）。
const SchemaV4 = `
CREATE TABLE IF NOT EXISTS price_sources (
  id              TEXT PRIMARY KEY,
  name            TEXT NOT NULL DEFAULT '',
  url             TEXT NOT NULL DEFAULT '',
  parser          TEXT NOT NULL DEFAULT 'sc_table',
  frequency_hours INTEGER NOT NULL DEFAULT 0,
  area            TEXT NOT NULL DEFAULT '',
  headers         TEXT NOT NULL DEFAULT '{}',
  enabled         INTEGER NOT NULL DEFAULT 1,
  last_fetch_at   TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS price_fetch (
  id          TEXT PRIMARY KEY,
  source_id   TEXT NOT NULL DEFAULT '',
  source_name TEXT NOT NULL DEFAULT '',
  url         TEXT NOT NULL DEFAULT '',
  period      TEXT NOT NULL DEFAULT '',
  fetched_at  TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'pending',
  summary     TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_price_fetch_status ON price_fetch(status);
CREATE TABLE IF NOT EXISTS cost_price_history (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL DEFAULT '',
  title      TEXT NOT NULL DEFAULT '',
  unit       TEXT NOT NULL DEFAULT '',
  price      REAL NOT NULL DEFAULT 0,
  source     TEXT NOT NULL DEFAULT '',
  period     TEXT NOT NULL DEFAULT '',
  fetched_at TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_price_history_name ON cost_price_history(name);
`

// SchemaV5 本地语义向量索引（P1-⑦）：semantic_vectors 共享表，按 (kind,id)
// 存 bge-m3 向量 JSON，供成本/知识/办公记忆跨库语义检索。入库即写、增量
// 更新（Ensure 只向量化缺失项），查询只嵌 query + 余弦，避免每查询全量批量。
const SchemaV5 = `
CREATE TABLE IF NOT EXISTS semantic_vectors (
  kind       TEXT NOT NULL,
  id         TEXT NOT NULL,
  vec        TEXT NOT NULL DEFAULT '[]',
  doc        TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (kind, id)
);
`

// SchemaV6 知识库版本历史（P1）：knowledge_history 保存每次内容变更前的快照，
// 供版本回溯与审核（配合 knowledge.version/reviewer 字段）。
const SchemaV6 = `
CREATE TABLE IF NOT EXISTS knowledge_history (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL DEFAULT '',
  title      TEXT NOT NULL DEFAULT '',
  version    INTEGER NOT NULL DEFAULT 0,
  category   TEXT NOT NULL DEFAULT '',
  phase      TEXT NOT NULL DEFAULT '',
  discipline TEXT NOT NULL DEFAULT '',
  tags       TEXT NOT NULL DEFAULT '[]',
  status     TEXT NOT NULL DEFAULT '',
  author     TEXT NOT NULL DEFAULT '',
  reviewer   TEXT NOT NULL DEFAULT '',
  source     TEXT NOT NULL DEFAULT '',
  body       TEXT NOT NULL DEFAULT '',
  changed_at TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_knowledge_history_name ON knowledge_history(name);
`

// SchemaV7 成本库多级分类（分类树 + 条目分类路径）：
// cost_categories 保存分类树节点（parent_id 自引用，0=根，可任意层级）；
// cost_entries 增加 category_path（"一级/二级/…/叶子"）用于树形过滤与分组，
// 旧 category 字段保留（叶子名 + 兼容 cost_search/cost_save 工具）。
// 已有条目按旧分类归入对应一级节点，后续由 Store.EnsureDefaultCategories 播种默认树。
const SchemaV7 = `
CREATE TABLE IF NOT EXISTS cost_categories (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  parent_id  INTEGER NOT NULL DEFAULT 0,
  name       TEXT NOT NULL DEFAULT '',
  sort       INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cost_cat_parent_name ON cost_categories(parent_id, name);
ALTER TABLE cost_entries ADD COLUMN category_path TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_cost_category_path ON cost_entries(category_path);
UPDATE cost_entries SET category_path = category WHERE category_path = '' AND category != '';
`

// SchemaV8 通用任务调度器（阶段 5 T5-1）：tasks 持久化任务表。
// 长任务（价格抓取/文件索引重建/批量导入等）统一走任务队列：状态机
// queued → running → succeeded|failed|cancelled，进度事件经 SSE/Wails 推前端；
// 取消（cancel）、重试（retry）、重启续跑（Startup 把 running 恢复 queued）。
const SchemaV8 = `
CREATE TABLE IF NOT EXISTS tasks (
  id           TEXT PRIMARY KEY,
  kind         TEXT NOT NULL DEFAULT '',
  label        TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'queued',
  progress     INTEGER NOT NULL DEFAULT 0,
  message      TEXT NOT NULL DEFAULT '',
  error        TEXT NOT NULL DEFAULT '',
  retry_count  INTEGER NOT NULL DEFAULT 0,
  max_retries  INTEGER NOT NULL DEFAULT 2,
  payload      TEXT NOT NULL DEFAULT '{}',
  result       TEXT NOT NULL DEFAULT '',
  created_at   INTEGER NOT NULL DEFAULT 0,
  started_at   INTEGER NOT NULL DEFAULT 0,
  finished_at  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status, created_at);
`

// SchemaV9 成本库蒸馏（参照 zaojia-database：价格三要素/口径/有效期/溯源）：
// cost_entries 增加 地区（region）、价格时间/期数（price_date）、价格口径
// （price_type：出厂价/到场价/安装综合价）、有效期至（valid_until）、导入原始
// 行号（source_row，0=手动录入未标注）；cost_price_history 同步记录发布时的
// 地区与口径，保证价格快照可追溯「哪个地区、什么口径、哪一期」。
const SchemaV9 = `
ALTER TABLE cost_entries ADD COLUMN region TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_entries ADD COLUMN price_date TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_entries ADD COLUMN price_type TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_entries ADD COLUMN valid_until TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_entries ADD COLUMN source_row INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cost_price_history ADD COLUMN region TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_price_history ADD COLUMN price_type TEXT NOT NULL DEFAULT '';
`

// SchemaV10 测算项目与沉淀闭环（zaojia-database 蒸馏：我的项目/工程量清单/版本留痕）：
// cost_projects 存测算项目容器；cost_estimate_items 存测算明细行（引用成本库单价、
// 数量×单价自动算金额）；cost_estimate_versions 存不可变版本快照（保存时对明细行
// 做 JSON 快照，支持回看/对比/恢复思路）。沉淀动作把明细行 UPSERT 回 cost_entries。
const SchemaV10 = `
CREATE TABLE IF NOT EXISTS cost_projects (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL DEFAULT '',
  project_type TEXT NOT NULL DEFAULT '',
  scale        TEXT NOT NULL DEFAULT '',
  craft        TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT '编制中',
  note         TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL DEFAULT '',
  updated_at   TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS cost_estimate_items (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id    TEXT NOT NULL DEFAULT '',
  name          TEXT NOT NULL DEFAULT '',
  title         TEXT NOT NULL DEFAULT '',
  category_path TEXT NOT NULL DEFAULT '',
  unit          TEXT NOT NULL DEFAULT '',
  quantity      REAL NOT NULL DEFAULT 0,
  price         REAL NOT NULL DEFAULT 0,
  amount        REAL NOT NULL DEFAULT 0,
  entry_name    TEXT NOT NULL DEFAULT '',
  source        TEXT NOT NULL DEFAULT '',
  note          TEXT NOT NULL DEFAULT '',
  sort          INTEGER NOT NULL DEFAULT 0,
  created_at    TEXT NOT NULL DEFAULT '',
  updated_at    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_estimate_items_project ON cost_estimate_items(project_id);
CREATE TABLE IF NOT EXISTS cost_estimate_versions (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL DEFAULT '',
  version    INTEGER NOT NULL DEFAULT 0,
  total      REAL NOT NULL DEFAULT 0,
  snapshot   TEXT NOT NULL DEFAULT '[]',
  note       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_estimate_versions_project ON cost_estimate_versions(project_id);
`

// SchemaV11 复盘笔记（zaojia-database 蒸馏：结论/适用边界/风险/证据/可信度/有效期/
// 复核状态 + 引用计数）。造价参考指标不做独立表——由已保存版本/已沉淀项目的明细行
// 实时聚合计算（分位数/中位数/均值），避免双写。
const SchemaV11 = `
CREATE TABLE IF NOT EXISTS cost_review_notes (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  title        TEXT NOT NULL DEFAULT '',
  conclusion   TEXT NOT NULL DEFAULT '',
  boundary     TEXT NOT NULL DEFAULT '',
  risk         TEXT NOT NULL DEFAULT '',
  evidence     TEXT NOT NULL DEFAULT '',
  confidence   TEXT NOT NULL DEFAULT '中',
  valid_until  TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT '草稿',
  category     TEXT NOT NULL DEFAULT '',
  project_type TEXT NOT NULL DEFAULT '',
  craft        TEXT NOT NULL DEFAULT '',
  ref_count    INTEGER NOT NULL DEFAULT 0,
  created_at   TEXT NOT NULL DEFAULT '',
  updated_at   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_review_notes_status ON cost_review_notes(status);
`

// SchemaV12 综合单价架构（用户定调：综合单价为一级，人材机为二级，对标
// 「市政成本测算手册」：专业→分部→综合单价子目→人材机组成）：
// cost_entries 增加 人工费/材料费/机械费 三个金额合计（人材机二级汇总）；
// cost_entry_components 保存综合单价子目的人材机组成明细行（kind=人工/材料/机械，
// 名称/单位/数量/单价/金额），一个综合单价子目对应多行组成，构成二级明细。
const SchemaV12 = `
ALTER TABLE cost_entries ADD COLUMN labor_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_entries ADD COLUMN material_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_entries ADD COLUMN machine_fee REAL NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS cost_entry_components (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  entry_name TEXT NOT NULL DEFAULT '',
  kind       TEXT NOT NULL DEFAULT '人工',
  title      TEXT NOT NULL DEFAULT '',
  unit       TEXT NOT NULL DEFAULT '',
  quantity   REAL NOT NULL DEFAULT 0,
  price      REAL NOT NULL DEFAULT 0,
  amount     REAL NOT NULL DEFAULT 0,
  note       TEXT NOT NULL DEFAULT '',
  sort       INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_entry_components_name ON cost_entry_components(entry_name);
`

// SchemaV13 费率追溯列（用户定调：费率入库但仅展示追溯、不参与计算）。
// 管理费/利润/垫资 为金额（元，与市政手册/蜘蛛网口径一致），税率为百分比。
// 字段可空（默认 0 = 未录入）。
const SchemaV13 = `
ALTER TABLE cost_entries ADD COLUMN management_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_entries ADD COLUMN profit_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_entries ADD COLUMN advance_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_entries ADD COLUMN tax_rate REAL NOT NULL DEFAULT 0;
`

// SchemaV14 双空间维度 S1 列落库（docs/gaea-space-dimension-design.md §1-2）：
// facts/tasks 增加 space_id（work/play；ADD COLUMN NOT NULL DEFAULT 'work'
// 对既有行零成本回填为 work），配空间过滤索引。不动 facts UNIQUE(project,name)
// （SQLite 不能 ALTER 约束，跨空间同名冲突留 S1.2 决策）。
const SchemaV14 = `
ALTER TABLE facts ADD COLUMN space_id TEXT NOT NULL DEFAULT 'work';
CREATE INDEX IF NOT EXISTS idx_facts_space ON facts(project, space_id);
ALTER TABLE tasks ADD COLUMN space_id TEXT NOT NULL DEFAULT 'work';
CREATE INDEX IF NOT EXISTS idx_tasks_space ON tasks(space_id, status);
`

// SchemaV15 造价 AI 化 v4.2（docs/gaea-v42-cost-ai-design.md §3）：
// 询价库数据点（四源归一：信息价/OCR报价/供应商比价/手动询价）+ 五算阶段值
// （估/概/预/结/决）。v4.2b 核心包已在 Open 内幂等自建同款表（成本包先行落地、
// 本迁移收编正式版本，CREATE IF NOT EXISTS 双路径兼容）。
const SchemaV15 = `
CREATE TABLE IF NOT EXISTS cost_inquiry_records (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  title       TEXT NOT NULL,
  spec        TEXT NOT NULL DEFAULT '',
  unit        TEXT NOT NULL DEFAULT '',
  price       REAL NOT NULL,
  source      TEXT NOT NULL DEFAULT '手动询价',
  supplier    TEXT NOT NULL DEFAULT '',
  region      TEXT NOT NULL DEFAULT '',
  price_date  TEXT NOT NULL DEFAULT '',
  valid_until TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT '现行',
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cost_inquiry_title ON cost_inquiry_records(title);
CREATE TABLE IF NOT EXISTS cost_stage_values (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL,
  stage      TEXT NOT NULL,
  amount     REAL NOT NULL,
  date       TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(project_id, stage)
);
`

// SchemaV16 AI 组价复核闭环 v4.158.0:cost_compose_records 组价确认留痕。
// 现状痛点:GaeaCostComposeApply 确认回写成本库后,证据链当场即丢(条目 Source
// 只剩「AI组价」四字,不可回看)。本表在每次确认回写成功后落一条完整
// CostComposeView JSON 快照(含 Evidence 证据链/Band 价格带/Components 人材机/
// Checks 合理性校验),供 GaeaCostComposeRecords 回看。仍是无确认不落库:
// 只在确认回写成功后写入(确认即留痕)。
const SchemaV16 = `
CREATE TABLE IF NOT EXISTS cost_compose_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  entry_name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  llm_used INTEGER NOT NULL DEFAULT 0,
  snapshot TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cost_compose_records_entry ON cost_compose_records(entry_name, created_at);
`

// SchemaV17 造价条目匹配键 v4.178.0:cost_entries 增加 code（定额编码/清单编码）
// + cost_estimate_items 增加 code（测算明细行可携带清单编码）。此前条目匹配
// 全靠标题字符串精确/子串匹配（docs/gaea-cost-domain-survey-2026-09.md §缺口 1），
// 无编码体系；编码是组价检索与归因对标的共同上游——同名不同地区/口径的条目
// 靠编码才可精确锚定。code 归一化存储（半角大写、去空白），空串=未录入。
const SchemaV17 = `
ALTER TABLE cost_entries ADD COLUMN code TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_estimate_items ADD COLUMN code TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_cost_code ON cost_entries(code);
`

// SchemaV18 记忆语义图谱 v4.210.0（阶段五 5.1 首刀，docs/gaea-memory-graph-51-design-2026-09.md）：
// memory_events 记忆事件日志（唯一事实源，追加式 INSERT，绝不 UPDATE/DELETE）+
// mem_graph_nodes/edges/meta 投影物化（可整表 DROP 后由日志重建，「日志即真相，
// 删库可重建」）。实体/事件/来源三向边：source -produces→ event -affects→ entity，
// entity -references→ entity（互引）。nodes.embedding 为向量占位列（BLOB，本刀
// 恒 NULL）——5.2 上下文编译的语义检索从这里起步，不先上 Neo4j。
const SchemaV18 = `
CREATE TABLE IF NOT EXISTS memory_events (
  seq INTEGER PRIMARY KEY AUTOINCREMENT,
  at INTEGER NOT NULL,
  op TEXT NOT NULL,
  name TEXT NOT NULL,
  project TEXT NOT NULL DEFAULT '',
  space TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '[]',
  refs TEXT NOT NULL DEFAULT '[]',
  excerpt TEXT NOT NULL DEFAULT '',
  source_session TEXT NOT NULL DEFAULT '',
  source_message TEXT NOT NULL DEFAULT '',
  actor TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_memory_events_name ON memory_events(project, name);
CREATE TABLE IF NOT EXISTS mem_graph_nodes (
  id TEXT PRIMARY KEY,
  ntype TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  desc TEXT NOT NULL DEFAULT '',
  weight REAL NOT NULL DEFAULT 1,
  state TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT '',
  mtype TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '[]',
  space TEXT NOT NULL DEFAULT '',
  project TEXT NOT NULL DEFAULT '',
  embedding BLOB
);
CREATE TABLE IF NOT EXISTS mem_graph_edges (
  src TEXT NOT NULL,
  tgt TEXT NOT NULL,
  etype TEXT NOT NULL,
  PRIMARY KEY (src, tgt, etype)
);
CREATE TABLE IF NOT EXISTS mem_graph_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

// SchemaV19 记忆生命周期三态 v4.213.0（阶段五 5.3 首刀）：facts 增加 pinned
// 列（固化态）。固化=用户明示「这条要长期保留」：免疫衰减归档、豁免归档
// 保留期硬删（CleanupArchived 跳过 pinned=1）、晨报预载/高频排序加权。
// 与 archived 正交：手动归档仍可作用于固化条（用户是老板），但清理不再
// 「90 天一刀切」地吃掉它。衰减不是存储列——衰减是纯函数评分
// （memory.DecayScore，吃 last_used_at/updated_at），三态可查走
// GaeaMemoryLifecycle；状态动作全部落 memory_events（op=pin/unpin/archive）。
const SchemaV19 = `
ALTER TABLE facts ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
`

// SchemaV20 任务会话维度 v4.180 结构刀：tasks 增加 session_id（提交任务的
// 会话标识，空串=非会话入口——cron/系统周期任务诚实留空不造数）。读取端
// Get/List 原样回带，前端任务中心按当前会话过滤（零新绑定：字段随 JSON
// 自动带出）。ADD COLUMN NOT NULL DEFAULT 空串，对既有行零成本回填。
const SchemaV20 = `
ALTER TABLE tasks ADD COLUMN session_id TEXT NOT NULL DEFAULT '';
`

// SchemaV21 记忆双时间轴 v4.240.0（市场调研刀#1，docs/gaea-market-survey-2026-09.md
// §2/§5 候选1）：memory_events 增加 recorded_at（日志写入时刻=事务时间，
// AppendEvent 服务端盖章，调用方不可伪造）。at 从此单一语义=事实时间（事件
// 所述事实的发生时刻）；此前 at 一列身兼二职（既是事实时间又是写入时间），
// 「事实在过去、写入在当下」的回填/导入/做梦衍生类路径无法两轴并存——
// 「日志即真相」缺审计完整性。旧行 recorded_at=0（无记录时间真相），读取端
// 归一回落 at，与 V20「零成本回填+诚实缺省」同口径。对标：Zep/Graphiti
// bi-temporal（valid time vs transaction time）是 2026 记忆层共识口径。
const SchemaV21 = `
ALTER TABLE memory_events ADD COLUMN recorded_at INTEGER NOT NULL DEFAULT 0;
`

// SchemaV22 测算版本号防重（2026-09-19 审计 P2 落地）：先 MAX 后 INSERT 两步
// 无事务+无唯一约束，并发保存同项目会落重复版本号。先清历史重复（bug 产物，
// 保留每组最新 rowid），再建唯一索引做硬约束背书；SaveVersion 写入侧同步改
// 单语句自算版本号（INSERT...SELECT MAX+1 原子，读改写窗口消除）。
const SchemaV22 = `
DELETE FROM cost_estimate_versions WHERE rowid NOT IN (
  SELECT MAX(rowid) FROM cost_estimate_versions GROUP BY project_id, version
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cost_versions_proj_ver
  ON cost_estimate_versions(project_id, version);
`

// SchemaV23 记忆事实单值键 v4.378.0（reasonix memory subject keys 蒸馏）：
// facts 增加 subject_key 列（事实回答的单值问题，如 project.package_manager）。
// 同空间同键仅一条活跃事实——Store.Save 写入侧拒绝撞键并回报持有者（修订走
// 原条目重写而非制造自相矛盾的新记忆）；空串=叙事型事实不参与约束。
// ADD COLUMN NOT NULL DEFAULT 空串，对既有行零成本回填（V19 pinned 同口径）。
const SchemaV23 = `
ALTER TABLE facts ADD COLUMN subject_key TEXT NOT NULL DEFAULT '';
`

// SchemaV24 会话检索索引 v4.384.0（dsh ⑧ session-query 蒸馏首刀）：过往会话
// 事件日志正文的 FTS5 全文索引（keyed by 日志路径，UNINDEXED 列承载元数据，
// 免联表）+ 新鲜度指纹表（size+mtime，搜索时惰性增量索引）。中文降级：FTS5
// unicode61 不切 CJK 子串，MATCH 零命中回退 LIKE 子串扫描（whisper 先例）。
const SchemaV24 = `
CREATE VIRTUAL TABLE IF NOT EXISTS session_fts USING fts5(
  content,
  path UNINDEXED,
  session_id UNINDEXED,
  role UNINDEXED,
  seq UNINDEXED,
  ts UNINDEXED
);
CREATE TABLE IF NOT EXISTS session_index_meta (
  path TEXT PRIMARY KEY,
  size INTEGER NOT NULL,
  mtime INTEGER NOT NULL
);
`

// SchemaV25 工料法成本数据库①：工料机资源库。
//
// 用户定调（2026-10）：工料机（人材机）是**基本数据**，综合单价分析表是靠
// 「工料机 × 消耗定额」核算出来的——即建立工料法成本数据库。此前工料机只
// 作为 cost_entry_components 的从属行挂在某个综合单价下，不能独立检索/维护/
// 调价；本迁移把它升格为独立主数据。
//
// 设计要点：
//   - code 唯一（人工/材料/机械前缀 + 序号），作为定额引用锚点；改址不改引用。
//   - base_price=基准价（编制口径），current_price=现行价（信息价更新后）；
//     两者分离才能做「基准价 vs 现行价」调差。
//   - kind 三分类（人工/材料/机械）与既有 Component.Kind 口径一致；不建严格
//     CHECK 以容忍源文件合并段标签（如「人工+机械」）。
//   - 唯一索引建在 (kind,title,spec,unit)——工料机的身份就是「什么类别、什么
//     名称、什么规格、什么单位」，同规格同单位重复入库没有语义。
const SchemaV25 = `
CREATE TABLE IF NOT EXISTS gf_resources (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  code           TEXT NOT NULL DEFAULT '',
  kind           TEXT NOT NULL DEFAULT '材料',
  title          TEXT NOT NULL DEFAULT '',
  spec           TEXT NOT NULL DEFAULT '',
  unit           TEXT NOT NULL DEFAULT '',
  base_price     REAL NOT NULL DEFAULT 0,
  current_price  REAL NOT NULL DEFAULT 0,
  category_path  TEXT NOT NULL DEFAULT '',
  source         TEXT NOT NULL DEFAULT '',
  supplier       TEXT NOT NULL DEFAULT '',
  region         TEXT NOT NULL DEFAULT '',
  price_date     TEXT NOT NULL DEFAULT '',
  price_type     TEXT NOT NULL DEFAULT '',
  valid_until    TEXT NOT NULL DEFAULT '',
  loss_rate      REAL NOT NULL DEFAULT 0,
  note           TEXT NOT NULL DEFAULT '',
  tags           TEXT NOT NULL DEFAULT '[]',
  status         TEXT NOT NULL DEFAULT '现行',
  created_at     TEXT NOT NULL DEFAULT '',
  updated_at     TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gf_resources_code ON gf_resources(code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gf_resources_ident ON gf_resources(kind, title, spec, unit);
CREATE INDEX IF NOT EXISTS idx_gf_resources_kind ON gf_resources(kind);
CREATE INDEX IF NOT EXISTS idx_gf_resources_title ON gf_resources(title);
CREATE TABLE IF NOT EXISTS gf_resource_prices (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  resource_id INTEGER NOT NULL DEFAULT 0,
  price       REAL NOT NULL DEFAULT 0,
  period      TEXT NOT NULL DEFAULT '',
  region      TEXT NOT NULL DEFAULT '',
  price_type  TEXT NOT NULL DEFAULT '',
  source      TEXT NOT NULL DEFAULT '',
  fetched_at  TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_gf_resource_prices_rid ON gf_resource_prices(resource_id);
`

// SchemaV26 工料法成本数据库②：消耗定额库。
//
// 定额 = 「每单位该子目消耗多少工料机」的标准含量（工料法内核）。全局共享
// （按专业/章节分层），项目层可覆盖含量与资源价（cost_estimate_item_components
// 承载，见 SchemaV27）。
//
// 设计要点：
//   - quota_items.resource_code 引用 gf_resources.code：资源改名改价不破坏引用，
//     资源删除时该行成为孤立引用（读取侧按 resource_code 回查，缺失显形为
//     「资源已删除」而非静默归零）。
//   - amount 不入列：金额=quantity×resource_price 是核算派生量，读时用当前
//     资源价重算（这正是「资源价一涨、所有引用子目同步重算」的落点）。
//   - loss_rate 为行级损耗率（如沥青 3%），与资源级 loss_rate 并存，行级优先。
const SchemaV26 = `
CREATE TABLE IF NOT EXISTS gf_quotas (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  code           TEXT NOT NULL DEFAULT '',
  title          TEXT NOT NULL DEFAULT '',
  specialty      TEXT NOT NULL DEFAULT '',
  chapter        TEXT NOT NULL DEFAULT '',
  unit           TEXT NOT NULL DEFAULT '',
  category_path  TEXT NOT NULL DEFAULT '',
  base_labor     REAL NOT NULL DEFAULT 0,
  base_material  REAL NOT NULL DEFAULT 0,
  base_machine   REAL NOT NULL DEFAULT 0,
  source         TEXT NOT NULL DEFAULT '',
  region         TEXT NOT NULL DEFAULT '',
  price_date     TEXT NOT NULL DEFAULT '',
  note           TEXT NOT NULL DEFAULT '',
  status         TEXT NOT NULL DEFAULT '现行',
  created_at     TEXT NOT NULL DEFAULT '',
  updated_at     TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gf_quotas_code ON gf_quotas(code);
CREATE INDEX IF NOT EXISTS idx_gf_quotas_specialty ON gf_quotas(specialty);
CREATE INDEX IF NOT EXISTS idx_gf_quotas_title ON gf_quotas(title);
CREATE TABLE IF NOT EXISTS gf_quota_items (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  quota_code    TEXT NOT NULL DEFAULT '',
  resource_code TEXT NOT NULL DEFAULT '',
  kind          TEXT NOT NULL DEFAULT '材料',
  title         TEXT NOT NULL DEFAULT '',
  spec          TEXT NOT NULL DEFAULT '',
  unit          TEXT NOT NULL DEFAULT '',
  quantity      REAL NOT NULL DEFAULT 0,
  resource_price REAL NOT NULL DEFAULT 0,
  loss_rate     REAL NOT NULL DEFAULT 0,
  note          TEXT NOT NULL DEFAULT '',
  sort          INTEGER NOT NULL DEFAULT 0,
  created_at    TEXT NOT NULL DEFAULT '',
  updated_at    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_gf_quota_items_quota ON gf_quota_items(quota_code);
CREATE INDEX IF NOT EXISTS idx_gf_quota_items_resource ON gf_quota_items(resource_code);
`

// SchemaV27 工料法成本数据库③：综合单价降级为核算结果 + 测算项贯通工料机。
//
// ① cost_entries 增 price_derived：0=手填/导入价（旧语义，默认，存量零变化）、
//
//	1=由工料机×消耗定额核算而来（核算缓存）。price 列名与对外契约不变，只是
//	语义从「唯一真值」降为「最近一次核算结果缓存」——成本库既有消费方
//	（PriceBand 分位 / contentband 含量对标 / matchindex / 检索 / 图谱 /
//	归因对标）零改动。
//
// ② cost_entry_components 增 quantity_source / price_source：标记该组成行的
//
//	含量与价格来源（global 定额 / project 项目覆盖 / manual 手工 / derived
//	核算派生），项目级覆盖留痕的落点。
//
// ③ cost_estimate_items 增 spec/labor_fee/material_fee/machine_fee/
//
//	management_fee/profit_fee/tax_rate/quota_code/override_*：测算项自带
//	工料机汇总与定额引用，综合单价不再进模版就退化成裸 price。
//
// ④ cost_projects 增取费参数（企管/规费/利润/税率 + 取费基数 + 计量基数）与
//
//	编制说明四段：模版「成本测算」表 13-18 行与「编制说明」sheet 的结构化落点。
const SchemaV27 = `
ALTER TABLE cost_entries ADD COLUMN price_derived INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cost_entry_components ADD COLUMN quantity_source TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_entry_components ADD COLUMN price_source TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_estimate_items ADD COLUMN spec TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_estimate_items ADD COLUMN labor_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_estimate_items ADD COLUMN material_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_estimate_items ADD COLUMN machine_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_estimate_items ADD COLUMN management_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_estimate_items ADD COLUMN profit_fee REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_estimate_items ADD COLUMN tax_rate REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_estimate_items ADD COLUMN quota_code TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_estimate_items ADD COLUMN override_quantity INTEGER NOT NULL DEFAULT 1;
ALTER TABLE cost_estimate_items ADD COLUMN override_price INTEGER NOT NULL DEFAULT 1;
ALTER TABLE cost_projects ADD COLUMN management_rate REAL NOT NULL DEFAULT 0.1;
ALTER TABLE cost_projects ADD COLUMN regulatory_rate REAL NOT NULL DEFAULT 0.02;
ALTER TABLE cost_projects ADD COLUMN profit_rate REAL NOT NULL DEFAULT 0.07;
ALTER TABLE cost_projects ADD COLUMN tax_rate REAL NOT NULL DEFAULT 0.09;
ALTER TABLE cost_projects ADD COLUMN base_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_projects ADD COLUMN measure_area REAL NOT NULL DEFAULT 0;
ALTER TABLE cost_projects ADD COLUMN code_std TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_projects ADD COLUMN basis_note TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_projects ADD COLUMN fee_note TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_projects ADD COLUMN scope_note TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_projects ADD COLUMN review_note TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_cost_entries_derived ON cost_entries(price_derived);
CREATE INDEX IF NOT EXISTS idx_estimate_items_quota ON cost_estimate_items(quota_code);
`

// SchemaV28 工料法④：**清单层落库**（分部分项工程量清单 + 项目费率）。
//
// 用户定调（2026-10-07）：「造价数据库只是一个数据库，你可以录入**费率、
// 清单、测算模版**，但不是把它们加起来」+「清单、定额、工料机是什么关系？
// 我让你录入的清单呢？」——
//
//	工料机=资源价格（基本数据）→ 定额=每单位子目消耗多少工料机 →
//	**清单=工程实体的分部分项列表（编码/名称/单位/工程量），每条套定额**；
//	费率=项目的取费参数（录入数据）。加总计算不在库里发生——导出五表
//	工作簿（活公式）时由 Excel 算，gaea 只存数据。
//
// V25/26/27 只落了前两层与定额骨架，清单与费率没有落库——本版补齐。
//
// ① gf_projects：一次导入=一个项目（五表封面 + 取费区费率：企管/规费/
//
//	利润/税率/利润基数含规费开关/控制价——全部是录入数据，不是计算结果）。
//
// ② gf_bill_items：清单项 × 工程量 × 引用定额（quota_code 套定额）。
//
//	UNIQUE(project_id, code)：同项目同编码走更新（幂等导入）。
const SchemaV28 = `
CREATE TABLE IF NOT EXISTS gf_projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  file_name TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  duration TEXT NOT NULL DEFAULT '',
  pricing TEXT NOT NULL DEFAULT '',
  management_rate REAL NOT NULL DEFAULT 0.1,
  regulatory_rate REAL NOT NULL DEFAULT 0.02,
  profit_rate REAL NOT NULL DEFAULT 0.07,
  tax_rate REAL NOT NULL DEFAULT 0.09,
  profit_includes_regulatory INTEGER NOT NULL DEFAULT 0,
  control_price REAL NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  UNIQUE(name)
);
CREATE TABLE IF NOT EXISTS gf_bill_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id INTEGER NOT NULL,
  code TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  unit TEXT NOT NULL DEFAULT '',
  division TEXT NOT NULL DEFAULT '',
  quantity REAL NOT NULL DEFAULT 0,
  quantity_expr TEXT NOT NULL DEFAULT '',
  quota_code TEXT NOT NULL DEFAULT '',
  feature TEXT NOT NULL DEFAULT '',
  price_override REAL NOT NULL DEFAULT 0,
  sort INTEGER NOT NULL DEFAULT 0,
  UNIQUE(project_id, code)
);
CREATE INDEX IF NOT EXISTS idx_bill_items_project ON gf_bill_items(project_id);
CREATE INDEX IF NOT EXISTS idx_bill_items_quota ON gf_bill_items(quota_code);
`

// SchemaV29 定额编码统一溯源列：gf_quotas.orig_code 保存工作簿里的原始清单
// 编码。统一规则（applyBillItems 前的定额落库段）：原码全局未用→直接用；
// 同源重导入（orig_code+源文件相同）→复用更新；标题+单位+工序特征全等
// →跨项目复用同一条定额（真统一——同内容共享全局标准）；否则原码-N 消歧。
// 没有它，「同项目重导入」与「他项目占用」无法区分，索引与引用都会漂。
const SchemaV29 = `
ALTER TABLE gf_quotas ADD COLUMN orig_code TEXT NOT NULL DEFAULT '';
`

// SchemaV30 工料法⑤：**清单↔定额 1:N 组合**（用户拍板 2026-10-07「如果一个
// 清单是几个定额组成的呢？」）。标准清单计价一条清单项可套多条定额子目
// （各带自身工程量），综合单价是几条定额的加权和——此前 gf_bill_items.
// quota_code 单值只能表达 1:1。
//
// gf_bill_quota_links 是引用事实全量表；单值 quota_code 列保留为「主定额」
// （旧展示/导出锚点不动），link 表在导入链双写。quantity=该定额参与合价的
// 工程量（0=跟随清单工程量）——录入数据，加总仍归五表 Excel。
const SchemaV30 = `
CREATE TABLE IF NOT EXISTS gf_bill_quota_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bill_item_id INTEGER NOT NULL,
  quota_code TEXT NOT NULL,
  quantity REAL NOT NULL DEFAULT 0,
  sort INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  UNIQUE(bill_item_id, quota_code)
);
CREATE INDEX IF NOT EXISTS idx_bill_quota_links_item ON gf_bill_quota_links(bill_item_id);
CREATE INDEX IF NOT EXISTS idx_bill_quota_links_quota ON gf_bill_quota_links(quota_code);
`
