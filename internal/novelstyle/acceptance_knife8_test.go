package novelstyle

import (
	"math"
	"strings"
	"testing"
	"time"
)

// ── 刀8 验收·线A：确定性 AI 味检测性能（<1s、无模型）+ 词级去味改写保真 ──

// knife8ParaTarget 每段拼装目标 rune 数（每段 ≥5000，三段合计一章规模）。
const knife8ParaTarget = 5200

// knife8NarrPool 叙事句池：句长参差（3~38 rune），标点混用（。？！……；，）。
var knife8NarrPool = []string{
	"夜航船靠岸的时候，江面上起了一层薄雾。",
	"灯灭了。",
	"老周头把缆绳在木桩上绕了三圈，打了个死结，又拽了拽，确认它不会再松。",
	"他没说话。",
	"岸上有人喊他的小名，声音穿过雾落到水面上又弹起来，像是回来了两次。",
	"雨点先是一滴一滴地敲船篷，后来就连成了片。",
	"灶膛里的火还温着，锅盖缝里冒出一丝白气。",
	"她数着楼梯的台阶，数到第七级的时候停住了。",
	"门轴响。",
	"巷子深处传来打铁的声音，一下，又一下，不紧不慢，敲得整条街都醒着。",
	"猫从墙头跳下去，落地没有声音。",
	"他把手里的信折了又折，折到不能再小，才塞进贴身的口袋。",
	"茶凉了。",
	"对岸的灯火一盏一盏地熄，最后只剩航标灯还在江心亮着，红一下，绿一下。",
	"她想起小时候也是这样的夜晚，母亲坐在灯下纳鞋底，线绳穿过布面，发出细密的声响。",
	"风大了。",
	"院子里那棵老槐树被吹得东倒西歪，树叶哗哗地响，像是谁在暗处翻一本很旧的书。",
	"他蹲在门槛上抽完最后一口烟，把烟头摁进泥里。",
	"更夫敲过三更，脚步声由远及近，又由近及远。",
	"水缸结了一层薄冰，指尖一碰就裂出纹路。",
	"她把针在头发里别了别，凑近油灯，继续缝那件洗得发白的褂子。",
	"狗叫了两声，又安静下去。",
	"山道上的雪被人踩实了，走上去咯吱咯吱地响。",
	"他挑着担子进了城门，城门洞里的风比外面冷得多，吹得人骨头缝里发酸。",
	"账房的算盘声停了。",
	"月亮升到正头顶的时候，院子里亮得能看见地上的蚂蚁。",
	"她把晒好的萝卜干收进竹匾，盖上纱布，压上一块洗干净的青砖。",
	"船工们光着膀子卸货，号子声一声压着一声，压过了江水的声音。",
	"他病了三天，第四天能下地的时候，院子里的草已经长到了膝盖。",
	"粥熬稠了，米油浮在最上面一层。",
	"远处传来火车的汽笛，很长的一声，拖过整个镇子的上空，又慢慢散掉。",
	"他张了张嘴，到底什么也没说……",
	"雨停了；风还没停。",
}

// knife8DialogPool 对话句池（引号直接引语，部分带叙述贴附）。
var knife8DialogPool = []string{
	"「你回来啦？」她隔着院子问，手里的活计没停。",
	"「吃了没？」他把碗推过去，「锅里还有。」",
	"「别等我了。」他说完就把门带上了。",
	"「这条路走不得，」老人摆摆手，「下雨天山体要塌。」",
	"「多少钱？」「不要钱，顺路。」",
	"「你爹走的时候，没受罪。」她轻声说。",
	"「明年开春，我把西屋翻一翻。」他望着房梁说。",
	"「小声点，孩子睡了。」",
	"「我不去。」她说得很轻，但没有一点商量的余地。",
	"「你等等我！」他在后面喊，鞋都跑掉了一只。",
	"「都是过去的事了。」她笑了笑，把话岔开了。",
	"「灯还留着呢，」他说，「怕你摸黑。」",
}

// knife8BuildPara 从叙事池 offset 位起轮转拼装一段 ≥target rune 的正文：
// 每 3 句叙事插 1 句对话；三段 offset 错开，同句复现间隔一个池长，避免相邻机械重复。
func knife8BuildPara(offset, target int) string {
	var b strings.Builder
	n := 0
	for i := 0; n < target; i++ {
		s := knife8NarrPool[(offset+i)%len(knife8NarrPool)]
		b.WriteString(s)
		n += len([]rune(s))
		if i%3 == 2 {
			d := knife8DialogPool[(offset+i)%len(knife8DialogPool)]
			b.WriteString(d)
			n += len([]rune(d))
		}
	}
	return b.String()
}

// TestKnife8_AiTasteScorer_PerfUnder1s 三段 ≥5000 rune 真实感章节样本逐段打分：
// 验收口径「AI 味检测<1s、确定性、无模型」——单次调用耗时须 ≤1s，分数落在 [0,100]。
func TestKnife8_AiTasteScorer_PerfUnder1s(t *testing.T) {
	for i := 0; i < 3; i++ {
		para := knife8BuildPara(i*11, knife8ParaTarget)
		n := len([]rune(para))
		if n < 5000 {
			t.Fatalf("段[%d] 长度不足: %d rune（须 ≥5000）", i, n)
		}
		start := time.Now()
		ts, err := ScoreTextNoRef(para)
		cost := time.Since(start)
		if err != nil {
			t.Fatalf("段[%d] ScoreTextNoRef err: %v", i, err)
		}
		if ts == nil {
			t.Fatalf("段[%d] 返回 nil", i)
		}
		if ts.Score < 0 || ts.Score > 100 {
			t.Errorf("段[%d] Score 应在 [0,100], got %d", i, ts.Score)
		}
		t.Logf("段[%d] runes=%d score=%d cost=%s", i, n, ts.Score, cost)
		if cost > time.Second {
			t.Errorf("段[%d] 打分耗时 %s 超过 1s（验收口径：AI 味检测<1s）", i, cost)
		}
	}
}

// knife8AiFidelityText 去味保真样本：含两个实体名（裴照/沈青梧）与一段对话，
// 确定性命中多族规则——黑名单（缓缓/眼帘/轻叹/嘴角勾起/仿佛/须臾/颔首）、
// 情绪直述（内心充满了）、句首连接词连用（然而×2）；词级替换治得了前两梯队的
// 词汇味，治不了句式套路——after 分数应降但不必归零。
const knife8AiFidelityText = `裴照推开祠堂的旧木门，尘埃在光柱里缓缓浮沉，香炉里的灰早已冷透。

沈青梧站在供桌旁，眼帘低垂，手里攥着半块玉佩。「你到底还是回来了。」她轻叹一声。

裴照颔首，嘴角勾起一抹若有若无的弧度，仿佛一切都不曾改变。「我只是路过。」

「路过？」沈青梧抬起头，声音发颤，「三年了，你路过得可真久。」然而，她没有再问。然而，她转身把门关上了。

裴照沉默了片刻，须臾，才缓缓开口：「家里的事，我来办。你不必再等。」

夜风穿堂而过，烛火明明灭灭。沈青梧仿佛没有听见，内心充满了说不出的酸楚，只是把玉佩攥得更紧，指节泛白。`

// TestKnife8_DeSlopRewrite_Fidelity 词级去味保真：分数严格下降、rune 篇幅漂移
// ≤10%（内置表为等长/近等长替换，允许词表演进）、实体名逐字保留、句数不减少
// （与打分同一 sentenceRanges 分句口径：词级替换不动句读标点）。
func TestKnife8_DeSlopRewrite_Fidelity(t *testing.T) {
	in := knife8AiFidelityText
	if !strings.Contains(in, "「") {
		t.Fatal("样本前置失败：未含对话")
	}
	before, err := ScoreTextNoRef(in)
	if err != nil {
		t.Fatalf("ScoreTextNoRef(before) err: %v", err)
	}
	if before.Score <= 0 {
		t.Fatalf("样本应为正分 AI 味, got %d", before.Score)
	}
	for _, kw := range []string{"黑名单", "show-don't-tell", "连接词"} {
		if !hasReason(before, kw) {
			t.Fatalf("样本未命中规则族: %s", kw)
		}
	}

	out, rep, err := DeSlopRewrite(in, before)
	if err != nil {
		t.Fatalf("DeSlopRewrite err: %v", err)
	}
	after, err := ScoreTextNoRef(out)
	if err != nil {
		t.Fatalf("ScoreTextNoRef(after) err: %v", err)
	}
	if len(rep.Changes) == 0 {
		t.Fatal("应有至少一个词级改写条目")
	}

	// ① 分数严格下降；传入 before 分与空白名单下报告分应与复测分一致。
	if after.Score >= before.Score {
		t.Errorf("分数应严格下降: before=%d after=%d", before.Score, after.Score)
	}
	if rep.BeforeScore != before.Score {
		t.Errorf("报告前分应与传入一致: rep=%d got=%d", rep.BeforeScore, before.Score)
	}
	if rep.AfterScore != after.Score {
		t.Errorf("报告后分应与复测分一致: rep=%d rescore=%d", rep.AfterScore, after.Score)
	}

	// ② rune 长度漂移 ≤10%（篇幅不变）。
	inR, outR := len([]rune(in)), len([]rune(out))
	if d := math.Abs(float64(outR-inR)) / float64(inR); d > 0.10 {
		t.Errorf("篇幅漂移超限: %d -> %d rune (%.1f%%)", inR, outR, d*100)
	}

	// ③ 两个实体名（两字/三字专名）逐字保留，出现次数不减。
	for _, name := range []string{"裴照", "沈青梧"} {
		if !strings.Contains(out, name) {
			t.Errorf("实体名 %s 未保留: %s", name, out)
		}
		if got, want := strings.Count(out, name), strings.Count(in, name); got < want {
			t.Errorf("实体名 %s 出现次数减少: %d -> %d", name, want, got)
		}
	}

	// ④ 句数不减少：sentenceRanges 按 。！？…（含 ASCII .!?）分句，词级替换不触碰。
	nb, ns := len(sentenceRanges([]rune(in))), len(sentenceRanges([]rune(out)))
	if ns < nb {
		t.Errorf("句数减少: %d -> %d", nb, ns)
	}

	// 残余命中的规则族（词级替换只治词汇味，句式类问题应存续）。
	for _, iss := range after.Issues {
		t.Logf("after 残余: [%s] %s", iss.Severity, iss.Reason)
	}
	t.Logf("before=%d after=%d changes=%d punctFixed=%d runes=%d->%d sents=%d->%d",
		before.Score, after.Score, len(rep.Changes), rep.PunctFixed, inR, outR, nb, ns)
}
