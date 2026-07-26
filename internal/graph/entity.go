package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wubigork/wubigork/internal/project"
)

// ── 实体数据库 ───────────────────────────────────────────────

// EntityType 实体类型
type EntityType string

const (
	EntityCharacter  EntityType = "character"
	EntityLocation   EntityType = "location"
	EntityItem       EntityType = "item"
	EntityEvent      EntityType = "event"
	EntityConcept    EntityType = "concept"
	EntityOrganization EntityType = "organization"
)

// EntityRelation 实体间关系
type EntityRelation struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
	Type   string `json:"type"` // appears_in / located_at / owns / member_of / influences
	Label  string `json:"label,omitempty"`
}

// EntityDB 实体数据库（存于项目 .wubigork/entities.json）
type EntityDB struct {
	Entities  []Entity         `json:"entities"`
	Relations []EntityRelation `json:"relations"`
}

// Entity 一个故事实体
type Entity struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        EntityType        `json:"type"`
	Description string            `json:"description,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"` // 自由属性
	ChapterRefs []string          `json:"chapter_refs,omitempty"` // 出场章节
}

// LoadEntityDB 从项目目录加载实体数据库
func LoadEntityDB(projectDir string) (*EntityDB, error) {
	path := filepath.Join(projectDir, ".wubigork", "entities.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EntityDB{}, nil
		}
		return nil, err
	}
	var db EntityDB
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("解析实体数据库失败: %w", err)
	}
	return &db, nil
}

// Save 保存实体数据库
func (db *EntityDB) Save(projectDir string) error {
	dir := filepath.Join(projectDir, ".wubigork")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "entities.json"), data, 0644)
}

// SyncFromProject 从项目文件同步实体（角色/组织/地点来自 characters.json 和 lorebook）
func (db *EntityDB) SyncFromProject(pm *project.Manager) error {
	// 从角色文件同步
	chars, err := pm.ReadCharacters()
	if err == nil && chars != nil {
		for _, ch := range chars.Characters {
			db.upsertEntity(Entity{
				ID:          ch.ID,
				Name:        ch.Name,
				Type:        EntityCharacter,
				Description: fmt.Sprintf("%s [%s] %s %s", ch.RoleType, ch.Gender, ch.Personality, ch.Background),
				Properties: map[string]string{
					"role_type":   ch.RoleType,
					"gender":      ch.Gender,
					"age":         ch.Age,
					"status":      ch.Status,
					"appearance":  ch.Appearance,
					"motivation":  ch.Motivation,
					"arc":         ch.Arc,
				},
			})
		}
		for _, org := range chars.Organizations {
			db.upsertEntity(Entity{
				ID:          org.ID,
				Name:        org.Name,
				Type:        EntityOrganization,
				Description: org.Description,
				Properties: map[string]string{
					"type":        org.Type,
					"power_level": org.PowerLevel,
					"location":    org.Location,
					"motto":       org.Motto,
				},
			})
		}
	}

	// 从 Lorebook 同步地点/物品/概念
	lorebook, err := pm.ReadLorebook()
	if err == nil && lorebook != nil {
		for _, entry := range lorebook.Entries {
			var etype EntityType
			switch entry.Category {
			case "location":
				etype = EntityLocation
			case "item":
				etype = EntityItem
			case "concept":
				etype = EntityConcept
			default:
				etype = EntityConcept
			}
			db.upsertEntity(Entity{
				ID:          "lorebook:" + entry.Key,
				Name:        entry.Key,
				Type:        etype,
				Description: entry.Content,
			})
		}
	}

	return nil
}

// upsertEntity 插入或更新实体
func (db *EntityDB) upsertEntity(e Entity) {
	for i, existing := range db.Entities {
		if existing.ID == e.ID {
			// 保留 ChapterRefs
			e.ChapterRefs = existing.ChapterRefs
			db.Entities[i] = e
			return
		}
	}
	db.Entities = append(db.Entities, e)
}

// Query 查询实体
func (db *EntityDB) Query(entityType EntityType) []Entity {
	var result []Entity
	for _, e := range db.Entities {
		if entityType == "" || e.Type == entityType {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetByName 按名称查找实体
func (db *EntityDB) GetByName(name string) *Entity {
	for i, e := range db.Entities {
		if strings.EqualFold(e.Name, name) {
			return &db.Entities[i]
		}
	}
	return nil
}

// GetByID 按 ID 查找实体
func (db *EntityDB) GetByID(id string) *Entity {
	for i, e := range db.Entities {
		if e.ID == id {
			return &db.Entities[i]
		}
	}
	return nil
}

// AddRelation 添加关系
func (db *EntityDB) AddRelation(r EntityRelation) {
	db.Relations = append(db.Relations, r)
}

// GetRelations 获取实体的所有关系
func (db *EntityDB) GetRelations(entityID string) []EntityRelation {
	var result []EntityRelation
	for _, r := range db.Relations {
		if r.FromID == entityID || r.ToID == entityID {
			result = append(result, r)
		}
	}
	return result
}

// AllEntityNames 获取所有实体名（用于链接检测）
func (db *EntityDB) AllEntityNames() []string {
	var names []string
	for _, e := range db.Entities {
		names = append(names, e.Name)
	}
	return names
}
