package memory

import "sort"

// ── 四臂召回融合（Nebula Memory 蒸馏 P1，docs/gaea-agent-memory-nebula-distill-2026-11.md）──
//
// 四臂（各臂负责把分归一到 [0,1]），权重沿 semantic/fusion.go 三路融合的
// 可解释先例（任何一条命中为什么排这里，都能用分量分与权重复算出来）：
//
//	final = (W_LEX*lexical + W_SEM*semantic + W_GRAPH*graph + pinned*W_PIN)
//	        × dormant × decay
//
//	词法臂 W_LEX  =0.40  BM25 归一分（精确词/型号/专名主力）
//	语义臂 W_SEM  =0.35  向量 cosine（本地 bge-m3；语义变体召回主力）
//	图谱臂 W_GRAPH=0.15  事件图扩散（references 邻居 =「想起 A 带出 B」）
//	置顶加成 W_PIN=0.10  固化条恒进结果（pinned 免疫休眠/衰减惩罚）
//	休眠乘 dormant=0.55  衰减态（LifecycleDecaying）检索惩罚（Nebula dormant）
//	衰减乘 decay  ∈[0.01,1] 稳定性感知新鲜度（RecallDecayFactor，P2）
//
// 权重和=1；final ∈ [0,1]。休眠惩罚乘子（Nebula dormant 0.55 蒸馏）由 P2
// 的访问动力学接线——刀 A 固定传 false，字段先行避免契约二次变更。

// 四臂权重（可解释性单一声明点；调整须同步本注释与测试基线）。
const (
	RecallWeightLexical  = 0.40
	RecallWeightSemantic = 0.35
	RecallWeightGraph    = 0.15
	RecallWeightPinned   = 0.10
)

// GraphArmDecay 是图谱臂的邻居分衰减：邻居分 = 种子分 × 0.5（1 跳）。
const GraphArmDecay = 0.5

// DormantPenalty 是休眠记忆在融合打分里的乘子（Nebula dormant 蒸馏值）。
const DormantPenalty = 0.55

// 臂名归因（canonical 序：词法→语义→图谱→置顶；前端徽章直接消费）。
const (
	RecallArmLexical  = "lexical"
	RecallArmSemantic = "semantic"
	RecallArmGraph    = "graph"
	RecallArmPinned   = "pinned"
)

// RecallArmHit 单臂命中（Name=memory slug，Score 已由装配层归一到 [0,1]）。
type RecallArmHit struct {
	Name  string
	Score float64
}

// RecallFuseInput 单条记忆的四臂分量输入（零值臂=该臂未命中；不在输入里的
// 记忆不参与融合——装配层只送「至少一臂命中或置顶」的条目）。
type RecallFuseInput struct {
	Name     string
	Lexical  float64
	Semantic float64
	Graph    float64
	Pinned   bool
	Dormant  bool // 休眠惩罚乘子开关（衰减态=LifecycleDecaying；pinned 豁免）
	// Decay 稳定性感知新鲜度因子 ∈ [0.01,1]（RecallDecayFactor 装配值）。
	// 契约：装配层恒填（零值=未提供，按 1 兼容直构调用方/旧测试）——合法
	// 计算值下限 0.01，永不合法地为 0。
	Decay float64
}

// RecallFusedHit 融合命中（分量+加权分+臂归因——可解释性的 wire 面）。
type RecallFusedHit struct {
	Name     string
	Lexical  float64
	Semantic float64
	Graph    float64
	Pinned   bool
	Dormant  bool
	Decay    float64
	Arms     []string
	Final    float64
}

// clamp01 把分数夹进 [0,1]（臂分数越界时诚实截断，不放大）。
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// FuseRecallArms 四臂融合+确定性排序（Final 降序 → pinned 优先 → name 字典序）。
// 纯函数：输入即全部事实，零 IO 零时钟。
func FuseRecallArms(in []RecallFuseInput) []RecallFusedHit {
	out := make([]RecallFusedHit, 0, len(in))
	for _, it := range in {
		lex, sem, graph := clamp01(it.Lexical), clamp01(it.Semantic), clamp01(it.Graph)
		base := RecallWeightLexical*lex + RecallWeightSemantic*sem + RecallWeightGraph*graph
		bonus := 0.0
		if it.Pinned {
			bonus = RecallWeightPinned
		}
		penalty := 1.0
		if it.Dormant && !it.Pinned {
			penalty = DormantPenalty
		}
		decay := 1.0
		if it.Decay > 0 {
			decay = clamp01(it.Decay)
			if it.Pinned {
				decay = 1.0 // pinned 免疫衰减（RecallDecayFactor 同口径，双保险）
			}
		}
		arms := make([]string, 0, 4)
		if lex > 0 {
			arms = append(arms, RecallArmLexical)
		}
		if sem > 0 {
			arms = append(arms, RecallArmSemantic)
		}
		if graph > 0 {
			arms = append(arms, RecallArmGraph)
		}
		if it.Pinned {
			arms = append(arms, RecallArmPinned)
		}
		out = append(out, RecallFusedHit{
			Name:     it.Name,
			Lexical:  lex,
			Semantic: sem,
			Graph:    graph,
			Pinned:   it.Pinned,
			Dormant:  it.Dormant,
			Decay:    decay,
			Arms:     arms,
			Final:    (base + bonus) * penalty * decay,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Final != b.Final {
			return a.Final > b.Final
		}
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		return a.Name < b.Name
	})
	return out
}

// ExpandGraphSeeds 图谱臂扩散：种子（词法/语义命中）沿实体级邻接表走 1 跳，
// 邻居分 = 种子分 × GraphArmDecay，同邻居取最高种子分（确定性）。种子自身与
// 邻接表里不存在的名字不产出。纯函数。
func ExpandGraphSeeds(seeds []RecallArmHit, adj map[string][]string) map[string]float64 {
	out := map[string]float64{}
	seedSet := make(map[string]bool, len(seeds))
	for _, s := range seeds {
		seedSet[s.Name] = true
	}
	for _, s := range seeds {
		for _, nb := range adj[s.Name] {
			if nb == s.Name || seedSet[nb] {
				continue
			}
			v := clamp01(s.Score) * GraphArmDecay
			if v > out[nb] {
				out[nb] = v
			}
		}
	}
	return out
}
