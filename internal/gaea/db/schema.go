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
