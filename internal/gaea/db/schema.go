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
