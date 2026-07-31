package app

import (
	"fmt"

	"github.com/gaea/gaea/internal/graph"
)

// ── 知识图谱 API ────────────────────────────────────────────

// BuildBacklinkIndex 构建全项目反向链接索引
func (a *App) BuildBacklinkIndex() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	idx, err := graph.BuildBacklinkIndex(pm)
	if err != nil {
		return nil, err
	}

	// 转换为可序列化格式
	result := make(map[string]interface{})
	for entity, links := range idx {
		var linkMaps []map[string]interface{}
		for _, l := range links {
			linkMaps = append(linkMaps, map[string]interface{}{
				"target":      l.Target,
				"source_file": l.SourceFile,
				"line_number": l.LineNumber,
				"context":     l.Context,
			})
		}
		result[entity] = linkMaps
	}

	return map[string]interface{}{
		"index":       result,
		"total_links": len(result),
	}, nil
}

// GetBacklinks 获取指定实体的反向链接
func (a *App) GetBacklinks(entityName string) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	idx, err := graph.BuildBacklinkIndex(pm)
	if err != nil {
		return nil, err
	}

	links := idx.GetBacklinks(entityName)
	var result []map[string]interface{}
	for _, l := range links {
		result = append(result, map[string]interface{}{
			"target":      l.Target,
			"source_file": l.SourceFile,
			"line_number": l.LineNumber,
			"context":     l.Context,
		})
	}
	return result, nil
}

// ParseLinks 解析文本中的 [[wiki-link]]
func (a *App) ParseLinks(content string) []string {
	return graph.ParseLinks(content)
}

// FindUnlinkedMentions 查找文本中未链接的实体名
func (a *App) FindUnlinkedMentions(content string, entityNames []string) []string {
	return graph.FindUnlinkedMentions(content, entityNames)
}

// SyncEntityDB 同步实体数据库
func (a *App) SyncEntityDB() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	db, err := graph.LoadEntityDB(pm.Dir)
	if err != nil {
		return nil, err
	}
	if err := db.SyncFromProject(pm); err != nil {
		return nil, err
	}
	if err := db.Save(pm.Dir); err != nil {
		return nil, err
	}

	all := db.Query("")
	return map[string]interface{}{
		"total":       len(all),
		"characters":  len(db.Query(graph.EntityCharacter)),
		"locations":   len(db.Query(graph.EntityLocation)),
		"items":       len(db.Query(graph.EntityItem)),
		"events":      len(db.Query(graph.EntityEvent)),
		"concepts":    len(db.Query(graph.EntityConcept)),
		"organizations": len(db.Query(graph.EntityOrganization)),
	}, nil
}

// QueryEntities 查询实体
func (a *App) QueryEntities(entityType string) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	db, err := graph.LoadEntityDB(pm.Dir)
	if err != nil {
		return nil, err
	}

	var entities []graph.Entity
	if entityType == "" {
		entities = db.Query("")
	} else {
		entities = db.Query(graph.EntityType(entityType))
	}

	var result []map[string]interface{}
	for _, e := range entities {
		result = append(result, map[string]interface{}{
			"id":           e.ID,
			"name":         e.Name,
			"type":         string(e.Type),
			"description":  e.Description,
			"properties":   e.Properties,
			"chapter_refs": e.ChapterRefs,
		})
	}
	return result, nil
}

// CheckConsistency 运行一致性检查
func (a *App) CheckConsistency() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	report, err := graph.CheckConsistency(pm)
	if err != nil {
		return nil, err
	}

	var issues []map[string]interface{}
	for _, issue := range report.Issues {
		issues = append(issues, map[string]interface{}{
			"severity":    issue.Severity,
			"category":    issue.Category,
			"entity_name": issue.EntityName,
			"description": issue.Description,
			"location":    issue.Location,
			"evidence":    issue.Evidence,
			"suggestion":  issue.Suggestion,
		})
	}

	return map[string]interface{}{
		"issues":       issues,
		"total_issues": report.TotalIssues,
		"summary":      report.Summary,
	}, nil
}

// GetEntityRelations 获取实体关系图数据
func (a *App) GetEntityRelations() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	db, err := graph.LoadEntityDB(pm.Dir)
	if err != nil {
		return nil, err
	}
	if err := db.SyncFromProject(pm); err != nil {
		return nil, err
	}

	// 构建节点列表
	var nodes []map[string]interface{}
	for _, e := range db.Query("") {
		nodes = append(nodes, map[string]interface{}{
			"id":    e.ID,
			"name":  e.Name,
			"type":  string(e.Type),
			"group": entityTypeToGroup(e.Type),
		})
	}

	// 构建边列表
	var edges []map[string]interface{}
	for _, r := range db.Relations {
		edges = append(edges, map[string]interface{}{
			"from":  r.FromID,
			"to":    r.ToID,
			"type":  r.Type,
			"label": r.Label,
		})
	}

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}, nil
}

func entityTypeToGroup(t graph.EntityType) int {
	switch t {
	case graph.EntityCharacter:
		return 0
	case graph.EntityOrganization:
		return 1
	case graph.EntityLocation:
		return 2
	case graph.EntityItem:
		return 3
	case graph.EntityEvent:
		return 4
	default:
		return 5
	}
}
