// Package proposal — 旧 JSON 方案无损迁移
package proposal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// MigrateLegacyJSON 将旧版 JSON 方案导入 SQLite（幂等，只读旧数据，不删原文件）。
// 全部旧方案挂到「未归档项目」；迁移完成后写 schema_meta 标记。
func MigrateLegacyJSON(st *Store) (int, error) {
	if st == nil || st.db == nil {
		return 0, nil
	}
	var marked string
	_ = st.db.QueryRow("SELECT value FROM schema_meta WHERE key='legacy_migrated_v1'").Scan(&marked)
	if marked == "1" {
		return 0, nil
	}
	entries, err := readLegacyIndex(st.legacyDir)
	if err != nil {
		return 0, nil // 无旧数据
	}
	proj, err := st.EnsureDefaultProject()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if st.HasProposal(e.ID) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(st.legacyDir, e.ID+".json"))
		if err != nil {
			continue
		}
		var p Proposal
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		p.ProjectID = proj.ID
		p.Version = 1
		if p.CreatedAt == "" {
			p.CreatedAt = now()
		}
		if p.UpdatedAt == "" {
			p.UpdatedAt = now()
		}
		var fileDocs []FileDoc
		if p.BidSummary != nil {
			fileDocs = p.BidSummary.RawFiles
		}
		if err := st.ImportLegacy(&p, fileDocs); err != nil {
			continue
		}
		count++
	}
	if _, err := st.db.Exec("INSERT OR REPLACE INTO schema_meta(key, value) VALUES('legacy_migrated_v1', '1')"); err != nil {
		return count, err
	}
	return count, nil
}

func readLegacyIndex(dir string) ([]indexEntry, error) {
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, err
	}
	var entries []indexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
