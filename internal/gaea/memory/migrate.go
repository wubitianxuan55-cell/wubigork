package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SQLiteStoreForProject 直接用 project slug 构造 SQLite 后端。迁移工具用它
// 遍历 ~/.config/gaea/projects/*/memory 时，不必反推 cwd。
func SQLiteStoreForProject(db *sql.DB, project string) Store {
	if db == nil || strings.TrimSpace(project) == "" {
		return Store{}
	}
	return Store{backend: &sqliteBackend{db: db, project: project}}
}

// migrateMarker 是 Hephaestus.db profile 表中的迁移完成标记。
const migrateMarker = "legacy_memory_migrated"

// MigrateLegacyFileMemory 将旧 Markdown 记忆（~/.config/gaea/projects/<slug>/memory）
// 幂等迁移到 Hephaestus.db facts 表。首次完成后写 profile 标记，后续启动跳过，
// 避免每次扫描文件。旧 .md 文件保留作备份不删除。返回迁移的项目数。
func MigrateLegacyFileMemory(userDir string, gdb *sql.DB) (int, error) {
	if userDir == "" || gdb == nil {
		return 0, nil
	}

	// 已迁移过则跳过
	var marker string
	err := gdb.QueryRow("SELECT value FROM profile WHERE key = ?", migrateMarker).Scan(&marker)
	if err == nil && marker != "" {
		return 0, nil
	}

	projectsDir := filepath.Join(userDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		// 无 projects 目录 = 全新安装，直接标记完成
		_, _ = gdb.Exec("INSERT OR REPLACE INTO profile(key, value, source, confidence, updated_at) VALUES(?,?,?,?,datetime('now'))",
			migrateMarker, "done", "migration", 1.0)
		return 0, nil
	}

	migrated := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		memDir := filepath.Join(projectsDir, e.Name(), "memory")
		if _, err := os.Stat(memDir); err != nil {
			continue // 项目无 memory 目录
		}
		fileStore := Store{Dir: memDir}
		sqliteStore := SQLiteStoreForProject(gdb, e.Name())

		// active 记忆
		for _, m := range fileStore.List() {
			if _, err := sqliteStore.Save(m); err != nil {
				return migrated, fmt.Errorf("migrate %s/%s: %w", e.Name(), m.Name, err)
			}
		}
		// archived 记忆（先导入再归档，保持归档状态）
		for _, am := range fileStore.ListArchived() {
			if _, err := sqliteStore.Save(am.Memory); err != nil {
				return migrated, fmt.Errorf("migrate archived %s/%s: %w", e.Name(), am.Name, err)
			}
			if _, err := sqliteStore.Archive(am.Name); err != nil {
				return migrated, fmt.Errorf("migrate archive %s/%s: %w", e.Name(), am.Name, err)
			}
		}
		if len(fileStore.List()) > 0 || len(fileStore.ListArchived()) > 0 {
			migrated++
		}
	}

	// 标记完成
	_, err = gdb.Exec("INSERT OR REPLACE INTO profile(key, value, source, confidence, updated_at) VALUES(?,?,?,?,datetime('now'))",
		migrateMarker, "done", "migration", 1.0)
	if err != nil {
		return migrated, fmt.Errorf("write migration marker: %w", err)
	}
	return migrated, nil
}
