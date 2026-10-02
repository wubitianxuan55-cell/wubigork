package app

// 故事层骨架（长篇刀3，规格 进度计划/gaea-novel-story-spine-20260930.md）。
//
// 四个绑定面：SpineGet/SpineSave（整表往返+确定性校验）/StoryHealth（七维度
// 确定性体检，零 LLM）/SpinePropose（AI 提炼骨架，提案不落盘）。
// 生成注入切片 storySpineSection 供 CreateChapter 与逐场景生成流共用
// （空 spine 零注入）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// NovelStorySpineGet 读故事脊椎（文件缺失返回空骨架，不算错）。
func (a *writingState) NovelStorySpineGet() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	sp, err := pm.ReadStorySpine()
	if err != nil {
		return nil, fmt.Errorf("读取故事骨架失败: %w", err)
	}
	raw, err := json.Marshal(sp)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// NovelStorySpineSave 整表保存（校验失败整单拒绝，不静默吞）。
func (a *writingState) NovelStorySpineSave(spineJSON string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	var sp types.StorySpine
	if err := json.Unmarshal([]byte(spineJSON), &sp); err != nil {
		return fmt.Errorf("故事骨架解析失败: %w", err)
	}
	if err := pm.WriteStorySpine(&sp); err != nil {
		return fmt.Errorf("故事骨架保存失败: %w", err)
	}
	return nil
}

// StoryHealthReport 结构体检报告（确定性零 LLM）。
type StoryHealthReport struct {
	CurrentChapter int                        `json:"current_chapter"`
	Findings       []types.StoryHealthFinding `json:"findings"`
	Counts         map[string]int             `json:"counts"` // 维度 → 条数
}

// NovelStoryHealth 故事骨架结构体检（七维度确定性，规格 §0 体检节）。
// 零 spine/零章=空报告不算错（新书正常态）；分析 V2 缺数据时中段维度如实标注。
func (a *writingState) NovelStoryHealth() (StoryHealthReport, error) {
	pm := a.getPM()
	if pm == nil {
		return StoryHealthReport{}, fmt.Errorf("请先打开项目")
	}
	sp, err := pm.ReadStorySpine()
	if err != nil {
		return StoryHealthReport{}, fmt.Errorf("读取故事骨架失败: %w", err)
	}
	cur, err := pm.MaxChapterNum()
	if err != nil {
		return StoryHealthReport{}, fmt.Errorf("扫描章节数失败: %w", err)
	}
	rep := StoryHealthReport{CurrentChapter: cur, Findings: []types.StoryHealthFinding{}, Counts: map[string]int{}}
	add := func(code, sev, ref, msg string) {
		rep.Findings = append(rep.Findings, types.StoryHealthFinding{Code: code, Severity: sev, Ref: ref, Message: msg})
		rep.Counts[code]++
	}

	// ① 支线遗忘：active/parked 且当前章-最后推进 > 10
	for i := range sp.Threads {
		t := &sp.Threads[i]
		if t.Status != types.ThreadStatusActive && t.Status != types.ThreadStatusParked {
			continue
		}
		last := t.LastAdvancedChapter
		if last <= 0 {
			last = t.OpenChapter
		}
		if last > 0 && cur-last > 10 {
			sev := "S2"
			if t.Status == types.ThreadStatusActive {
				sev = "S1" // 活跃线遗忘比停放线更急
			}
			add("thread_stale", sev, t.Name,
				fmt.Sprintf("支线「%s」（%s，第%d章开启）已 %d 章未推进（最后推进第%d章）：推进它，或显式改为 parked/abandoned",
					t.Name, threadStatusLabel(t.Status), t.OpenChapter, cur-last, last))
		}
	}

	// ② 未解问题超期：open 且超 15 章未答
	for i := range sp.OpenQuestions {
		q := &sp.OpenQuestions[i]
		if q.Status != types.QuestionOpen {
			continue
		}
		if cur-q.RaisedChapter > 15 {
			add("question_overdue", "S2", q.ID,
				fmt.Sprintf("未解问题「%s」第%d章提出，已挂 %d 章：安排回答，或作者显式放弃（改状态）", q.Question, q.RaisedChapter, cur-q.RaisedChapter))
		}
	}

	// ③ 节拍滞后：planned 绑定章 + 容差 3 < 当前章
	for i := range sp.Beats {
		b := &sp.Beats[i]
		if b.Status == types.BeatStatusHit || b.Chapter <= 0 {
			continue
		}
		if cur-b.Chapter > 3 {
			add("beat_overdue", "S2", b.Name,
				fmt.Sprintf("节拍「%s」计划第%d章，当前已第%d章仍未标达成：补标 hit（如实际已过）或改绑后续章", b.Name, b.Chapter, cur))
		}
	}

	// ④ 中段塌陷：分析 V2 强度中段（30%~70%）均值 < 全书均值×0.75
	if cur >= 6 {
		if f, ferr := pm.ReadAnalysisV2File(); ferr == nil && f != nil && len(f.Items) >= 4 {
			type pt struct{ ch, v int }
			var pts []pt
			for _, it := range f.Items {
				v := it.Result.EmotionalArc.Intensity
				if v > 0 {
					pts = append(pts, pt{it.ChapterNum, v})
				}
			}
			if len(pts) >= 4 {
				sort.Slice(pts, func(i, j int) bool { return pts[i].ch < pts[j].ch })
				lo := pts[0].ch + (pts[len(pts)-1].ch-pts[0].ch)*3/10
				hi := pts[0].ch + (pts[len(pts)-1].ch-pts[0].ch)*7/10
				sumAll, nAll, sumMid, nMid := 0, 0, 0, 0
				for _, p := range pts {
					sumAll += p.v
					nAll++
					if p.ch >= lo && p.ch <= hi {
						sumMid += p.v
						nMid++
					}
				}
				if nMid >= 2 && nAll > 0 {
					meanAll := float64(sumAll) / float64(nAll)
					meanMid := float64(sumMid) / float64(nMid)
					if meanMid < meanAll*0.75 {
						add("mid_sag", "S2", fmt.Sprintf("第%d-%d章", lo, hi),
							fmt.Sprintf("中段塌陷：第%d-%d章情感强度均值 %.1f 低于全书均值 %.1f 的 75%%——中段安排一次价值翻转或支线推进", lo, hi, meanMid, meanAll))
					}
				}
			}
		}
	}

	// ⑤ 弧线水位覆盖：最后一个水位点 < 当前进度 1/3
	for i := range sp.Arcs {
		arc := &sp.Arcs[i]
		if len(arc.Beats) == 0 {
			continue
		}
		lastCh := 0
		for _, b := range arc.Beats {
			if b.Chapter > lastCh {
				lastCh = b.Chapter
			}
		}
		if cur >= 6 && lastCh < cur/3 {
			add("arc_thin", "S3", arc.Name,
				fmt.Sprintf("弧线「%s」最后一个水位点在第%d章，当前已第%d章——弧线长期无推进，补一个 escalate 水位（反证+代价递增）", arc.Name, lastCh, cur))
		}
	}

	// ⑥ 终局未闭合：进入最后 15%（climax 节拍优先估计终局点）仍有 active 线程/open 问题
	finaleAt := 0
	for i := range sp.Beats {
		if sp.Beats[i].ID == types.BeatIDClimax && sp.Beats[i].Chapter > 0 {
			finaleAt = sp.Beats[i].Chapter
			break
		}
	}
	if finaleAt == 0 {
		finaleAt = cur * 85 / 100
	}
	if cur >= finaleAt && cur > 0 {
		var openThreads, openQs []string
		for i := range sp.Threads {
			if sp.Threads[i].Status == types.ThreadStatusActive || sp.Threads[i].Status == types.ThreadStatusParked {
				openThreads = append(openThreads, sp.Threads[i].Name)
			}
		}
		for i := range sp.OpenQuestions {
			if sp.OpenQuestions[i].Status == types.QuestionOpen {
				openQs = append(openQs, sp.OpenQuestions[i].Question)
			}
		}
		if len(openThreads) > 0 {
			add("finale_open", "S1", strings.Join(openThreads, "、"),
				fmt.Sprintf("已进入终局段（第%d章 ≥ 预计 climax 第%d章），支线仍未闭合：%s——终局前收束或显式 abandoned", cur, finaleAt, strings.Join(openThreads, "、")))
		}
		if len(openQs) > 0 {
			add("finale_open", "S2", strings.Join(openQs, "、"),
				fmt.Sprintf("终局段仍有未解问题：%s——安排回答（读者会带着这些问题读结尾）", strings.Join(openQs, "；")))
		}
	}

	// ⑦ 主题缺失（M12 报告制，永不阻断）
	if strings.TrimSpace(sp.Theme.ControllingIdea) == "" {
		add("theme_missing", "S3", "theme",
			"控制理念为空：这本书在讲什么道理还没写下来——用「AI 提炼骨架」或手填一句正命题+反论最强陈述")
	}

	return rep, nil
}

func threadStatusLabel(s string) string {
	switch s {
	case types.ThreadStatusActive:
		return "活跃"
	case types.ThreadStatusParked:
		return "停放"
	case types.ThreadStatusClosed:
		return "已闭合"
	case types.ThreadStatusAbandoned:
		return "已放弃"
	}
	return s
}

// storySpineSection 生成注入的故事层切片（CreateChapter 与逐场景流共用）。
// 空 spine / 无任何可注入项 → 返回空串（零注入零回归）。
func (a *writingState) storySpineSection(pm interface {
	ReadStorySpine() (*types.StorySpine, error)
}, chapterNum int) string {
	sp, err := pm.ReadStorySpine()
	if err != nil || sp == nil {
		return ""
	}
	var b strings.Builder
	// 本章绑定的节拍位
	for i := range sp.Beats {
		bt := &sp.Beats[i]
		if bt.Chapter == chapterNum && bt.Chapter > 0 {
			status := "计划达成"
			if bt.Status == types.BeatStatusHit {
				status = "已达成"
			}
			fmt.Fprintf(&b, "- 节拍「%s」（%s）：本章承载这一结构转折\n", bt.Name, status)
		}
	}
	// 活跃线程（推进/可收束素材）
	for i := range sp.Threads {
		t := &sp.Threads[i]
		if t.Status == types.ThreadStatusActive || t.Status == types.ThreadStatusParked {
			fmt.Fprintf(&b, "- 支线「%s」（%s，第%d章开启）：%s——可自然推进或埋钩子\n",
				t.Name, threadStatusLabel(t.Status), t.OpenChapter, t.Note)
		}
	}
	// 未解问题（悬念素材）
	for i := range sp.OpenQuestions {
		q := &sp.OpenQuestions[i]
		if q.Status == types.QuestionOpen {
			fmt.Fprintf(&b, "- 悬而未决：「%s」（第%d章提出）——不要无意中回答，也不要遗忘\n", q.Question, q.RaisedChapter)
		}
	}
	// 主角弧线最近水位点（行动口径）
	for i := range sp.Arcs {
		arc := &sp.Arcs[i]
		last := -1
		for j := range arc.Beats {
			if arc.Beats[j].Chapter < chapterNum && arc.Beats[j].Chapter > last {
				last = j
			}
		}
		if last >= 0 {
			bt := arc.Beats[last]
			stage := arcStageLabel(bt.Stage)
			fmt.Fprintf(&b, "- 「%s」弧线当前水位（第%d章·%s）：%s——%s 的行动要带着这个错误信念的当前状态\n",
				arc.Name, bt.Chapter, stage, bt.Note, arc.Name)
		}
	}
	body := strings.TrimRight(b.String(), "\n")
	if body == "" {
		return ""
	}
	return "## 故事层骨架（纵向结构切片，本章写作要对齐）\n" + body
}

func arcStageLabel(s string) string {
	switch s {
	case types.ArcStageSetup:
		return "确立"
	case types.ArcStageProvoke:
		return "初次反证"
	case types.ArcStageEscalate:
		return "反证递进"
	case types.ArcStageProof:
		return "行动推翻"
	}
	return "未标阶段"
}

// storySpineProposal AI 提案面（不落盘形态）。
type storySpineProposal struct {
	Theme struct {
		ControllingIdea string `json:"controlling_idea"`
		CounterIdea     string `json:"counter_idea"`
	} `json:"theme"`
	Arcs []struct {
		Name      string `json:"name"`
		Want      string `json:"want"`
		Need      string `json:"need"`
		Misbelief string `json:"misbelief"`
		FirstBeat string `json:"first_beat"` // 现有水位一句话（含章号），作者审批时手动落表
	} `json:"arcs"`
	Beats []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Chapter int    `json:"chapter"`
	} `json:"beats"`
	Threads []struct {
		Name        string `json:"name"`
		MICEType    string `json:"mice_type"`
		OpenChapter int    `json:"open_chapter"`
		Status      string `json:"status"`
		Note        string `json:"note"`
	} `json:"threads"`
	Questions []struct {
		Question      string `json:"question"`
		RaisedChapter int    `json:"raised_chapter"`
		Status        string `json:"status"`
	} `json:"questions"`
}

// NovelStorySpinePropose 从故事主线+大纲+章摘要提炼骨架初稿（提案不落盘，确认制）。
func (a *writingState) NovelStorySpinePropose() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	thread := ""
	if of, err := pm.ReadOutlines(); err == nil && of != nil {
		thread = strings.TrimSpace(of.StoryThread)
	}
	if thread == "" {
		return nil, fmt.Errorf("故事主线为空：先在大纲规划里生成一次主线（或手填），AI 才能提炼骨架")
	}
	maxCh, _ := pm.MaxChapterNum()
	summaries, _ := pm.ReadMainlineChapterSummaries()
	var sb strings.Builder
	sb.WriteString("故事主线：\n" + thread + "\n\n")
	if maxCh > 0 {
		sb.WriteString(fmt.Sprintf("已写 %d 章。\n", maxCh))
	}
	if len(summaries) > 0 {
		n := len(summaries)
		if n > 8 {
			n = 8
		}
		sb.WriteString("最近章摘要（从旧到新）：\n")
		for _, s := range summaries[len(summaries)-n:] {
			sb.WriteString("- " + util.Truncate(s.Summary, 120) + "\n")
		}
		sb.WriteString("\n")
	}

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}
	system := "你是长篇小说的结构编辑。从故事主线与既有章节提炼「故事骨架」：一句控制理念与它的最强反论；主要角色的 want/need/misbelief 弧线；五个结构节拍（hook/first_plot_point/midpoint/second_plot_point/climax）的建议章号；已开启的支线（按 MICE 分类）与未解问题。" +
		"只输出一个 JSON 对象，第一个字符是 {，最后一个字符是 }，不要解释文字或 Markdown 围栏。已有内容的章号从摘要推断，未写的部分给计划值。"
	sb.WriteString(`字段契约：{"theme":{"controlling_idea":"正命题一句话","counter_idea":"反论最强陈述一句话"},"arcs":[{"name":"角色名","want":"表层欲望","need":"深层需要","misbelief":"错误信念","first_beat":"当前水位一句话"}],"beats":[{"id":"hook|first_plot_point|midpoint|second_plot_point|climax","name":"中文名","chapter":章号}],"threads":[{"name":"支线名","mice_type":"query|character|event|place|thing","open_chapter":章号,"status":"active|parked|closed|abandoned","note":"一句话"}],"questions":[{"question":"读者视角的开放悬念","raised_chapter":章号,"status":"open|answered"}]}`)

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model, system, sb.String(), ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.4, MaxTokens: 3500, TimeoutMinutes: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("提炼骨架失败: %w", err)
	}
	var p storySpineProposal
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &p); err != nil {
		return nil, fmt.Errorf("提炼结果解析失败: %w", err)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
