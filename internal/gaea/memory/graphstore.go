package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// ── 图谱物化与遍历（阶段五 5.1）────────────────────────────────────
//
// mem_graph_nodes/edges/meta（db SchemaV18）是 ProjectEvents 投影的物化落点，
// 面向「读多写零」：写路径只追加日志；物化在读取前按水位对账懒重建——
// meta.max_seq 落后于日志（或图空/表被删）时整体重投影。因此「删库可重建」
// 不需要专门命令：DROP 掉 mem_graph_* 再读一次即自愈。

// RebuildGraph 从事件日志整体重投影并物化（先清 mem_graph_* 三表再写），
// 返回节点数。日志不可用（DB nil）时返回 0, nil（调用方按无图处理）。
func RebuildGraph(db *sql.DB) (int, error) {
	if db == nil {
		return 0, nil
	}
	events, err := (&EventLog{DB: db}).LoadEvents()
	if err != nil {
		return 0, fmt.Errorf("load events: %w", err)
	}
	g := ProjectEvents(events)
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM mem_graph_nodes`); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM mem_graph_edges`); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM mem_graph_meta`); err != nil {
		return 0, err
	}
	for _, n := range g.Nodes {
		tags := "[]"
		if len(n.Tags) > 0 {
			if b, err := json.Marshal(n.Tags); err == nil {
				tags = string(b)
			}
		}
		if _, err := tx.Exec(`
INSERT INTO mem_graph_nodes(id, ntype, name, desc, weight, state, kind, mtype, tags, space, project, embedding)
VALUES(?,?,?,?,?,?,?,?,?,?,?,NULL)`,
			n.ID, n.NType, n.Name, n.Desc, n.Weight, n.State, n.Kind, n.MemType, tags, n.Space, n.Project); err != nil {
			return 0, err
		}
	}
	for _, e := range g.Edges {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO mem_graph_edges(src, tgt, etype) VALUES(?,?,?)`,
			e.Src, e.Tgt, e.EType); err != nil {
			return 0, err
		}
	}
	dangling := "[]"
	if b, err := json.Marshal(g.Dangling); err == nil {
		dangling = string(b)
	}
	for _, kv := range [][2]string{
		{"max_seq", fmt.Sprintf("%d", g.MaxSeq)},
		{"dangling", dangling},
	} {
		if _, err := tx.Exec(`INSERT INTO mem_graph_meta(key, value) VALUES(?,?)`, kv[0], kv[1]); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(g.Nodes), nil
}

// EnsureGraphFresh 按水位对账：物化缺失或落后于日志（有未投影事件）时重建。
// 返回是否发生了重建。对账是两次小查询（MAX(seq) vs meta），图新时近零开销。
func EnsureGraphFresh(db *sql.DB) (bool, error) {
	if db == nil {
		return false, nil
	}
	maxLog, err := (&EventLog{DB: db}).MaxEventSeq()
	if err != nil {
		return false, err
	}
	var maxProjected sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(CAST(value AS INTEGER)) FROM mem_graph_meta WHERE key='max_seq'`).Scan(&maxProjected); err != nil {
		return false, err
	}
	// 图空而日志有事件 / 水位落后 → 重建。日志为空且图空是合法初态，不动。
	if maxLog == 0 && !maxProjected.Valid {
		return false, nil
	}
	if maxProjected.Valid && maxProjected.Int64 >= maxLog {
		return false, nil
	}
	_, err = RebuildGraph(db)
	return true, err
}

// MaterializedGraph 读取物化图谱（节点按 id、边按 (src,tgt,etype) 排序——
// 写入序即排序序，读出直接稳定）。悬空列表从 meta 读回。调用前应 EnsureGraphFresh。
func MaterializedGraph(db *sql.DB) (*Graph, error) {
	g := &Graph{}
	if db == nil {
		return g, nil
	}
	rows, err := db.Query(`SELECT id, ntype, name, desc, weight, state, kind, mtype, tags, space, project FROM mem_graph_nodes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n GraphNode
		var tags string
		if err := rows.Scan(&n.ID, &n.NType, &n.Name, &n.Desc, &n.Weight, &n.State, &n.Kind, &n.MemType, &tags, &n.Space, &n.Project); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(tags), &n.Tags)
		if len(n.Tags) == 0 {
			n.Tags = nil // 与投影输出的 nil 口径一致（可重建性 DeepEqual 依赖）
		}
		g.Nodes = append(g.Nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	erows, err := db.Query(`SELECT src, tgt, etype FROM mem_graph_edges ORDER BY src, tgt, etype`)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var e GraphEdge
		if err := erows.Scan(&e.Src, &e.Tgt, &e.EType); err != nil {
			continue
		}
		g.Edges = append(g.Edges, e)
	}
	if err := erows.Err(); err != nil {
		return nil, err
	}
	var dangling string
	if err := db.QueryRow(`SELECT value FROM mem_graph_meta WHERE key='dangling'`).Scan(&dangling); err == nil {
		_ = json.Unmarshal([]byte(dangling), &g.Dangling)
		if len(g.Dangling) == 0 {
			g.Dangling = nil // 与投影输出的 nil 口径一致（可重建性 DeepEqual 依赖）
		}
	}
	var maxSeq sql.NullInt64
	if err := db.QueryRow(`SELECT CAST(value AS INTEGER) FROM mem_graph_meta WHERE key='max_seq'`).Scan(&maxSeq); err == nil {
		g.MaxSeq = maxSeq.Int64
	}
	return g, nil
}

// Neighbor 是图遍历的一跳命中（记忆互引可图遍历的行形态）。
type Neighbor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	NType string `json:"ntype"`
	State string `json:"state,omitempty"`
	Hops  int    `json:"hops"` // 距起点的跳数（起点=0）
	Via   string `json:"via"`  // 最后一条边的类型（produces/affects/references）
}

// GraphNeighbors 从 start（节点 id、[MEM:name] 记忆键或显示名）出发做 BFS，
// 返回 hops 跳内的可达节点（不含起点）。解析顺序：节点 id 精确 → 显示名 →
// 实体 id 后缀（mem:<project>/<name> 的 /<name> 段——[MEM:name] 寻址不知道
// project 段，后缀匹配兜底）。互引遍历边双向可走；hops<=0 视为 1。
func GraphNeighbors(db *sql.DB, start string, hops int) []Neighbor {
	if db == nil || strings.TrimSpace(start) == "" {
		return nil
	}
	if hops <= 0 {
		hops = 1
	}
	g, err := MaterializedGraph(db)
	if err != nil || len(g.Nodes) == 0 {
		return nil
	}
	index := make(map[string]int, len(g.Nodes))
	byName := make(map[string]string, len(g.Nodes))
	for i, n := range g.Nodes {
		index[n.ID] = i
		byName[n.Name] = n.ID
	}
	startID := start
	if _, ok := index[startID]; !ok {
		if id, ok := byName[start]; ok {
			startID = id
		} else {
			// [MEM:name] 键：实体 id 后缀（跨项目同名取字典序第一个，稳定）。
			suffix := "/" + start
			found := ""
			for _, n := range g.Nodes {
				if n.NType == NodeEntity && strings.HasSuffix(n.ID, suffix) && (found == "" || n.ID < found) {
					found = n.ID
				}
			}
			if found == "" {
				return nil
			}
			startID = found
		}
	}
	// 邻接表（边双向可走：互引/触达是语义关系，遍历不关心方向）。
	adj := map[string][]struct {
		to   string
		edge string
	}{}
	for _, e := range g.Edges {
		adj[e.Src] = append(adj[e.Src], struct {
			to   string
			edge string
		}{e.Tgt, e.EType})
		adj[e.Tgt] = append(adj[e.Tgt], struct {
			to   string
			edge string
		}{e.Src, e.EType})
	}
	type queued struct {
		id  string
		hop int
		via string
	}
	frontier := []queued{{id: startID, hop: 0}}
	seen := map[string]bool{startID: true}
	var out []Neighbor
	for len(frontier) > 0 {
		cur := frontier[0]
		frontier = frontier[1:]
		if cur.hop >= hops {
			continue
		}
		for _, nb := range adj[cur.id] {
			if seen[nb.to] {
				continue
			}
			seen[nb.to] = true
			n := g.Nodes[index[nb.to]]
			out = append(out, Neighbor{
				ID: n.ID, Name: n.Name, NType: n.NType, State: n.State,
				Hops: cur.hop + 1, Via: nb.edge,
			})
			frontier = append(frontier, queued{id: nb.to, hop: cur.hop + 1})
		}
	}
	return out
}

// GraphStats 是图谱规模速览（总览卡/图页统计行）。
type GraphStats struct {
	Events   int64    `json:"events"`   // 日志事件总数
	Nodes    int      `json:"nodes"`    // 实体节点数（物化）
	Edges    int      `json:"edges"`    // 边数（物化）
	Dangling []string `json:"dangling"` // 悬空互引目标
}

// GraphMaterializationStats 汇总日志与物化两侧规模（不做对账，对账走
// EnsureGraphFresh）。
func GraphMaterializationStats(db *sql.DB) GraphStats {
	st := GraphStats{Dangling: []string{}}
	if db == nil {
		return st
	}
	if n, err := (&EventLog{DB: db}).MaxEventSeq(); err == nil {
		st.Events = n
	}
	var nodes, edges int
	if err := db.QueryRow(`SELECT COUNT(*) FROM mem_graph_nodes WHERE ntype='entity'`).Scan(&nodes); err == nil {
		st.Nodes = nodes
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM mem_graph_edges`).Scan(&edges); err == nil {
		st.Edges = edges
	}
	var dangling string
	if err := db.QueryRow(`SELECT value FROM mem_graph_meta WHERE key='dangling'`).Scan(&dangling); err == nil {
		_ = json.Unmarshal([]byte(dangling), &st.Dangling)
	}
	if st.Dangling == nil {
		st.Dangling = []string{}
	}
	return st
}
