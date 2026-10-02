package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/maturecraft"
	"github.com/gaea/gaea/internal/novelcontext"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// CreateChapter 生成章节。minWords 为单章目标字数（<=0 使用默认 5000）；
// temperature 为生成温度（<=0 使用模型服务端默认值）。
//
// 写前硬闸默认开（规格 docs/gaea-longform-novel-system-2026-09.md §7.3）：章节计划不齐备 /
// 跨章关键事件重复 / 大纲节点契约缺项一律拒绝生成。签名保持不变（绑定面稳定），
// 该入口恒等于 allowOverride=false；显式覆盖走 CreateChapterWithOverride。
func (a *writingState) CreateChapter(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string, skillName string, minWords int, temperature float64) (map[string]interface{}, error) {
	return a.createChapter(setting, prevSummary, plotReq, chapterNum, branchFromNodeID, skillName, minWords, temperature, false)
}

// CreateChapterWithOverride 显式覆盖写前硬闸的生成入口（§7.3 的 AllowOverride，默认关）。
// allowOverride=true 表示作者确认「没有计划也要生成」：预检照跑、结论照记日志，但不阻断。
// 绑定层要把覆盖意图透传给前端时转发本方法即可——新增参数不触碰既有 CreateChapter 调用点。
func (a *writingState) CreateChapterWithOverride(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string, skillName string, minWords int, temperature float64, allowOverride bool) (map[string]interface{}, error) {
	return a.createChapter(setting, prevSummary, plotReq, chapterNum, branchFromNodeID, skillName, minWords, temperature, allowOverride)
}

// createChapter 章节生成实现：章号定号 → 写前硬闸（planPrecheck）→ 意图注入
// （章节计划 + 大纲要点）→ 上下文增强 → 流式生成。
func (a *writingState) createChapter(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string, skillName string, minWords int, temperature float64, allowOverride bool) (map[string]interface{}, error) {
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	_ = prevSummary // 参数弃用占位：摘要改由下方 buildPrevSummaryWindow 滑窗推导，保留签名稳定绑定面

	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("project not open")
	}

	// 1. 小说设定不允许为空
	if setting == "" {
		return nil, fmt.Errorf("小说设定为空，请先在「设定」页面填写世界观")
	}
	// 刀6：setting 槽入预算——此前整篇设定无裁剪直进模板 P0 槽（预算外堆料的
	// 最大残留）；超长截断并如实告知（分维度要点仍在「世界观要点」区段）。
	if r := runeLen(setting); r > ctxSettingBudget {
		setting = truncateBudget(setting, ctxSettingBudget) +
			"\n（设定过长已截断；分维度要点见下方「世界观要点」区段）"
	}

	// 2. 加载 Skill 写作指导
	skillMD := ""
	if skillName != "" && a.skillLoader != nil {
		skillMD = skillName // 保存 skill 名称供后续注入
		slog.Info("CreateChapter 将注入 Skill", "name", skillName)
	}

	// 3. 从章节节点树提取前文摘要（t3 首刀：最近 10 章窗口预算化，规格
	// docs/distill/03-long-range-consistency.md §11.3 缺口 1 / §12.1——原实现
	// 200 rune×全部前章无界拼接，200 章≈40k rune 的 prompt 前缀）。
	of, err := pm.ReadOutlines()
	if err != nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	limitChapter := chapterNum
	if limitChapter <= 0 {
		limitChapter = len(of.Nodes) + 1
	}
	// 摘要回退链（spec §12.4-5）：大纲节点 Summary → 章节摘要文件 → 跳过该章。
	resolvePrevSummary := func(n types.OutlineNode) string {
		if s := strings.TrimSpace(n.Summary); s != "" {
			return s
		}
		if cs, err := pm.ReadChapterSummary(n.OrderIndex); err == nil && cs != nil {
			if s := strings.TrimSpace(cs.Summary); s != "" {
				return s
			}
		}
		return ""
	}
	prevSummary = buildPrevSummaryWindow(of.Nodes, limitChapter, resolvePrevSummary)

	if minWords <= 0 {
		minWords = 5000
	}

	// 4. 本章章号（生成前口径，与 ensureChapterNode 定号逻辑同源）：显式指定直接用；
	//    分支续写取父节点章号；否则顺延为新章。硬闸与意图注入都按它取计划/大纲节点。
	ctxChapterNum := resolveTargetChapterNum(of, chapterNum, branchFromNodeID)

	// 5. 写前硬闸（规格 §7.3）：在真正调用模型**之前**判定「有没有抓手」——计划齐备性 +
	//    跨章关键事件去重 + 大纲节点契约。判据唯一来源是 planPrecheck（线C 的
	//    NovelChapterGatePrecheck 走同一入口，两处结论必然一致）；存在 S1/S2 即拒绝，
	//    只有调用方显式覆盖（allowOverride）才放行——放行也照记日志，不假装没这回事。
	if ctxChapterNum > 0 {
		report, perr := a.planPrecheck(ctxChapterNum)
		if perr != nil {
			return nil, fmt.Errorf("生成前预检失败（第%d章）：%w", ctxChapterNum, perr)
		}
		if report.Blocking && !allowOverride {
			return nil, planGateError(report)
		}
		if report.Blocking {
			slog.Warn("CreateChapter 显式覆盖写前硬闸", "chapter", ctxChapterNum,
				"missing", strings.Join(report.Missing, "、"))
		}
	}

	// 6. 意图注入（规格 §7.3）：把「这一章为什么存在」变成 prompt 区段——章节计划
	//    （六项判据字段全量渲染）+ 大纲节点 KeyPoints/Emotion（此前零进入 prompt）。
	//    两者为空时渲染空串，模板按空跳过（prompt 中不出现空区段）；此时硬闸已在
	//    上一步拦下（除非显式覆盖）。读计划失败静默跳过注入——绝不因注入失败中断主链路。
	planSec, outlineSec := "", ""
	var chapterPlan *types.ChapterPlan
	if ctxChapterNum > 0 {
		if plans, err := readChapterPlansForGate(pm); err == nil && plans != nil {
			if p, ok := plans.Plans[strconv.Itoa(ctxChapterNum)]; ok {
				planSec = buildChapterPlanSection(&p)
				chapterPlan = &p
			}
		}
		outlineSec = buildOutlinePointsSection(findOutlineNodeByNum(of.Nodes, ctxChapterNum))
	}

	// 7. 构建 prompt（通过模板 + Skill 注入）。第一章是全书开篇，走开篇专用
	//    模板（缺失回落通用，见 pickCreateChapterTemplate）。
	tmpl := pickCreateChapterTemplate(a.eng, ctxChapterNum)
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 create-chapter 模板文件")
	}

	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"plot_req":         plotReq,
		"setting":          setting,
		"characters":       a.buildChapterCastSection(pm, chapterPlan), // 焦点分级：计划角色焦点登场，其余降名册备查
		"character_states": a.buildCharacterStatesSection(pm),          // t5 §7.4 状态机回灌（P1，空则不渲染）
		"prev_summary":     prevSummary,
		"chapter_plan":     planSec,                                  // 刀1 §7.3（P0）：本章计划区段，空则不渲染
		"outline_points":   outlineSec,                               // 刀1 §7.3（P1）：大纲 KeyPoints/Emotion，空则不渲染
		"mature_craft":     maturecraft.CraftSection(pm.Meta.Mature), // v4.439 成人向工艺区段（P1）：非成人向为空串零渲染
	})
	// 上下文增强：追加分层伏笔调度 + 世界观要点区段。全部容错注入——读取失败或
	// 无数据时静默跳过（不追加空区段），绝不因增强失败中断章节生成主链路；
	// 新增区段合计受 ctxBudgetTotal 预算约束，超出逐段截断。
	// 本章计划/大纲要点两个新区段先按各自上限占用额度，剩余额度再给伏笔/文风/世界观
	// （同一 ctxBudgetTotal 口径，不叠加成第二套预算）。
	ctxBudgetLeft := ctxBudgetTotal - runeLen(planSec) - runeLen(outlineSec)
	// 刀6：hint=计划+大纲要点文本（世界观相关性排序）；digest 指令并入文风区段
	// （此前独立追加=文风两处的延续，本刀合并为单一区段）。
	ctxHint := planSec + "\n" + outlineSec
	digestInstr := a.digestBody(pm)
	if extra := buildChapterContextSectionsWithin(pm, ctxChapterNum, ctxBudgetLeft, ctxHint, digestInstr); extra != "" {
		userPrompt += extra + "\n"
	}

	// 刀3：故事层骨架切片（节拍位/活跃线程/未解问题/弧线水位）——空 spine 零注入，
	// 读取失败静默跳过（同上容错纪律）。切片短且有独立结构价值：不挤占上方
	// 伏笔/文风/世界观的 ctxBudget 额度，独立追加在增强区段之后。
	spineSec := a.storySpineSection(pm, ctxChapterNum)
	if spineSec != "" {
		userPrompt += spineSec + "\n"
	}

	// 阶段开篇章（第 21/41/…章）：先合账再开新账——注入上一阶段总结（LLM 合成、
	// 失败回落逐章摘要）与新阶段开篇纪律。非阶段开篇章零注入；内部任何失败都
	// 降级不阻断主链路（同 spine 纪律）。
	if sec := a.stageRecapSection(pm, ctxChapterNum); sec != "" {
		userPrompt += "\n" + sec + "\n"
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	// create-chapter 模板以 {word_count} 占位符声明目标字数（prompts/create-chapter.json），
	// 用户可在创作页调整目标字数；用实际 minWords 精确替换占位符，避免模型仍按
	// 固定字数生成，同时杜绝旧 ReplaceAll("5000") 误伤模板中其他 "5000" 字样。
	systemPrompt = substituteWordCount(systemPrompt, minWords)
	if skillMD != "" && a.skillLoader != nil {
		if s := a.skillLoader.Get(skillMD); s != nil {
			systemPrompt = a.skillLoader.InjectSkill(systemPrompt, skillMD)
			slog.Info("CreateChapter Skill 已注入", "name", s.Name, "version", s.Version)
		} else {
			slog.Warn("CreateChapter Skill 未找到", "skill", skillMD)
		}
	}
	const maxContinues = 20
	// 8. 确定/创建节点（同步，前端立即可用）——硬闸已通过（或已显式覆盖）才走到这里，
	//    被拒绝的请求不建节点、不落盘。落盘失败必须**中止本次生成**：把节点当成
	//    已建好继续跑流式生成，正文会落到一个大纲里不存在的章号上，作者看到的
	//    「已建好」是假的（N8）。
	targetNum, nodeID, branch, err := a.ensureChapterNode(pm, of, chapterNum, branchFromNodeID)
	if err != nil {
		return nil, err
	}
	if nodeID == "" {
		return nil, fmt.Errorf("创建章节节点失败")
	}

	// 9. 场景圣经注入（novelcontext，POV 感知）：把本场景/本章的世界状态 +
	//    按 POV 裁剪的知识视图 + 未回收伏笔 + 时间锚点 + 文风，编译成一段紧凑
	//    场景圣经注入生成 prompt，替代「扁平截断前文摘要」的失忆问题。
	//    任何编译失败都静默跳过——绝不因增强失败中断生成主链路。
	if bible, err := novelcontext.BuildSceneBibleFromChapter(pm, targetNum); err == nil && bible != nil {
		// 相关记忆召回（t3-P2）：结构化 query → 语义检索 → 阈值/兜底筛选，
		// 注入 SceneBible 记忆区段。全链容错：无库/无 embedding/无记忆返回 nil。
		if node := findOutlineNodeByID(of.Nodes, nodeID); node != nil {
			bible.Memories = a.recallStoryMemories(pm, node, plotReq, targetNum)
		}
		if r := bible.Render(ctxSceneBibleBudget); r != "" {
			userPrompt += "\n\n" + r
		}
	}

	// 10. 按章节互斥（T6-7.2）：同一章节（同一 NNN.md 目标文件）并发生成直接拒绝，
	//    不同章节可并行。登记时创建请求级 context，供前端 CancelCreateChapter 取消，
	//    取消经该 context 传播到 ChatStream 与流读取循环。
	genKey := chapterGenKey(targetNum, branch)
	genCtx, genCancel, err := a.registerChapterGen(genKey, targetNum, branch)
	if err != nil {
		return nil, err
	}

	// 启动流式生成 + 字数守卫（续写模式）
	a.chapterGenWG.Add(1)
	go func() {
		defer a.chapterGenWG.Done()
		defer a.unregisterChapterGen(genKey, genCancel)
		a.streamCreateChapter(genCtx, pm, of, setting, prevSummary, plotReq, chapterNum, branchFromNodeID, systemPrompt, userPrompt, minWords, maxContinues, temperature, targetNum, nodeID, branch, skillName == "story-deslop")
	}()

	return map[string]interface{}{
		"streaming":  true,
		"chapterNum": targetNum,
		"nodeId":     nodeID,
		"branch":     branch,
	}, nil
}

// chapterGenKey 章节生成任务标识：目标章节文件（NNN.md / NNN{branch}.md）。
func chapterGenKey(chapterNum int, branch string) string {
	return fmt.Sprintf("%d|%s", chapterNum, branch)
}

// registerChapterGen 登记进行中的章节生成并创建请求级 context。
// 同一章节已在生成时返回明确错误（拒绝并发写同一 NNN.md）。取消已请求但
// 生成协程尚未退出（登记值为 nil，见 CancelCreateChapter）同样如实拒绝——
// 协程仍活着，放行会并发出两个写者（P1 修复）。
func (a *writingState) registerChapterGen(key string, chapterNum int, branch string) (context.Context, context.CancelFunc, error) {
	a.chapterGenMu.Lock()
	defer a.chapterGenMu.Unlock()
	if a.chapterGenCancels == nil {
		a.chapterGenCancels = make(map[string]context.CancelFunc)
	}
	if prev, running := a.chapterGenCancels[key]; running {
		label := fmt.Sprintf("第%d章", chapterNum)
		if branch != "" {
			label = fmt.Sprintf("第%d%s章", chapterNum, branch)
		}
		if prev == nil {
			// 取消已请求、生成协程尚未退出：登记表清理由 unregisterChapterGen
			// 独占（协程退出时才删），此窗口内不放行同章再生成。
			return nil, nil, fmt.Errorf("%s 的生成正在取消中，请等它退出后再重新生成", label)
		}
		return nil, nil, fmt.Errorf("%s 正在生成中，请等待完成或先取消", label)
	}
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.chapterGenCancels[key] = cancel
	return ctx, cancel, nil
}

// unregisterChapterGen 生成结束（成功/失败/取消）后移除登记并释放取消函数：
// 登记表清理由本函数独占（取消路径只置 nil 占位，不删条目），故表项的消失
// 恒等于「生成协程已退出」，register 的并发判定因此不会误放行。
// cancel 幂等：生成已结束，仅释放 context 资源。
func (a *writingState) unregisterChapterGen(key string, cancel context.CancelFunc) {
	a.chapterGenMu.Lock()
	delete(a.chapterGenCancels, key)
	a.chapterGenMu.Unlock()
	cancel()
}

// CancelCreateChapter 取消指定章节的进行中生成（T6-7.2）。
// 取消后 streamCreateChapter 会把已生成部分落盘并向前端发 cancelled 事件。
// 幂等：目标章节没有进行中生成、或本次取消已请求过时返回 false。
//
// P1 修复：取消只调用 cancel，不动登记表的条目——条目改置 nil（「已请求取消、
// 协程未退出」的窄状态），由 unregisterChapterGen 在协程真正退出时独占清除。
// 旧实现此处 delete 会在协程还活着时摘掉互斥，取消后立刻再生成同章即放行，
// 造成同一 NNN.md 两个写者。幂等仍成立：重复取消看到 nil 占位即返回 false。
func (a *writingState) CancelCreateChapter(chapterNum int, branch string) bool {
	key := chapterGenKey(chapterNum, branch)
	a.chapterGenMu.Lock()
	cancel, running := a.chapterGenCancels[key]
	if running && cancel != nil {
		a.chapterGenCancels[key] = nil // 占位：已请求取消，等 unregisterChapterGen 清
	}
	a.chapterGenMu.Unlock()
	if !running || cancel == nil {
		return false
	}
	cancel()
	return true
}

// wordCountPlaceholder create-chapter 模板中目标字数的占位符
// （prompts/create-chapter.json 的 task 与 output 两处字数声明均使用它）。
const wordCountPlaceholder = "{word_count}"

// wordCountPlaceholderV2 t6 起占位符统一迁移到双花括号语法（规格书 §2 Q1
// 裁决：与 {{name}} 渲染口径一致，避免与 JSON 字面单花括号混淆）。
const wordCountPlaceholderV2 = "{{word_count}}"

// substituteWordCount 将模板中的目标字数占位符精确替换为实际字数。t6 起为
// 双语法（规格书 §4.3）：先替换新语法 {{word_count}}，再兜底替换旧语法
// {word_count}——模板文件随 t6 迁移到新语法，但用户手改盘上的旧语法模板
// 不炸（迁移期兼容，调用点零改动）。
// 旧实现 strings.ReplaceAll("5000") 会误伤模板中其他 "5000" 字样；占位符替换
// 只命中字数声明位，其余数字字样原样保留。
func substituteWordCount(prompt string, minWords int) string {
	n := strconv.Itoa(minWords)
	replaced := strings.ReplaceAll(prompt, wordCountPlaceholderV2, n)
	return strings.ReplaceAll(replaced, wordCountPlaceholder, n)
}

// chapterCurrentBody 计算当前已生成的纯正文（不含 ---CHAPTER_SUMMARY--- 摘要）。
// 首次生成且未见摘要标记时正文在 fullText 中；出现标记或进入续写后正文在 bodyText 中。
func chapterCurrentBody(fullText, bodyText string, attempt int, summaryStarted bool) string {
	if attempt > 0 || summaryStarted {
		return bodyText
	}
	if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
		return fullText[:idx]
	}
	return fullText
}

// chapterPartialFileSuffix 取消生成残稿的文件名标记（NNN.partial-<时间戳>-<序号>.md）。
const chapterPartialFileSuffix = ".partial-"

// chapterPartialSeq 残稿文件名自增序号：同进程内保证同一刻两次取消不撞名。
var chapterPartialSeq atomic.Uint64

// chapterBodyExists 目标章是否已有正文：
//   - 分支章：NNN{branch}.md 非空即算；
//   - 主线：blob（chapters/NNN.md）非空，或（v4 场景制）该章已有承载非空正文的场景。
//
// 供取消生成时的覆盖保护使用：生成开始时快照一次（existedBefore），取消落盘前
// 再复查一次（生成期间目标章可能被其他写者补上正文）。读取失败按「无正文」处理
// ——保护判定只用于「不覆盖」，宁可直接写正稿也不额外阻断既有落盘路径。
func chapterBodyExists(pm *project.Manager, targetNum int, branch string) bool {
	if pm == nil || targetNum <= 0 {
		return false
	}
	if branch != "" {
		blob, err := pm.ReadChapterBranch(targetNum, branch)
		return err == nil && strings.TrimSpace(blob) != ""
	}
	if blob, err := pm.ReadChapter(targetNum); err == nil && strings.TrimSpace(blob) != "" {
		return true
	}
	if !pm.IsV4() {
		return false // 场景制之外没有独立正文载体
	}
	sm := pm.SceneManager(targetNum)
	metas, err := sm.List()
	if err != nil {
		return false
	}
	for _, meta := range metas {
		if sc, rerr := sm.Read(meta.ID); rerr == nil && strings.TrimSpace(sc.Content) != "" {
			return true
		}
	}
	return false
}

// chapterPartialPath 取消残稿落点：与正稿同目录、同基名 + .partial-<ts>.md。
// 目录一律取自 project.Manager 的章节路径（沿用 chapters/ 布局护栏，不自行
// 拼接项目路径）；分支章用分支正稿路径派生。
//
// 时间戳到纳秒 + 进程内自增序号：旧的秒级时间戳（20060102150405）在同一秒内
// 两次取消会落到同一文件名，后一份残稿覆盖前一份（M5）。纳秒尾在 Windows 上
// 仍可能取到同值，故再用序号兜底，保证每次取消各留一份残稿。
func chapterPartialPath(pm *project.Manager, targetNum int, branch string) string {
	base := pm.ChapterPath(targetNum)
	if branch != "" {
		base = pm.ChapterBranchPath(targetNum, branch)
	}
	ts := time.Now().Format("20060102150405.000000000")
	return strings.TrimSuffix(base, ".md") + chapterPartialFileSuffix + ts +
		fmt.Sprintf("-%d", chapterPartialSeq.Add(1)) + ".md"
}

// writeCancelledPartialSidecar 把取消残稿另存为侧车文件（不覆盖正稿），返回落点。
// 原子写复用 kernel 共享实现 fileutil.AtomicWrite——project.Manager 的
// writeFileAtomic 走的是同一套临时文件 + RenameWithRetry 语义。
func writeCancelledPartialSidecar(pm *project.Manager, targetNum int, branch, partial string) (string, error) {
	if pm == nil {
		return "", fmt.Errorf("project not open")
	}
	path := chapterPartialPath(pm, targetNum, branch)
	if err := fileutil.AtomicWrite(path, []byte(partial), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// saveCancelledPartial 取消生成时把已生成部分落盘并通知前端（T6-7.2）。
// 保持正常完成的落盘路径不变，仅补充取消场景的部分写入；未生成任何内容
// （partial 为空）时只发 cancelled 事件、不写空文件，避免覆盖既有章节。
//
// 覆盖保护（P0）：existedBefore 为「本次生成开始时目标章是否已有正文」的
// 快照（由 streamCreateChapter 在生成开始时记录），取消时再复查一次。只要
// 目标章已有正文（blob 非空 / v4 场景承载正文），残稿绝不覆盖正稿：
//   - 残稿另存为 NNN.partial-<yyyyMMddHHmmss>.md，正稿字节原样保留；
//   - 事件补 partialSaved / partialPath / notice 三个可选字段，前端据此
//     如实提示「正文已存在，残稿另存为 …」（不再谎称正文已被部分保存）；
//   - slog.Info 记录落点。
//
// 目标章此前不存在时才保持现行为：把已生成部分写入正稿。
func (a *writingState) saveCancelledPartial(pm *project.Manager, fullText, bodyText string, attempt int, summaryStarted bool, targetNum int, nodeID, branch string, existedBefore bool) {
	partial := strings.TrimSpace(chapterCurrentBody(fullText, bodyText, attempt, summaryStarted))
	label := fmt.Sprintf("第%d章", targetNum)
	if branch != "" {
		label = fmt.Sprintf("第%d%s章", targetNum, branch)
	}
	payload := map[string]interface{}{
		"type":       "cancelled",
		"chapterNum": targetNum,
		"branch":     branch,
		"nodeId":     nodeID,
		"total":      len([]rune(partial)),
	}
	if partial == "" {
		a.emit("create-chapter-stream", payload)
		return
	}
	if existedBefore || chapterBodyExists(pm, targetNum, branch) {
		path, err := writeCancelledPartialSidecar(pm, targetNum, branch, partial)
		if err != nil {
			// 另存失败也绝不回落到覆盖正稿：如实报失败（不带 content）。
			slog.Warn("取消生成：正文已存在，残稿另存失败", "chapter", targetNum, "branch", branch, "error", err)
			a.emit("create-chapter-stream", payload)
			return
		}
		payload["content"] = partial
		payload["partialSaved"] = true
		payload["partialPath"] = path
		payload["notice"] = fmt.Sprintf("%s正文已存在，残稿未覆盖正稿，已另存为 %s", label, filepath.Base(path))
		slog.Info("取消生成：正文已存在，残稿另存未覆盖正稿", "chapter", targetNum, "branch", branch, "path", path)
		a.emit("create-chapter-stream", payload)
		return
	}
	var err error
	if branch != "" {
		err = pm.WriteChapterBranch(targetNum, branch, partial)
	} else {
		err = pm.WriteChapter(targetNum, partial)
	}
	if err != nil {
		slog.Warn("取消生成：已生成部分落盘失败", "chapter", targetNum, "branch", branch, "error", err)
	} else {
		payload["content"] = partial
	}
	a.emit("create-chapter-stream", payload)
}

// streamCreateChapter 在后台 goroutine 中流式生成章节，字数不足时续写。
// ctx 为请求级 context（T6-7.2）：由 CreateChapter 绑定入口创建，前端调用
// CancelCreateChapter 时取消；取消传播到 ChatStream 与流读取循环。
func (a *writingState) streamCreateChapter(ctx context.Context, pm *project.Manager, of *types.OutlineFile, setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID, systemPrompt, userPrompt string, minWords, maxContinues int, temperature float64, targetNum int, nodeID string, branch string, deSlop bool) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CreateChapter stream panic", "panic", r)
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("内部错误: %v", r)})
		}
	}()

	var fullText string // 全部原始内容（含摘要标记，用于提取 summary）
	var bodyText string // 纯正文（不含 ---CHAPTER_SUMMARY--- 及摘要，用于字数和最终保存）
	currentPrompt := userPrompt

	// 生成开始时的目标章状态快照（P0 覆盖保护）：本次生成开始前已有正文时，
	// 取消残稿绝不覆盖正稿（另存 NNN.partial-<ts>.md）。必须在此处记录而非
	// 等到取消时判文件是否存在——正常完成路径同样会写正稿，取消时目标文件
	// 已存在并不等于「本次产物」。
	existedBefore := chapterBodyExists(pm, targetNum, branch)

	// S1.5-B play 内容护栏：temperature_max/max_output_tokens 钳制（未配置
	// = 零值 = 请求与现状逐字节一致），每次续写尝试均按同一钳制值下发。
	g := playGuardrails()

	for attempt := 0; attempt <= maxContinues; attempt++ {
		if attempt > 0 {
			a.emit("create-chapter-stream", map[string]interface{}{
				"type":    "phase",
				"phase":   "continuing",
				"attempt": attempt,
				"current": len([]rune(bodyText)),
				"target":  minWords,
			})
			// 续写模式：基于已有内容继续
			bodyLen := len([]rune(bodyText))
			need := minWords - bodyLen
			// 截取已有内容末尾作为续写上下文（避免截开头导致重复）
			// 续写尾部 1500 rune（原 500）；正文不足 1500 时 tailRunes 保持全量（带全章）。
			allRunes := []rune(bodyText)
			tailRunes := allRunes
			const tailCtx = 1500
			if len(allRunes) > tailCtx {
				tailRunes = allRunes[len(allRunes)-tailCtx:]
			}
			tailText := string(tailRunes)
			if len(allRunes) > tailCtx {
				tailText = "…(前文省略)\n" + tailText
			}
			currentPrompt = fmt.Sprintf("【续写指令】当前已写%d字，请从断点处直接继续写至少%d字。不要重复已写内容，不要加章节标题或前言，直接接着写正文。\n\n已有内容末尾：\n%s\n\n请继续：",
				bodyLen, need, tailText)
			slog.Info("章节字数不足，启动续写", "current", bodyLen, "need", need, "attempt", attempt)
		} else {
			a.emit("create-chapter-stream", map[string]interface{}{
				"type":   "phase",
				"phase":  "writing",
				"target": minWords,
			})
		}

		// 向 AI 控制台发送请求日志
		featEng, featModel, _ := a.routeModel("novel")
		a.emit("xai-output", map[string]interface{}{
			"type":   "request",
			"model":  featModel,
			"system": systemPrompt,
			"user":   currentPrompt,
		})

		req := &ai.ChatRequest{
			Model:    featModel,
			EngineID: featEng,
			Feature:  "novel",
			Messages: []ai.ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: currentPrompt}},
		}
		applyChapterGuardrails(req, temperature, g)
		chunks, err := a.clientRef().ChatStream(ctx, req)
		if err != nil {
			// 连接建立阶段被取消（用户在流开始前点停止）：ChatStream 返回 ctx.Err()，
			// 不能当「生成失败」报——与流读取循环里的取消分支等价，落盘已生成
			// 部分并走 cancelled 事件（此时通常为空稿，后端只发事件不写空文件；
			// 连接尚未建立，本轮必然没进入摘要段，summaryStarted 恒 false）。
			if ctx.Err() != nil {
				a.saveCancelledPartial(pm, fullText, bodyText, attempt, false, targetNum, nodeID, branch, existedBefore)
				return
			}
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": err.Error()})
			return
		}

		var summaryStarted bool
	loop:
		for {
			select {
			case <-ctx.Done():
				// T6-7.2 用户取消：把已生成部分落盘后退出（不再续写）
				a.saveCancelledPartial(pm, fullText, bodyText, attempt, summaryStarted, targetNum, nodeID, branch, existedBefore)
				return
			case chunk, ok := <-chunks:
				if !ok {
					break loop
				}
				if chunk.Error != "" {
					if ctx.Err() != nil {
						// 取消导致的流中断（parseStreamEvents 在 ctx 取消时发 error 帧）：
						// 与上方 ctx.Done 分支等价，同样落盘已生成部分。
						a.saveCancelledPartial(pm, fullText, bodyText, attempt, summaryStarted, targetNum, nodeID, branch, existedBefore)
						return
					}
					a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": chunk.Error})
					return
				}
				if chunk.Done {
					break loop
				}
				fullText += chunk.Content

				if !summaryStarted {
					if attempt == 0 {
						// 首次生成：流式发送正文（需处理摘要标记）
						if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
							summaryStarted = true
							bodyText = fullText[:idx] // 锁定纯正文（标记之前）
							// 计算当前 chunk 中位于标记之前的字节数，用字节切片；
							// 原实现按 rune 下标切片，遇到中文等多字节字符会越界。
							prefixLen := idx - (len(fullText) - len(chunk.Content))
							if prefixLen > 0 && prefixLen <= len(chunk.Content) {
								bodyPart := chunk.Content[:prefixLen]
								a.emit("create-chapter-stream", map[string]interface{}{
									"type": "chunk", "content": bodyPart, "total": len([]rune(bodyText)),
								})
							}
						} else {
							a.emit("create-chapter-stream", map[string]interface{}{
								"type": "chunk", "content": chunk.Content, "total": len([]rune(fullText)),
							})
						}
					} else {
						// 续写：直接流式发送（无摘要标记），同时追加到纯正文
						bodyText += chunk.Content
						a.emit("create-chapter-stream", map[string]interface{}{
							"type": "chunk", "content": chunk.Content, "total": len([]rune(bodyText)),
						})
					}
				}
			}
		}

		// 首次生成无摘要标记时，bodyText = fullText
		if attempt == 0 && !summaryStarted {
			bodyText = fullText
		}

		// 检查字数（用纯正文 bodyText）
		if len([]rune(bodyText)) >= minWords {
			break // 达标
		}
	}

	// 提取正文和摘要
	content := strings.TrimSpace(bodyText) // 纯正文（含续写内容）

	// 去 AI 味后处理（story-deslop 确定性引擎）：仅替换 AI 高频词 + 归一标点，
	// 不改变情节/人物/篇幅；分数未改善则不落盘。成功时用去味文本覆盖 content，
	// 并把报告带进 done 的 aiTaste，供作者知悉改了多少。
	var deSlopReport *novelstyle.RewriteReport
	if deSlop {
		ensureNovelStyleWords()    // 惯例目录词表覆盖每进程加载一次
		ensureNovelStylePatterns() // 模式级门禁覆盖（oh-story T2）
		// 打分口径与手动「一键去味」路径一致（G8）：先算 before 分并应用书级白名单，
		// 再传入 DeSlopRewriteEx。旧实现传 nil，内部自行算的 before 分**不含白名单
		// 豁免**，而 after 分含——before 被抬高，AfterScore < BeforeScore 这道安全闸
		// 偏松，同一文本两条路径还能得到不同的 before 分。
		if rx, rep, err := deSlopRewriteWithin(content, bookWhitelist(pm)); err == nil && rep != nil && rep.AfterScore < rep.BeforeScore {
			if rx != "" {
				content = rx
				deSlopReport = rep
				slog.Info("章节去 AI 味已完成", "chapter", targetNum, "before", rep.BeforeScore, "after", rep.AfterScore, "changes", len(rep.Changes))
			}
		}
	}

	summary := ""
	if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
		summary = strings.TrimSpace(fullText[idx+len("---CHAPTER_SUMMARY---"):])
		summary = util.Truncate(summary, 100)
	}

	// 将摘要落盘，分支章节使用独立摘要文件，避免分支复用主线摘要
	if summary != "" {
		label := fmt.Sprintf("第%d章", targetNum)
		if branch != "" {
			label = fmt.Sprintf("第%d%s章", targetNum, branch)
		}
		chapterSummary := &types.ChapterSummary{Title: label, Summary: summary}
		var summaryErr error
		if branch != "" {
			summaryErr = pm.WriteChapterBranchSummary(targetNum, branch, chapterSummary)
		} else {
			summaryErr = pm.WriteChapterSummary(targetNum, chapterSummary)
		}
		if summaryErr != nil {
			slog.Warn("章节摘要落盘失败", "chapter", targetNum, "branch", branch, "error", summaryErr)
		}
	}

	// 更新节点摘要并保存（读-改-写 + 按 nodeID 定点合并，见 mergeChapterWriteBack）。
	//
	// 护栏（线C E2E 取证）：模型这次可能漏输出 ---CHAPTER_SUMMARY---，此时 summary 为空；
	// 若照写会清空节点既有摘要，而写前硬闸的 outline_summary_empty（S2）随即把**同一章**
	// 的重新生成拦下——作者看到的是「去补大纲摘要」，真实原因被掩盖。故摘要**非空才覆盖**
	// （同款纪律见线A mergeStoryThread），空/纯空白保留原值；Status 仍推进到 done
	// （本章正文已落盘），只有摘要不回退。回写只作用于目标节点，生成期间作者对
	// 其它节点/其它章的并发编辑以最新快照为准（G1）。
	a.mergeChapterWriteBack(pm, nodeID, summary, types.OutlineDone)
	if branch != "" {
		if err := pm.WriteChapterBranch(targetNum, branch, content); err != nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("save chapter: %v", err)})
			return
		}
	} else {
		if err := pm.WriteChapter(targetNum, content); err != nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("save chapter: %v", err)})
			return
		}
		// 整章重写=场景重置：旧拆分/旧 POV 元数据不再对应新正文，
		// 删除既有场景并从新 blob 物化单场景（无场景的章 no-op）。
		rebuildScenesFromBlob(pm, targetNum)
	}

	// AI 控制台响应日志
	_, respModel, _ := a.routeModel("novel")
	a.emit("xai-output", map[string]interface{}{
		"type":    "response",
		"model":   respModel,
		"content": content,
		"length":  len([]rune(content)),
	})

	// 生成完成：用 novelstyle 确定性引擎给本章打 AI 味分（0-100，越高越 AI 味），
	// 随 done 事件回传给前端，作者立即看到；失败不阻断 done（仅附空结果）。
	aiTaste := map[string]any{}
	if taste, terr := novelstyle.ScoreTextNoRef(content); terr == nil && taste != nil {
		novelstyle.ApplyWhitelist(taste, content, bookWhitelist(pm)) // 书级白名单豁免后再报分
		aiTaste = map[string]any{"score": taste.Score, "issues": taste.Issues}
	}
	if deSlopReport != nil {
		aiTaste["deSlop"] = deSlopReport
	}
	if len(aiTaste) == 0 {
		aiTaste = nil
	}

	a.emit("create-chapter-stream", map[string]interface{}{
		"type":       "done",
		"content":    content,
		"chapterNum": targetNum,
		"branch":     branch,
		"summary":    summary,
		"nodeId":     nodeID,
		"total":      len([]rune(content)),
		"aiTaste":    aiTaste,
	})

	// 异步提取章节角色 + 生成后自动门：两者共用章节生成的 WaitGroup（N7），
	// 取消/切书后不残留无主协程——chapterGenWG 的归零即「本章生成链全部收尾」，
	// waitGensDone 这类以 WG 为准的调用方才不会在清理中途提前放行。
	a.chapterGenWG.Add(1)
	go func() {
		defer a.chapterGenWG.Done()
		a.extractCharactersAfterChapter(pm, content, targetNum)
	}()

	// 生成后自动门（GenerationGate 闭环收口，规格 进度计划/gaea-gen-gate-closure-
	// 20260916.md）：确定性三路+分析路（V2 落盘/伏笔同步/记忆回填）异步过水；
	// 分支章在门内跳分析路（登记表是主线口径）。
	a.chapterGenWG.Add(1)
	go func() {
		defer a.chapterGenWG.Done()
		a.runAutoGateAfterGeneration(pm, targetNum, content, branch)
	}()
}

// resolveTargetChapterNum 计算「本章章号」：显式指定（>0）直接用；分支续写
// 取父节点章号；否则顺延为新章 = **全树最大 order_index + 1**。
// 旧实现 len(顶层节点)+1 有两种实弹事故：卷结构（卷一→第1章，len=1 → 算出
// 已存在的 1 反复覆盖）与缺节点跳章——磁盘大纲是唯一事实源，前端 store 同步
// 缺口不再影响章号。
// 分支父节点不存在时返回 0（调用方按无章号降级处理）。
func resolveTargetChapterNum(of *types.OutlineFile, chapterNum int, branchFromNodeID string) int {
	if chapterNum > 0 {
		return chapterNum
	}
	if branchFromNodeID != "" {
		for i := range of.Nodes {
			if of.Nodes[i].ID == branchFromNodeID {
				return of.Nodes[i].OrderIndex
			}
		}
		return 0
	}
	maxOrder := 0
	var walk func(nodes []types.OutlineNode)
	walk = func(nodes []types.OutlineNode) {
		for i := range nodes {
			if nodes[i].OrderIndex > maxOrder {
				maxOrder = nodes[i].OrderIndex
			}
			if len(nodes[i].Children) > 0 {
				walk(nodes[i].Children)
			}
		}
	}
	walk(of.Nodes)
	return maxOrder + 1
}

// pickCreateChapterTemplate 选章节正文生成模板：第一章是全书开篇，钩子铺设与
// 设定释放节奏和常规章节不同，走专用模板 create-chapter-first；专用模板缺失时
// 回落通用模板 create-chapter（旧磁盘 prompts/ 或测试桩只带通用模板也能跑）。
func pickCreateChapterTemplate(eng *prompt.Engine, chapterNum int) *prompt.Template {
	tmpl := eng.Get("create-chapter")
	if chapterNum == 1 {
		if first := eng.Get("create-chapter-first"); first != nil {
			tmpl = first
		}
	}
	return tmpl
}

// ensureChapterNode 确定章节号并创建/复用节点（同步，在 AI 生成前执行）。
//
// 新建节点时节点落盘失败返回 error（N8）：调用方据此中止本次生成，
// 绝不把「没写进 outline.json 的章」当成已建好继续生成正文。
func (a *writingState) ensureChapterNode(pm *project.Manager, of *types.OutlineFile, chapterNum int, branchFromNodeID string) (targetNum int, nodeID string, branch string, err error) {
	if chapterNum > 0 {
		targetNum = chapterNum
		for _, n := range of.Nodes {
			if n.OrderIndex == chapterNum && n.Branch == "" {
				return targetNum, n.ID, "", nil
			}
		}
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, Title: fmt.Sprintf("第%d章", targetNum),
			OrderIndex: targetNum, Status: types.OutlineWriting,
		})
		if werr := pm.WriteOutlines(of); werr != nil {
			return 0, "", "", fmt.Errorf("章节节点落盘失败（第%d章）：%w", targetNum, werr)
		}
		return targetNum, nodeID, "", nil
	}
	if branchFromNodeID != "" {
		var parent *types.OutlineNode
		for i := range of.Nodes {
			if of.Nodes[i].ID == branchFromNodeID {
				parent = &of.Nodes[i]
				break
			}
		}
		if parent == nil {
			return 0, "", "", nil
		}
		targetNum = parent.OrderIndex
		used := map[string]bool{}
		for _, n := range of.Nodes {
			if n.OrderIndex == targetNum && n.Branch != "" {
				used[n.Branch] = true
			}
		}
		for _, l := range []string{"a", "b", "c"} {
			if !used[l] {
				branch = l
				break
			}
		}
		if branch == "" {
			return 0, "", "", nil
		}
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, ParentID: branchFromNodeID,
			Title:      fmt.Sprintf("第%d%s章", targetNum, branch),
			OrderIndex: targetNum, Branch: branch,
			Status: types.OutlineWriting, ChapterFile: fmt.Sprintf("%03d%s.md", targetNum, branch),
		})
		if werr := pm.WriteOutlines(of); werr != nil {
			return 0, "", "", fmt.Errorf("分支节点落盘失败（第%d%s章）：%w", targetNum, branch, werr)
		}
		return targetNum, nodeID, branch, nil
	}
	targetNum = len(of.Nodes) + 1
	nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
	of.Nodes = append(of.Nodes, types.OutlineNode{
		ID: nodeID, Title: fmt.Sprintf("第%d章", targetNum),
		OrderIndex: targetNum, Status: types.OutlineWriting,
	})
	if werr := pm.WriteOutlines(of); werr != nil {
		return 0, "", "", fmt.Errorf("章节节点落盘失败（第%d章）：%w", targetNum, werr)
	}
	return targetNum, nodeID, "", nil
}

// mergeChapterWriteBack 把本次生成真正产生的事实（节点摘要 + done 状态）
// 定点合并回**最新**大纲快照并落盘，绝不整表回写生成开始时的旧快照。
//
// 为什么不能整表写回：一次流式生成持续数分钟，期间作者会在阅读页保存正文、
// 在大纲页增删节点/续写/改摘要（全部写同一个 outline.json）。生成收尾时拿
// 分钟前的快照覆盖，会静默回滚期间的全部编辑——删掉的节点复活、其它章状态
// 退回、摘要编辑丢失（G1）。故此处重新读盘取最新快照：
//   - 只写目标节点（nodeID 命中者）的 Summary 与 Status；
//   - 其余节点的一切字段（含期间新增/删除的节点、其它章 Status、KeyPoints、
//     Emotion）以最新快照为准，一律不被本次生成触碰；
//   - Summary 空/纯空白不覆盖（沿用既有的硬闸反噬护栏：模型漏输出摘要时若
//     写空会清掉节点摘要，写前硬闸随即把同一章拦下）；
//   - 目标节点已被并发删除时**不加回**，如实记日志并跳过本次回写
//     （宁可不写，不复活已删节点）。
func (a *writingState) mergeChapterWriteBack(pm *project.Manager, nodeID, summary string, status types.OutlineNodeStatus) {
	if pm == nil || nodeID == "" {
		return
	}
	latest, err := pm.ReadOutlines()
	if err != nil || latest == nil {
		slog.Warn("生成收尾：重读最新大纲失败，跳过本次大纲回写", "node", nodeID, "error", err)
		return
	}
	target := findOutlineNodeByID(latest.Nodes, nodeID)
	if target == nil {
		slog.Warn("生成收尾：目标节点在最新大纲中已不存在（生成期间被删除），跳过本次回写",
			"node", nodeID)
		return
	}
	if s := strings.TrimSpace(summary); s != "" {
		target.Summary = s
	}
	target.Status = status
	if werr := pm.WriteOutlines(latest); werr != nil {
		slog.Warn("生成收尾：大纲回写失败（本次生成事实未落盘）", "node", nodeID, "error", werr)
	}
}

// ── 章节生成上下文增强（伏笔 / 世界观 / 角色卡）──────────────────────
// 以下注入全部容错：文件缺失、JSON 损坏一律静默跳过对应区段，
// 绝不让上下文增强失败导致章节生成失败。

// 上下文注入预算（均为 rune 数）。各区段先按条目/维度截断，合计超过
// ctxBudgetTotal 时再逐段对半压缩，保证 prompt 不爆炸。// extractNewCharacters 从章节摘要中提取新角色，与项目角色 + 全局角色库对照去重。
// 返回两部分：
//
//	newChars       项目与角色库都没有的全新角色名
//	libraryMatches 角色库中已有同名角色（前端提示直接关联，不新建）

func (a *writingState) extractNewCharacters(pm *project.Manager, summary *types.ChapterSummary) (newChars []string, libraryMatches []characterlib.Character) {
	if summary == nil || len(summary.CharactersAppeared) == 0 {
		return nil, nil
	}

	// 1. 对照项目 characters.json 去重
	existingNames := make(map[string]bool)
	if cf, err := pm.ReadCharacters(); err == nil && cf != nil {
		for _, ch := range cf.Characters {
			existingNames[ch.Name] = true
		}
	}
	var unknown []string
	for _, name := range summary.CharactersAppeared {
		if name == "" || existingNames[name] {
			continue
		}
		unknown = append(unknown, name)
	}
	if len(unknown) == 0 {
		return nil, nil
	}

	// 2. 对照全局角色库去重：库内已有同名 → 走关联而非新建
	if a.app != nil && a.app.charLib != nil {
		var remain []string
		for _, name := range unknown {
			c, err := a.app.charLib.FindByName(name)
			if err != nil || c == nil {
				remain = append(remain, name)
				continue
			}
			libraryMatches = append(libraryMatches, *c)
		}
		unknown = remain
	}
	return unknown, libraryMatches
}

// extractCharactersAfterChapter 章节生成后异步提取角色
// 调用 chapter-summary AI → 提取 characters_appeared → 对照去重 → 通知前端
func (a *writingState) extractCharactersAfterChapter(pm *project.Manager, content string, chapterNum int) {
	if a.chapterAgent == nil {
		slog.Warn("extractCharactersAfterChapter: chapterAgent 未初始化")
		return
	}

	// 1. 调用 AI 提取结构化摘要（含 characters_appeared）
	slog.Info("开始提取章节角色", "chapter", chapterNum)
	summary, err := a.chapterAgent.GenerateSummary(a.ctx, content)
	if err != nil {
		slog.Warn("提取章节角色失败", "chapter", chapterNum, "error", err)
		return
	}

	// 2. 对照项目 + 角色库去重
	newChars, libMatches := a.extractNewCharacters(pm, summary)
	if len(newChars) == 0 && len(libMatches) == 0 {
		slog.Info("章节角色提取完成，无新角色", "chapter", chapterNum, "appeared", len(summary.CharactersAppeared))
		return
	}

	// 3. 通知前端发现新角色（全新名字 + 库内已有同名角色）
	payload := map[string]interface{}{
		"chapterNum": chapterNum,
		"characters": newChars,
	}
	if len(libMatches) > 0 {
		matches := make([]map[string]interface{}, 0, len(libMatches))
		for _, c := range libMatches {
			matches = append(matches, map[string]interface{}{
				"id":          c.ID,
				"name":        c.Name,
				"roleType":    c.RoleType,
				"portraitUrl": c.PortraitURL,
			})
		}
		payload["libraryMatches"] = matches
	}
	slog.Info("发现新角色", "chapter", chapterNum, "new", newChars, "libraryMatches", len(libMatches))
	a.emit("new-characters-discovered", payload)
}
