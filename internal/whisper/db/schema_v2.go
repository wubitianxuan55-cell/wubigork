// Package db — Schema V2: 增量迁移 V1 → V2
// 新增知识图谱 + 对话追踪 + OpenForU + 全文搜索
// 100% 对齐 ackem src/main/db/schemaV2.ts
package db

const SchemaV2 = `
CREATE TABLE IF NOT EXISTS knowledge_triples (
  id              TEXT PRIMARY KEY,
  subject         TEXT NOT NULL,
  predicate       TEXT NOT NULL,
  object          TEXT NOT NULL,
  confidence      REAL NOT NULL DEFAULT 0,
  source_fact_ids TEXT,
  created_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS turn_traces (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  date       TEXT NOT NULL,
  trace_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS diary (
  date      TEXT PRIMARY KEY,
  content   TEXT NOT NULL,
  meta_json TEXT,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS openforu_workspaces (
  id         TEXT PRIMARY KEY,
  data_json  TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS openforu_sessions (
  id         TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  data_json  TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS openforu_runs (
  id         TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  data_json  TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS shared_events (
  id         TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  data_json  TEXT NOT NULL,
  created_at TEXT NOT NULL
);

-- FTS5 全文搜索虚拟表
CREATE VIRTUAL TABLE IF NOT EXISTS memory_facts_fts USING fts5(
  fact_id,
  subject,
  summary,
  triggers_text,
  content='memory_facts',
  content_rowid='rowid'
);

CREATE VIRTUAL TABLE IF NOT EXISTS episodes_fts USING fts5(
  episode_id,
  summary,
  keywords_text,
  dominant_emotion,
  content='episodes',
  content_rowid='rowid'
);
`
