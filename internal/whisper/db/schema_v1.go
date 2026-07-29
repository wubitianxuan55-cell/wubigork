// Package db — Schema V1: 初始建表
// 100% 对齐 ackem src/main/db/schemaV1.ts
package db

// SchemaV1 初始建表 — 7 张核心表
const SchemaV1 = `
CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS companion_state (
  session_id TEXT PRIMARY KEY,
  version    TEXT NOT NULL,
  state_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chat_history (
  session_id TEXT PRIMARY KEY,
  rows_json  TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_facts (
  id                TEXT PRIMARY KEY,
  domain            TEXT NOT NULL,
  subcategory       TEXT NOT NULL,
  subject           TEXT NOT NULL,
  summary           TEXT NOT NULL,
  weight            REAL NOT NULL DEFAULT 0,
  confidence        REAL NOT NULL DEFAULT 0,
  status            TEXT NOT NULL DEFAULT 'active',
  emotional_context TEXT,
  self_relevance    REAL NOT NULL DEFAULT 0,
  triggers          TEXT,
  triggers_text     TEXT,
  update_trail      TEXT,
  source_session_id TEXT,
  source_turn_index INTEGER DEFAULT 0,
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL,
  derived_from      TEXT,
  fact_layer        TEXT DEFAULT 'raw',
  tier              TEXT DEFAULT 'archival'
);

CREATE TABLE IF NOT EXISTS episodes (
  id                  TEXT PRIMARY KEY,
  summary             TEXT NOT NULL,
  emotional_intensity REAL NOT NULL DEFAULT 0,
  dominant_emotion    TEXT NOT NULL DEFAULT '',
  keywords            TEXT,
  prev_episode_id     TEXT,
  source_session_id   TEXT,
  start_turn          INTEGER DEFAULT 0,
  end_turn            INTEGER DEFAULT 0,
  created_at          TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS procedural_habits (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  ts   TEXT NOT NULL,
  text TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS kv_store (
  namespace  TEXT NOT NULL,
  key        TEXT NOT NULL,
  value      TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (namespace, key)
);
`
