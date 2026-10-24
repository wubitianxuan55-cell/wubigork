package app

import (
	"fmt"
	"sort"
	"strings"
)

// ── 网文技法检索库（webnovel-writer 九表思想机制重推导蒸馏）────────────────
//
// 上游思想：技法知识以「关键词/意图同义词/适用题材/核心摘要/详细展开/毒点」
// 的行式账本存档，按需检索注入不常驻（GPL-3.0 隔离：只重推导机制，内容全自写，
// 不搬上游任何文本）。gaea 口径：
//   - 检索零 LLM：确定性子串计分（关键词/分类名命中，宁漏勿误）；
//   - 生成链注入：章节计划/大纲要点文本命中关键词的条目进上下文（advisory
//     区段，Rank 4 最先牺牲——技法是锦上添花，不是创作约束，排在世界观之后）；
//   - NovelTechniqueSearch 绑定：作者手动检索（建议制只读，零落盘，不写任何
//     项目文件）。
//
// v1 边界：内置账本只读，不支持项目级增补（写路径属记忆写入确认红线范畴，
// 实弹有需求再立刀）。

// novelTechnique 单条技法（九类：场景写法/爽点节奏/桥段套路/人设关系/写作技法/
// 命名规则/裁决规则/金手指/题材调性）。
type novelTechnique struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Keywords []string `json:"keywords"` // 检索词（含意图同义词；2+ 字子串匹配）
	Genres   []string `json:"genres"`   // 适用题材；空=通用
	Summary  string   `json:"summary"`  // 一句话核心
	Detail   string   `json:"detail"`   // 展开指令（怎么写）
	Pitfall  string   `json:"pitfall"`  // 毒点（这样写会坏）
}

// novelTechniques 内置技法账本（内容自写；ID 稳定——注入与检索的排序锚）。
var novelTechniques = []novelTechnique{
	// ── 场景写法 ──
	{ID: "scene-battle", Category: "场景写法", Keywords: []string{"打斗", "战斗", "交手", "厮杀", "对决", "武斗", "打架"},
		Summary: "打斗写「换招得失」不写招式流水账",
		Detail:  "每个回合要有攻防得失——谁占上风、付出了什么代价；动作之间夹一句心理或环境变化让节奏透气；胜负手要能回溯到前文铺垫的能力或道具，不凭空翻盘。",
		Pitfall: "招式名词堆砌、从头打到尾没有任何代价和转折，读者会跳读。"},
	{ID: "scene-confession", Category: "场景写法", Keywords: []string{"告白", "表白", "摊牌", "示爱", "情愫", "心意"},
		Summary: "情感摊牌写「时机错位」比写「言语完美」更动人",
		Detail:  "让告白发生在压力或误解未解开的时刻，话没说完被打断或答非所问，都好过顺畅圆满；用动作细节（攥紧的衣角、移开的视线）代替心情形容。",
		Pitfall: "双方情绪一步到位、你情我愿零波折，是情感戏最平的写法。"},
	{ID: "scene-banquet", Category: "场景写法", Keywords: []string{"宴会", "群戏", "聚会", "酒席", "宴席", "庆典"},
		Summary: "群戏只给两三个焦点人物镜头，其余当背景板",
		Detail:  "用主角视点扫全场，选两三个有戏份的人给动作和台词，其他人以整体噪声存在；宴会的核心是桌下博弈——使眼色、敬酒顺序、座次，不是菜色描写。",
		Pitfall: "每个出场人物都给一段介绍，节奏当场垮掉。"},
	{ID: "scene-chase", Category: "场景写法", Keywords: []string{"追逐", "逃亡", "追杀", "逃跑", "突围"},
		Summary: "逃亡戏写「代价递增」——每逃过一关丢一样东西",
		Detail:  "追兵步步逼近的压力线持续在线，用短句加快节奏；每关脱身都付出可见代价（受伤/丢物资/暴露底牌），最后一关的脱身手段必须是前文埋过的。",
		Pitfall: "主角靠地形巧合连续白嫖脱身，紧张感清零。"},

	// ── 爽点节奏 ──
	{ID: "payoff-face-slap", Category: "爽点节奏", Keywords: []string{"打脸", "扬眉吐气", "报仇", "雪耻", "轻视", "羞辱", "退婚"},
		Summary: "打脸三拍：先压后扬再踩实，压制期越具体打脸越爽",
		Detail:  "先让对手的轻视落在具体事件（退婚/夺功/当众羞辱），主角隐忍积累，爆发点选公开场合让旁观者见证，打完留一句戳破对手逻辑的话收尾。",
		Pitfall: "对手纯蠢坏无脑嚣张、压制期一笔带过直接打脸，爽感没有落差。"},
	{ID: "payoff-levelup", Category: "爽点节奏", Keywords: []string{"升级", "突破", "晋阶", "进阶", "修为", "境界"},
		Summary: "升级要有「卡关—破关代价—新能力验证」三件套",
		Detail:  "突破前先写瓶颈的具体表现（招式走形/旧法失灵），破关要消耗点什么（资源/寿元/机缘），升级后新能力先小规模验证再上强度。",
		Pitfall: "无痛连升、数值通胀、新能力当场满级，后期就写不动了。"},
	{ID: "payoff-treasure", Category: "爽点节奏", Keywords: []string{"夺宝", "机缘", "奇遇", "宝物", "捡漏", "秘境"},
		Summary: "机缘要「挣来」——先立资格门槛再给奖励",
		Detail:  "宝物前设考验（实力/心性/前文伏笔），让主角的既有积累成为拿到它的理由；同一场景可以给对手一条更轻松的贪路径并让其翻车，反衬主角路径。",
		Pitfall: "天降馅饼白捡、主角躺着获得奖励，读者不认账。"},
	{ID: "payoff-identity", Category: "爽点节奏", Keywords: []string{"身份揭晓", "身世", "揭露身份", "马甲", "真面目", "认出"},
		Summary: "身份揭晓前铺三处可回查的暗线，揭晓要直接改变力量关系",
		Detail:  "暗线用语气习惯/随身物/本能反应这类具体物；揭晓场合选利益攸关方在场，让揭晓带来实际的利益重排而不只是众人惊呼。",
		Pitfall: "无铺垫突然掏身份、揭晓后没有实际后果，就是空响。"},

	// ── 桥段套路 ──
	{ID: "trope-tournament", Category: "桥段套路", Keywords: []string{"比武", "擂台", "大赛", "武道会", "选拔", "竞技"},
		Summary: "大会戏的看点在场外，赛场内只打关键三场",
		Detail:  "三场=遭遇战（露一手）、宿敌战（旧怨清算）、决赛（留悬念或被搅局）；穿插看台上各方势力的反应与暗中下注来推进主线。",
		Pitfall: "场场全写、对手都是工具人轮番送，是中期节奏毒药。"},
	{ID: "trope-misunderstand", Category: "桥段套路", Keywords: []string{"误会", "误解", "错怪", "信息差"},
		Summary: "误会要双方各有正当理由，且限章内解决或明确挂账",
		Detail:  "误会源于信息差且双方都按合理逻辑行动；给读者全知视角看两人错位；解决要么快（当章说开）要么立期限（何时必须摊牌），不许无限拖延。",
		Pitfall: "一句话能说清的事硬拖十章、角色降智不沟通，是弃书点。"},
	{ID: "trope-mentor", Category: "桥段套路", Keywords: []string{"师父", "师傅", "师徒", "引路人", "导师", "前辈"},
		Summary: "师父要有私心或有残缺，不给全能保姆",
		Detail:  "师父的教导留一手或带条件（门派恩怨/自身局限），主角先消化师父给的框架再自行突破；师父退场安排在主角能自立之后，退场前把未了恩怨转交。",
		Pitfall: "师父全程兜底、主角遇事喊师父，主角就立不住。"},
	{ID: "trope-enemy-alliance", Category: "桥段套路", Keywords: []string{"结盟", "联手", "暂时合作", "敌人的敌人", "共敌"},
		Summary: "与敌结盟写「共同敌人到期日」，先小人后君子",
		Detail:  "结盟时把分歧和期限摆上桌面（敌人一死各回各路），执行中留一次背刺未遂或救场相助的摇摆；事后再相遇时的态度变化是关系戏的好料。",
		Pitfall: "昨天死敌今天无条件信任、结盟后矛盾全消失，关系失真。"},

	// ── 人设关系 ──
	{ID: "char-villain", Category: "人设关系", Keywords: []string{"反派", "对手", "宿敌", "大反派", "恶人"},
		Summary: "反派按自己的逻辑赢，不按主角的逻辑蠢",
		Detail:  "给反派一条自洽的动机链（诉求→手段→底线），让他至少赢主角一次半（拿走东西/逼出代价）；反派的高光时刻让读者短暂站他，落幕补一笔人性余味。",
		Pitfall: "反派全靠手下蠢、输在最后关头话多，就是纸片人。"},
	{ID: "char-support", Category: "人设关系", Keywords: []string{"配角", "次要角色", "功能角色", "队友"},
		Summary: "配角一人一功能一标记，出场即带任务",
		Detail:  "给每个活跃配角一个专属行为标记（口头禅/小动作/视角偏见），功能完成或重复两次后要么给新任务要么降频；配角可以挡一次关键刀，但不能替主角做决定。",
		Pitfall: "配角戏份膨胀抢主线、名字一堆脸谱全糊，读者记不住。"},
	{ID: "char-relation-evolve", Category: "人设关系", Keywords: []string{"关系变化", "破镜", "和好", "决裂", "反目", "和解"},
		Summary: "关系每升一格都要一个事件作台阶，不许无事升温",
		Detail:  "破裂靠利益或原则冲突（不靠误会拖戏），和好靠共同经历新事件而非旧情绑架；关系变化后双方相处细节要跟着变（称呼/距离/默契）。",
		Pitfall: "关系原地反复横跳、上一章决裂下一章喝酒，读者情感账户破产。"},

	// ── 写作技法 ──
	{ID: "craft-open-chapter", Category: "写作技法", Keywords: []string{"开头", "章首", "开篇", "第一段", "起笔"},
		Summary: "章首三行内落「变化中的具体场景」，不写回顾摘要",
		Detail:  "直接切进动作或对话的中段，用一句异常细节示警；上一章的必要衔接压缩成半句带过。",
		Pitfall: "章首复述上章剧情、天气起床流水线，是章末弃读高发点。"},
	{ID: "craft-cliffhanger", Category: "写作技法", Keywords: []string{"章末", "悬念", "断章", "钩子", "下章", "留扣"},
		Summary: "章末停在「信息即将揭晓的前一拍」或「代价刚落地的瞬间」",
		Detail:  "新信息抛出一半（人到了、话没说完）、决定已做后果未至；钩子必须当场给具体由头（一件物/一句话/一个变化），不开空吓天窗。",
		Pitfall: "为断而断、每章末都喊狼来了，三次后读者免疫。"},
	{ID: "craft-foreshadow", Category: "写作技法", Keywords: []string{"伏笔", "铺垫", "埋线", "前照", "回收"},
		Summary: "伏笔埋「可回查的具体物」，回收时读者能翻回找到",
		Detail:  "埋的时候当作正常细节写（有当下功能），回收前在中途提一次加深印象；一处伏笔一处回收，跨卷长线伏笔记账管理（gaea 伏笔面板即此账本）。",
		Pitfall: "回收靠读者失忆、当章埋当章收，不算伏笔算平铺。"},
	{ID: "craft-dialogue", Category: "写作技法", Keywords: []string{"对话", "台词", "对白", "说话"},
		Summary: "对话写「错位与目的」，不写信息传声筒",
		Detail:  "每个说话人带自己的目的与潜台词，答非所问、岔开话题都是真实感来源；台词间用动作/环境间隔，长对白拆场；信息量大的设定不走对话走画面。",
		Pitfall: "两人互相复述双方都知道的事（「正如你所说……」），是说明文披对话皮。"},

	// ── 命名规则 ──
	{ID: "naming-character", Category: "命名规则", Keywords: []string{"取名", "命名", "角色名", "名字", "起名"},
		Summary: "主角团名字避生僻字、避同声母、姓氏错开",
		Detail:  "常同框的角色名字首字和字数错开（两字/三字混用），读音上口可念；反派或势力名可稍怪但不超过一次查字典成本。",
		Pitfall: "名字全两字同姓扎堆、读者靠发型记人，关系戏全糊。"},
	{ID: "naming-power", Category: "命名规则", Keywords: []string{"功法", "招式", "技能名", "命名体系", "武学"},
		Summary: "招式名遵循「流派命名法」一以贯之，不混体系",
		Detail:  "给每个体系定命名规则（数字序/意象族/意境族），同一体系内风格统一、等级感可从名字读出；主角招牌技的名字要能被读者口口相传。",
		Pitfall: "招式名越取越长、中二词堆砌、体系混杂，记不住还降格调。"},

	// ── 裁决规则 ──
	{ID: "canon-power-ceiling", Category: "裁决规则", Keywords: []string{"力量体系", "等级", "天花板", "上限", "战力"},
		Summary: "力量天花板早立标准晚兑现，越顶必须付大代价",
		Detail:  "开篇用旁观者反应或史实侧写顶格强者的量级，主角越级挑战要付明确代价（重伤/底牌/人情）；天花板一旦写过，后文不许无痛刷新。",
		Pitfall: "战力通胀、上限随剧情需要反复上调，是老读者出走点。"},
	{ID: "canon-cost", Category: "裁决规则", Keywords: []string{"代价", "等价交换", "消耗", "透支", "偿还"},
		Summary: "一切越规收益挂账，透支要还",
		Detail:  "越出常规的能力获取都要有可见成本（资源/时间/关系/寿命），透支的账分期还且还的时候要有戏；免费午餐出现一次，规则信用就破产。",
		Pitfall: "主角屡次白嫖超规收益、代价写一句就再不提，金手指失信。"},

	// ── 金手指 ──
	{ID: "cheat-design", Category: "金手指", Keywords: []string{"金手指", "外挂", "系统", "特殊能力", "异能"},
		Summary: "金手指一条主规则加明确限制，不给万能瑞士军刀",
		Detail:  "核心能力一条说清（是什么/怎么触发/上限在哪），限制条件开篇立好（次数/代价/冷却）；成长靠主角用主规则做组合创新，不靠临时发新功能。",
		Pitfall: "金手指按剧情需要随时加新功能，读者对规则失去敬畏。"},
	{ID: "cheat-secret", Category: "金手指", Keywords: []string{"隐瞒身份", "保密", "暴露", "藏拙", "识破"},
		Summary: "金手指保密要有「一次惊险不揭穿」的近失章",
		Detail:  "安排一次差一点被识破的近失（被人看出端倪/临时圆谎），让保密成为持续的戏剧张力；每多一个人知道，写清其守密的理由与风险。",
		Pitfall: "主角当众用挂无人起疑、知情人名单膨胀无代价，悬念漏气。"},

	// ── 题材调性 ──
	{ID: "tone-xianxia", Category: "题材调性", Genres: []string{"仙侠", "玄幻"},
		Keywords: []string{"仙侠", "修仙", "修真", "问道"},
		Summary:  "仙侠的仙味在敬畏感——天地规则大于人",
		Detail:   "等级差距用体感描写（凡人见元婴腿软），机缘与天道挂钩带宿命感，争斗讲究因果与气数；人物说话文白相间不出戏。",
		Pitfall:  "修仙写成都市职场互怼、境界只是数值，仙味全失。"},
	{ID: "tone-urban", Category: "题材调性", Genres: []string{"都市", "现实"},
		Keywords: []string{"都市", "现代", "行业", "商战", "职场"},
		Summary:  "都市文的真实感靠专业细节与规则代价",
		Detail:   "选一个行业深写（医疗/律政/商战），用真实流程细节撑可信度；超规事件要处理社会性后果（监控/舆论/机构）；金手指与现实主义二选一为调性主轴。",
		Pitfall:  "行业细节全靠想象、主角横行无社会摩擦，悬浮感十足。"},
	{ID: "tone-mystery", Category: "题材调性", Genres: []string{"悬疑", "灵异"},
		Keywords: []string{"悬疑", "推理", "解谜", "探案", "诡异"},
		Summary:  "悬疑的公平性：读者与主角拿同样的线索",
		Detail:   "关键线索在揭晓前必须亮过相（可藏在不起眼处），主角推理链每步有据；小悬念章章有、大悬念卷卷一个；恐怖感靠具体感官细节不靠形容词堆。",
		Pitfall:  "揭晓时掏读者没见过的线索、主角靠直觉破案，推理信用破产。"},
}

const (
	// ctxTechniqueBudget 技法提示区段预算（rune，含标题）。
	ctxTechniqueBudget = 520
	// ctxTechniqueMaxItems 注入的最多条目数（宁少勿滥）。
	ctxTechniqueMaxItems = 3
	// ctxTechniqueLineLen 单条目渲染上限（rune；摘要+要点+毒点压成一行）。
	ctxTechniqueLineLen = 170
	// techniqueSearchMax 手动检索返回上限。
	techniqueSearchMax = 20
)

// techniqueMatchScore 条目对文本的命中分（纯函数，确定性）：关键词（含分类名）
// 每命中一个计 1 分（子串包含——中文无词边界，与实体卡触发注入同口径）。
// 短于 2 rune 的关键词不参与匹配（「戏」「光」这类单字误伤面太大）。
func techniqueMatchScore(t novelTechnique, text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	score := 0
	terms := t.Keywords
	if len(t.Category) >= 2 {
		terms = append(append([]string(nil), t.Keywords...), t.Category)
	}
	for _, kw := range terms {
		if len([]rune(kw)) >= 2 && strings.Contains(text, kw) {
			score++
		}
	}
	return score
}

// searchTechniques 检索技法账本（零 LLM）：命中分降序、同分按账本序（ID 稳定）。
// query 为空返回空（面板初始态走分类目录，不倒全量）。
func searchTechniques(query string, limit int) []novelTechnique {
	q := strings.TrimSpace(query)
	if q == "" || limit <= 0 {
		return nil
	}
	type hit struct {
		idx, score int
	}
	var hits []hit
	for i, t := range novelTechniques {
		if s := techniqueMatchScore(t, q); s > 0 {
			hits = append(hits, hit{i, s})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].score > hits[b].score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]novelTechnique, 0, len(hits))
	for _, h := range hits {
		out = append(out, novelTechniques[h.idx])
	}
	return out
}

// techniqueCategories 分类目录（面板初始态：九类名 + 各类条数，账本序）。
func techniqueCategories() []map[string]interface{} {
	var order []string
	counts := map[string]int{}
	for _, t := range novelTechniques {
		if counts[t.Category] == 0 {
			order = append(order, t.Category)
		}
		counts[t.Category]++
	}
	out := make([]map[string]interface{}, 0, len(order))
	for _, c := range order {
		out = append(out, map[string]interface{}{"category": c, "count": counts[c]})
	}
	return out
}

// NovelTechniqueSearch 技法库检索（作者手动查；零 LLM、只读不落盘）。
// 空查询返回空结果 + 分类目录（面板初始态）。
func (a *writingState) NovelTechniqueSearch(query string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"results":    searchTechniques(query, techniqueSearchMax),
		"categories": techniqueCategories(),
		"total":      len(novelTechniques),
	}, nil
}

// buildTechniqueSection 章节生成的技法提示区段（advisory）：hint（本章计划+大纲
// 要点文本）命中关键词的条目注入，至多 3 条、合计不超预算、整条不入宁少勿截半。
// 零命中返回 ""（不占额度）。定位是「参考」——标题明示非创作约束。
func buildTechniqueSection(hint string) string {
	matched := searchTechniques(hint, ctxTechniqueMaxItems)
	if len(matched) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## 本章技法提示（参考——锦上添花，非创作约束；与本章无关的条目忽略）\n")
	used := runeLen(b.String())
	for _, t := range matched {
		line := fmt.Sprintf("- %s｜%s——%s｜毒点：%s", t.Category, t.Summary, t.Detail, t.Pitfall)
		l := runeLen(line)
		if used+l > ctxTechniqueBudget {
			break // 预算耗尽：整条不入（截半的技法指导比没有更糟）
		}
		b.WriteString(truncateBudget(line, ctxTechniqueLineLen) + "\n")
		used += l
	}
	return strings.TrimRight(b.String(), "\n")
}
