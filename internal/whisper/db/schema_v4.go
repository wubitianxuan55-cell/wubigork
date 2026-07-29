// Package db — Schema V4: 记忆联想层 + 遗忘体系
// 100% 对齐 ackem src/main/db/schemaV4.ts
package db

const SchemaV4 = `
CREATE TABLE IF NOT EXISTS memory_associations (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  fact_id_a          TEXT NOT NULL,
  fact_id_b          TEXT NOT NULL,
  association_type   TEXT NOT NULL DEFAULT 'related',
  strength           REAL NOT NULL DEFAULT 0.5,
  last_activated_at  TEXT,
  FOREIGN KEY (fact_id_a) REFERENCES memory_facts(id),
  FOREIGN KEY (fact_id_b) REFERENCES memory_facts(id)
);

CREATE INDEX IF NOT EXISTS idx_assoc_fact_a ON memory_associations(fact_id_a);
CREATE INDEX IF NOT EXISTS idx_assoc_fact_b ON memory_associations(fact_id_b);

CREATE TABLE IF NOT EXISTS temporal_anchors (
  id                  TEXT PRIMARY KEY,
  anchor_date         TEXT NOT NULL,
  anchor_type         TEXT NOT NULL,
  recurrence_rule     TEXT,
  linked_fact_ids     TEXT,
  emotional_valence   REAL DEFAULT 0,
  emotional_intensity REAL DEFAULT 0,
  domain              TEXT,
  summary             TEXT NOT NULL
);

ALTER TABLE memory_facts ADD COLUMN sensitivity TEXT DEFAULT 'normal';
`
