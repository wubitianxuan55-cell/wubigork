// Package db — Schema V10: 成人记忆隐私等级
// 100% 对齐 ackem src/main/db/schemaV10.ts
package db

const SchemaV10 = `
ALTER TABLE memory_facts ADD COLUMN privacy_level TEXT DEFAULT 'normal';
CREATE INDEX IF NOT EXISTS idx_facts_privacy_level ON memory_facts(privacy_level);
`
