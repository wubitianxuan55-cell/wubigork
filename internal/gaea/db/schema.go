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
