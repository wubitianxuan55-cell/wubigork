package pantheon

// A2a 环形互动检测（2026-11-07，蒸馏自 OpenMausBot room-post-budget.ts 的
// closesRing）。**稳健性池纯函数——当前不接线运行循环**（两场论证结论：
// 源码变体在 10 个历史会话上回放 0 正例，星形误杀风险未排除前不进生产；
// 挂靠 B1 试点随行采集 would-fire，届时只接线不重写）。
//
// 语义（对齐源码直证 + 论证冻结的测试向量）：
//   - 边 = DiscussionAct ∈ {challenge, respond} 且 TargetContributionID 能
//     定位到一条**他人**贡献（自指边忽略）；
//   - 环 = ≥3 个互异席位首尾相接（A→B→C→A）；
//   - 星形免疫（论证 V5 向量）：多席指向同一 hub（如全员质询主席小结）
//     不构成环——hub 只入度不出度；
//   - 字段缺失 = 无边（insufficient），绝不按措辞猜测补边（宿主归因纪律：
//     主张方只从明确引用归因）。
//
// 返回环上的席位 ID（按环序，首尾相同）；无环时 ok=false。
func DetectInteractionRing(contributions []Contribution) ([]string, bool) {
	owner := make(map[string]string, len(contributions))
	order := []string{} // 席位首现序，保证遍历确定性
	for _, c := range contributions {
		if c.ID == "" || c.AgentID == "" {
			continue
		}
		if _, exists := owner[c.ID]; !exists {
			owner[c.ID] = c.AgentID
		}
	}
	seen := map[string]bool{}
	for _, c := range contributions {
		if !seen[c.AgentID] {
			seen[c.AgentID] = true
			order = append(order, c.AgentID)
		}
	}
	adj := make(map[string][]string, len(order))
	for _, c := range contributions {
		if c.DiscussionAct != DiscussionActChallenge && c.DiscussionAct != DiscussionActRespond {
			continue
		}
		if c.AgentID == "" || c.TargetContributionID == "" {
			continue
		}
		target, exists := owner[c.TargetContributionID]
		if !exists || target == c.AgentID {
			continue // 未定位 = 无边；自指 = 无边。
		}
		adj[c.AgentID] = append(adj[c.AgentID], target)
	}
	// DFS 找 ≥3 互异节点的环；path 即访问栈，环判定=目标已在栈中且栈段长度足够。
	var path []string
	var found []string
	var dfs func(node string) bool
	dfs = func(node string) bool {
		if idx := indexOf(path, node); idx >= 0 {
			if len(path)-idx >= 3 {
				found = append([]string(nil), path[idx:]...)
				return true
			}
			return false // 短环（自环/二元）不判。
		}
		path = append(path, node)
		for _, next := range adj[node] {
			if dfs(next) {
				return true
			}
		}
		path = path[:len(path)-1]
		return false
	}
	for _, id := range order {
		path = path[:0]
		if dfs(id) {
			return found, true
		}
	}
	return nil, false
}

func indexOf(values []string, want string) int {
	for i, v := range values {
		if v == want {
			return i
		}
	}
	return -1
}
