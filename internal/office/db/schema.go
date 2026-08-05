// Package db — 方案编写板块数据库网关（嵌入式 SQLite，单例 per officeDir）
//
// P1 数据底座：projects/proposals/sections/files/versions/templates 六表 + schema_meta。
// 迁移链模式对齐 internal/gaea/db（schema_meta.user_version 递增）。
package db

const SchemaV1 = `
CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  category   TEXT NOT NULL DEFAULT '',
  client     TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS proposals (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title        TEXT NOT NULL,
  category     TEXT NOT NULL DEFAULT '',
  template     TEXT NOT NULL DEFAULT '',
  requirements TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'draft',
  version      INTEGER NOT NULL DEFAULT 1,
  bid_summary  TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL DEFAULT '',
  updated_at   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_proposals_project ON proposals(project_id);

CREATE TABLE IF NOT EXISTS sections (
  id          TEXT PRIMARY KEY,
  proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
  parent_id   TEXT NOT NULL DEFAULT '',
  "index"     INTEGER NOT NULL DEFAULT 0,
  level       INTEGER NOT NULL DEFAULT 1,
  title       TEXT NOT NULL DEFAULT '',
  content     TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'pending',
  sources     TEXT NOT NULL DEFAULT '[]',
  created_at  TEXT NOT NULL DEFAULT '',
  updated_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_sections_proposal ON sections(proposal_id, parent_id, "index");

CREATE TABLE IF NOT EXISTS files (
  id          TEXT PRIMARY KEY,
  proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
  kind        TEXT NOT NULL DEFAULT 'attachment',
  name        TEXT NOT NULL,
  path        TEXT NOT NULL DEFAULT '',
  size        INTEGER NOT NULL DEFAULT 0,
  markdown    TEXT NOT NULL DEFAULT '',
  ocr_status  TEXT NOT NULL DEFAULT '',
  meta        TEXT NOT NULL DEFAULT '{}',
  created_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_files_proposal ON files(proposal_id);

CREATE TABLE IF NOT EXISTS versions (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
  version     INTEGER NOT NULL,
  summary     TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_versions_proposal ON versions(proposal_id, version);

CREATE TABLE IF NOT EXISTS templates (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  sections    TEXT NOT NULL DEFAULT '[]',
  created_at  TEXT NOT NULL DEFAULT '',
  updated_at  TEXT NOT NULL DEFAULT ''
);
`

// SchemaV2 招标解析结果表：结构化字段 + 来源定位。
const SchemaV2 = `
CREATE TABLE IF NOT EXISTS parse_results (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id TEXT NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
  file_id     TEXT NOT NULL DEFAULT '',
  field       TEXT NOT NULL,
  value       TEXT NOT NULL DEFAULT '',
  page        INTEGER NOT NULL DEFAULT 0,
  start       INTEGER NOT NULL DEFAULT 0,
  end         INTEGER NOT NULL DEFAULT 0,
  snippet     TEXT NOT NULL DEFAULT '',
  confidence  REAL NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_parse_results_proposal ON parse_results(proposal_id);
`

// SchemaV3 项目事实基线：跨方案共享的项目级事实（工期/单位/人员/设备/术语等）。
const SchemaV3 = `
CREATE TABLE IF NOT EXISTS project_facts (
  project_id TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
  facts      TEXT NOT NULL DEFAULT '{}',
  updated_at TEXT NOT NULL DEFAULT ''
);
`

// SchemaV4 章节字数目标与字数（P3 大纲与撰写引擎）。
const SchemaV4 = `
ALTER TABLE sections ADD COLUMN word_target INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sections ADD COLUMN words INTEGER NOT NULL DEFAULT 0;
`
