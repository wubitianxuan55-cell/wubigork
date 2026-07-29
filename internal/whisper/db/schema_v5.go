// Package db — Schema V5: 年龄动态计算
// 100% 对齐 ackem src/main/db/schemaV5.ts
package db

const SchemaV5 = `
ALTER TABLE memory_facts ADD COLUMN age_value INTEGER;
ALTER TABLE memory_facts ADD COLUMN age_birth_year INTEGER;
ALTER TABLE memory_facts ADD COLUMN age_birthday_mmdd TEXT;
ALTER TABLE memory_facts ADD COLUMN age_recorded_at TEXT;
ALTER TABLE memory_facts ADD COLUMN age_is_estimate INTEGER DEFAULT 0;
`
