package app

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"
	"github.com/gaea/gaea/internal/gaea/wspath"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// ── 会话条目缓存（刀E v4.249，普查#5）────────────────────────────
// 上下文看板/节点详情/轨迹/Agent 网络（+ resync 外共五处）绑定共用
// ReadEntriesFor 全量读+逐行解析，前端轮询刷新时同份日志被反复解析——
// 会话越长看板越卡。事件日志是 append-only（单写入器整行追加），文件
// size+mtime 未变 ⇒ 内容未变；legacy 会话文件同理（投影只依赖该文件）。
// 缓存切片跨调用方只读共享（fold/detail/trajectory 层均不原地改写）。

type cachedEntries struct {
	key     string
	entries []session.LogEntry
}

var (
	entriesCacheMu sync.Mutex
	entriesCache   = map[string]cachedEntries{}
)

var (
	surfaceCacheMu sync.Mutex
	surfaceCache   = map[string]contextview.SurfaceSnapshot{}
	dnaCacheMu     sync.Mutex
	dnaCache       = map[string][]contextview.DnaSample{}
	commandPageMu  sync.Mutex
	commandPageMap = map[string]contextview.CommandPage{}
)

// readEntriesForCached 是 session.ReadEntriesFor 的缓存形态。key=日志文件与
// legacy 会话文件的 size+mtime 组合；缓存上限 8 个会话，超出整体清空（面板
// 场景同时活跃的会话数远小于此）。
func readEntriesForCached(sessionPath string) ([]session.LogEntry, error) {
	key, keyErr := entriesCacheKey(sessionPath)
	entriesCacheMu.Lock()
	if keyErr == nil {
		if c, ok := entriesCache[sessionPath]; ok && c.key == key {
			e := c.entries
			entriesCacheMu.Unlock()
			return e, nil
		}
	}
	entriesCacheMu.Unlock()

	entries, err := session.ReadEntriesFor(sessionPath)
	if err != nil {
		return nil, err
	}
	if keyErr == nil {
		entriesCacheMu.Lock()
		if len(entriesCache) > 8 {
			entriesCache = map[string]cachedEntries{}
		}
		entriesCache[sessionPath] = cachedEntries{key: key, entries: entries}
		entriesCacheMu.Unlock()
	}
	return entries, nil
}

func entriesCacheKey(sessionPath string) (string, error) {
	stat := func(p string) string {
		if st, err := os.Stat(p); err == nil {
			return fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())
		}
		return "-"
	}
	logPath := session.LogPathFor(sessionPath)
	if logPath == "" {
		return "", fmt.Errorf("empty log path")
	}
	return stat(logPath) + "|" + stat(sessionPath), nil
}

// GaeaContextView 返回会话的上下文构成快照（dsh-context Go 移植 Phase A）：
// 六分类当前组成、逐请求趋势、上下文事件、模型可见节点与归档。
// sessionPath 可选：显式传入=按该会话读取（UI 会话切换语义）；缺省=内核
// 当前会话（兼容旧调用）。会话或日志不存在时返回空快照（ok=true），不报错。
// 事件日志缺失时回退 legacy 会话投影（旧会话仍可看板，见 session.ReadEntriesFor）。
func (a *App) GaeaContextView(sessionPath ...string) (contextview.ContextTimeline, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return contextview.EmptyTimeline(), nil
	}
	entries, err := readEntriesForCached(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return contextview.EmptyTimeline(), nil
		}
		return contextview.EmptyTimeline(), err
	}
	window := gaeaContextWindow()
	tl := contextview.FoldTimeline(entries, int64(window), 0)
	tl.Compact = gaeaCompactTiers()
	return tl, nil
}

// GaeaContextNodeDetail 懒加载浏览器节点的「完整调用」详情（v4.80）：按 seq
// 回读会话日志（tool_result 配对 dispatch 取参数；user/assistant 取全文），
// 不随节点列表整包下发。sessionPath 可选，语义同 GaeaContextView。
func (a *App) GaeaContextNodeDetail(seq int64, sessionPath ...string) (contextview.NodeDetail, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return contextview.NodeDetail{}, fmt.Errorf("会话未就绪")
	}
	entries, err := readEntriesForCached(path)
	if err != nil {
		return contextview.NodeDetail{}, err
	}
	d, ok := contextview.NodeDetailFor(entries, seq)
	if !ok {
		return contextview.NodeDetail{}, fmt.Errorf("未找到 seq=%d 的可展开节点", seq)
	}
	resolveNodeImages(&d, gaeaCwd())
	resolveNodeAttachments(&d, gaeaCwd())
	return d, nil
}

// GaeaContextSurface 返回历史 request_header 时点的模型可见 surface。日志前缀
// 重折叠保证历史步骤不会被当前 live 节点冒充；entries cache key + seq 命中
// 前端趋势悬停/钉住预览，最多保留 32 份，文件变化后旧键自然失效。
func (a *App) GaeaContextSurface(seq int64, sessionPath ...string) (contextview.SurfaceSnapshot, error) {
	if seq <= 0 {
		return contextview.SurfaceSnapshot{}, fmt.Errorf("请求 seq 无效")
	}
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return contextview.SurfaceSnapshot{}, fmt.Errorf("会话未就绪")
	}
	version, err := entriesCacheKey(path)
	if err != nil {
		return contextview.SurfaceSnapshot{}, err
	}
	cacheKey := path + "\x00" + strconv.FormatInt(seq, 10) + "\x00" + version
	surfaceCacheMu.Lock()
	if hit, ok := surfaceCache[cacheKey]; ok {
		surfaceCacheMu.Unlock()
		return hit, nil
	}
	surfaceCacheMu.Unlock()

	entries, err := readEntriesForCached(path)
	if err != nil {
		return contextview.SurfaceSnapshot{}, err
	}
	snapshot, ok := contextview.SurfaceAt(entries, seq, int64(gaeaContextWindow()))
	if !ok {
		return contextview.SurfaceSnapshot{}, fmt.Errorf("未找到 seq=%d 的历史请求 surface", seq)
	}
	surfaceCacheMu.Lock()
	if len(surfaceCache) >= 32 {
		surfaceCache = map[string]contextview.SurfaceSnapshot{}
	}
	surfaceCache[cacheKey] = snapshot
	surfaceCacheMu.Unlock()
	return snapshot, nil
}

// GaeaContextDna 批量返回多个历史 request_header 的趋势 DNA 带。单次日志
// 遍历折叠，最多 128 个 seq；前端只请求横向可见窗口及前一步，避免逐请求
// 重扫整份日志。缓存键含日志 size+mtime，日志变化后自然失效。
func (a *App) GaeaContextDna(seqs []int64, sessionPath ...string) ([]contextview.DnaSample, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return nil, fmt.Errorf("会话未就绪")
	}
	unique := make([]int64, 0, len(seqs))
	seen := map[int64]struct{}{}
	for _, seq := range seqs {
		if seq <= 0 {
			continue
		}
		if _, ok := seen[seq]; ok {
			continue
		}
		seen[seq] = struct{}{}
		unique = append(unique, seq)
		if len(unique) >= 128 {
			break
		}
	}
	if len(unique) == 0 {
		return []contextview.DnaSample{}, nil
	}
	cacheSeqs := append([]int64(nil), unique...)
	sort.Slice(cacheSeqs, func(i, j int) bool { return cacheSeqs[i] < cacheSeqs[j] })
	version, err := entriesCacheKey(path)
	if err != nil {
		return nil, err
	}
	var keyBuilder strings.Builder
	keyBuilder.WriteString(path)
	keyBuilder.WriteByte(0)
	keyBuilder.WriteString(version)
	for _, seq := range cacheSeqs {
		keyBuilder.WriteByte(0)
		keyBuilder.WriteString(strconv.FormatInt(seq, 10))
	}
	cacheKey := keyBuilder.String()
	dnaCacheMu.Lock()
	if hit, ok := dnaCache[cacheKey]; ok {
		dnaCacheMu.Unlock()
		bySeq := make(map[int64]contextview.DnaSample, len(hit))
		for _, sample := range hit {
			bySeq[sample.Seq] = sample
		}
		out := make([]contextview.DnaSample, 0, len(unique))
		for _, seq := range unique {
			sample, ok := bySeq[seq]
			if !ok {
				sample = contextview.DnaSample{Seq: seq, Bands: []contextview.DnaBand{}}
			}
			out = append(out, sample)
		}
		return out, nil
	}
	dnaCacheMu.Unlock()

	entries, err := readEntriesForCached(path)
	if err != nil {
		return nil, err
	}
	samples := contextview.DnaAt(entries, unique, int64(gaeaContextWindow()))
	dnaCacheMu.Lock()
	if len(dnaCache) >= 32 {
		dnaCache = map[string][]contextview.DnaSample{}
	}
	dnaCache[cacheKey] = append([]contextview.DnaSample(nil), samples...)
	dnaCacheMu.Unlock()
	return samples, nil
}

// GaeaContextCommands 分页读取 400 条常规窗口之外的更早命令历史。
// beforeSeq 为排他游标；limit 缺省/非法由 contextview 层收口到 200。
func (a *App) GaeaContextCommands(beforeSeq int64, limit int, sessionPath ...string) (contextview.CommandPage, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return contextview.CommandPage{}, fmt.Errorf("会话未就绪")
	}
	version, err := entriesCacheKey(path)
	if err != nil {
		return contextview.CommandPage{}, err
	}
	cacheKey := path + "\x00" + version + "\x00" + strconv.FormatInt(beforeSeq, 10) + "\x00" + strconv.Itoa(limit)
	commandPageMu.Lock()
	if hit, ok := commandPageMap[cacheKey]; ok {
		commandPageMu.Unlock()
		return hit, nil
	}
	commandPageMu.Unlock()
	entries, err := readEntriesForCached(path)
	if err != nil {
		return contextview.CommandPage{}, err
	}
	page := contextview.CommandPageAt(entries, beforeSeq, limit)
	commandPageMu.Lock()
	if len(commandPageMap) >= 32 {
		commandPageMap = map[string]contextview.CommandPage{}
	}
	commandPageMap[cacheKey] = page
	commandPageMu.Unlock()
	return page, nil
}

// resolveNodeImages 把详情里的图片引用解析为缩略卡数据（2.5b 后半）：相对
// 引用按 gaea cwd 解析为绝对路径；文件存在则仅解码头部取尺寸，并按官方
// patch 口径（⌈w/28⌉×⌈h/28⌉，先档位缩放再封顶）估算 token。文件缺失/
// 解码失败诚实降级（Exists=false / 尺寸留零），绝不阻塞详情主体。
func resolveNodeImages(d *contextview.NodeDetail, cwd string) {
	if len(d.ImageRefs) == 0 {
		return
	}
	images := make([]contextview.NodeImage, 0, len(d.ImageRefs))
	for _, ref := range d.ImageRefs {
		img := contextview.NodeImage{Ref: ref, Path: ref}
		if !filepath.IsAbs(ref) && cwd != "" {
			img.Path = filepath.Join(cwd, ref)
			img.RefCwd = cwd
		}
		f, err := os.Open(img.Path)
		if err != nil {
			images = append(images, img) // Exists=false：前端灰态「文件不存在」
			continue
		}
		cfg, _, err := image.DecodeConfig(f)
		_ = f.Close()
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			// 文件在但尺寸未知（svg/ico 等非栅格或不受支持格式）。
			img.Exists = true
			images = append(images, img)
			continue
		}
		img.Exists = true
		img.Width, img.Height = cfg.Width, cfg.Height
		est := contextview.EstimateImageTokens(cfg.Width, cfg.Height)
		img.ScaledW, img.ScaledH = est.ScaledW, est.ScaledH
		img.StdTokens, img.HighTokens = est.StdTokens, est.HighTokens
		images = append(images, img)
	}
	d.Images = images
}

// resolveNodeAttachments 只解析持久化上传/附件目录中的非图片文件。路径必须
// 落在当前工作区内；越界/缺失如实 Exists=false，不读取文件正文，也不伪造
// MIME/大小。文件名来自存储路径（上传时已经过安全化）。
func resolveNodeAttachments(d *contextview.NodeDetail, cwd string) {
	if len(d.AttachmentRefs) == 0 {
		return
	}
	out := make([]contextview.NodeAttachment, 0, len(d.AttachmentRefs))
	for _, ref := range d.AttachmentRefs {
		path := ref
		if !filepath.IsAbs(path) && cwd != "" {
			path = filepath.Join(cwd, filepath.FromSlash(path))
		}
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
		item := contextview.NodeAttachment{
			Ref: ref, Path: path, Name: filepath.Base(path),
			MimeType: attachmentMIME(path),
		}
		if err == nil && cwd != "" && wspath.Within(cwd, path) {
			if stat, statErr := os.Stat(path); statErr == nil && stat.Mode().IsRegular() {
				item.Exists = true
				item.Size = stat.Size()
			}
		}
		out = append(out, item)
	}
	d.Attachments = out
}

func attachmentMIME(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if typ := mime.TypeByExtension(ext); typ != "" {
		return typ
	}
	switch ext {
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	}
	return "application/octet-stream"
}

// resolveGaeaSessionPath 归一会话路径：显式传入（UI 正在查看的会话）优先，
// 空/缺省回退内核当前会话（兼容旧调用）。UI 会话切换后内核 ctrl 仍指向最近
// 活跃会话——上下文/轨迹/网络按 UI 会话展示必须显式传路径，否则切到历史
// 会话时看到的是内核会话的固定内容。
func resolveGaeaSessionPath(args []string) string {
	for _, p := range args {
		if s := strings.TrimSpace(p); s != "" {
			// 显式路径须落在会话目录内（2026-09-19 审计 P2：同族
			// GaeaDeleteSession/GaeaSessionStats 均过 sessionDirForPath 校验，
			// 三个只读看板口漏了——任意目录下符合命名约定的 JSONL 可被解析
			// 回传）。非法视为未传，走内核回退/空快照的既有分支。
			if sessionDirForPath(s) == "" {
				return ""
			}
			return s
		}
	}
	if c := gaeaCtrl(); c != nil {
		return c.SessionPath()
	}
	return ""
}

// GaeaTrajectory 返回会话的轨迹时间线（dsh 轨迹标签的 Go 移植）：
// 按 轮次 → 步骤 组织用户输入、推理、回复、工具调用与过程事件。
// sessionPath 可选：显式传入=按该会话读取（UI 会话切换语义）；缺省=内核
// 当前会话（兼容旧调用）。会话或日志不存在时返回空快照（ok=true），不报错。
// 事件日志缺失时回退 legacy 会话投影（旧会话仍可看板，见 session.ReadEntriesFor）。
func (a *App) GaeaTrajectory(sessionPath ...string) (trajectory.Trajectory, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return trajectory.EmptyTrajectory(), nil
	}
	entries, err := readEntriesForCached(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trajectory.EmptyTrajectory(), nil
		}
		return trajectory.EmptyTrajectory(), err
	}
	return trajectory.FoldTrajectory(entries), nil
}

// GaeaAgentNetwork 返回会话的 Agent 网络（主 agent 根 + 子代理树），并用
// subagents/ meta 富化子代理节点的任务摘要/状态/模型。sessionPath 可选：
// 显式传入=按该会话读取（UI 会话切换语义）；缺省=内核当前会话（兼容旧调用）。
// 事件日志缺失时回退 legacy 会话投影（见 session.ReadEntriesFor）。
func (a *App) GaeaAgentNetwork(sessionPath ...string) (trajectory.AgentNetwork, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return trajectory.EmptyAgentNetwork(), nil
	}
	entries, err := readEntriesForCached(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trajectory.EmptyAgentNetwork(), nil
		}
		return trajectory.EmptyAgentNetwork(), err
	}
	window := gaeaContextWindow()
	net := trajectory.FoldAgentNetwork(entries, int64(window))
	enrichAgentNetwork(&net, a.GaeaSubagentRuns(path))
	return net, nil
}

// gaeaContextWindow 取内核上下文窗口大小（窗口属内核配置与会话无关；内核
// 未初始化时返回 0，折叠管线自行兜底）。
func gaeaContextWindow() int {
	c := gaeaCtrl()
	if c == nil {
		return 0
	}
	_, window := c.ContextSnapshot()
	return window
}

// gaeaCompactTiers 取内核生效的三层压缩水位（同属内核配置与会话无关）。
// 内核未初始化时返回 nil——前端退回静态阈值呈现，不伪造水位。
func gaeaCompactTiers() *contextview.CompactTiers {
	c := gaeaCtrl()
	if c == nil {
		return nil
	}
	soft, ratio, force := c.CompactionTiers()
	if ratio <= 0 || soft <= 0 || force <= 0 {
		return nil
	}
	return &contextview.CompactTiers{Soft: soft, Ratio: ratio, Force: force}
}

// runMatchesNode 判定一个已落盘 run 是否已被树上的节点承载：
// ref 直等（合成节点 id=sa_ ref），或任务摘要前缀双向（与前端
// AgentTree.matchRunForNode / 后端富化同口径）。
func runMatchesNode(r SubagentRunView, node trajectory.AgentNode) bool {
	if r.Ref != "" && r.Ref == node.ID {
		return true
	}
	return node.Task != "" && r.Task != "" &&
		(strings.HasPrefix(r.Task, node.Task) || strings.HasPrefix(node.Task, r.Task))
}

// enrichAgentNetwork 把 subagents/ meta 合并进 Agent 网络子代理节点：
// 按任务摘要匹配（子代理 transcript 首条 user 消息 ≈ task 调用的 prompt 预览）。
// 树来自事件日志折叠，「零工具调用」的子代理（纯调研/禁用工具）在日志里
// 没有子记录、FoldAgentNetwork 不为其建节点——这里对没有任何节点承载的
// run 补挂合成节点（id=sa_ ref），保证任何已落盘运行都在树上可见。
func enrichAgentNetwork(net *trajectory.AgentNetwork, runs SubagentRunsView) {
	if !runs.Available || len(runs.Runs) == 0 {
		return
	}
	matched := make([]bool, len(runs.Runs))
	for i := range net.Root.Children {
		node := &net.Root.Children[i]
		for j, r := range runs.Runs {
			if runMatchesNode(r, *node) {
				node.Status = r.Status
				node.Model = r.Model
				if node.Task == "" {
					node.Task = r.Task
				}
				matched[j] = true
				break
			}
		}
	}
	for j, r := range runs.Runs {
		if matched[j] || r.Kind == "model_tool" {
			continue
		}
		status := r.Status
		if status == "failed" {
			status = "error"
		} else if status != "running" {
			status = "completed"
		}
		net.Root.Children = append(net.Root.Children, trajectory.AgentNode{
			ID:        r.Ref,
			Name:      r.Task,
			Kind:      "subagent",
			Status:    status,
			Model:     r.Model,
			Task:      r.Task,
			ToolCalls: r.ToolCalls,
			FirstTs:   r.CreatedAt.Unix(),
			LastTs:    r.UpdatedAt.Unix(),
		})
	}
}

// GaeaSubagentContextView 返回指定子代理（sa_ ref）会话的上下文构成快照
// （2.5e 后半：Agent 网络节点 → 子代理上下文跳转）。与 GaeaContextView
// 同一折叠管线，仅数据源换成子代理 transcript；ref 非法/transcript 缺失
// 诚实报错。
func (a *App) GaeaSubagentContextView(sessionPath, ref string) (contextview.ContextTimeline, error) {
	if !gaeaAgent.ValidRunRef(ref) {
		return contextview.ContextTimeline{}, fmt.Errorf("非法的子代理引用 %q", ref)
	}
	dir := sessionDirForPath(sessionPath)
	if dir == "" {
		return contextview.ContextTimeline{}, fmt.Errorf("会话路径无法定位目录")
	}
	transcript := filepath.Join(dir, "subagents", ref+".jsonl")
	// 子代理 transcript 是独立 jsonl（无 legacy 投影语义），直接读修复管道。
	entries, err := session.ReadLogRepaired(transcript)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return contextview.EmptyTimeline(), nil
		}
		return contextview.ContextTimeline{}, err
	}
	tl := contextview.FoldTimeline(entries, 0, 0)
	return tl, nil
}

// ── 跨会话洞察（刀2 v4.542，对标 dsh-context Context Insights）──────
// 扫描工作区全部会话日志折叠汇总。缓存按「文件 size+mtime」失效——
// 会话日志 append-only，未变文件复用上次折叠结果（dsh 缓存轻量化同思路：
// 摘要常驻、变更重算）。子代理 transcript（<dir>/subagents/）不入扫描面：
// 其 token 已属对应主会话派生行为，洞察口径=主会话（家族视图在编队页）。

type cachedInsightsFold struct {
	key  string
	fold contextview.InsightsSessionFold
}

var (
	insightsCacheMu sync.Mutex
	insightsCache   = map[string]cachedInsightsFold{}
)

// maxInsightsSessions 是单次洞察扫描的会话上限（最近修改优先；超出诚实
// 计入 Truncated，不静默吞掉）。
const maxInsightsSessions = 300

// insightsCacheKey 是失效键：size+mtime（append-only 日志的充分条件）。
func insightsCacheKey(path string) string {
	st, err := os.Stat(path)
	if err != nil {
		return "-"
	}
	return fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())
}

// GaeaContextInsights 返回跨会话洞察快照。rangeKey: today|7d|30d|all
// （未知按 30d 兜底）；includeArchived 时纳入 archive/ 会话。
// 文件读取失败的单会话跳过（不拖垮整页；错误面在日志，诚实少一卡）。
func (a *App) GaeaContextInsights(rangeKey string, includeArchived bool) (contextview.ContextInsights, error) {
	base := gaeaConfig.WorkspaceSessionDir(gaeaCwd(), "")
	infos, err := gaeaAgent.ListSessions(base)
	if err != nil {
		infos = nil
	}
	if includeArchived {
		if arc, err := gaeaAgent.ListArchivedSessions(base); err == nil {
			infos = append(infos, arc...)
		}
	}
	// 最近修改优先截断（List 已按 ModTime 新→旧；archived 追加段重新排序）。
	sort.SliceStable(infos, func(i, j int) bool { return infos[i].ModTime.After(infos[j].ModTime) })
	truncated := 0
	if len(infos) > maxInsightsSessions {
		truncated = len(infos) - maxInsightsSessions
		infos = infos[:maxInsightsSessions]
	}
	titles := loadSessionTitlesAll(base)
	folds := make([]contextview.InsightsSessionFold, 0, len(infos))
	for _, s := range infos {
		key := insightsCacheKey(s.Path)
		insightsCacheMu.Lock()
		c, ok := insightsCache[s.Path]
		insightsCacheMu.Unlock()
		if !ok || c.key != key {
			entries, err := readEntriesForCached(s.Path)
			if err != nil {
				continue // 单会话读取失败：诚实跳过
			}
			c = cachedInsightsFold{
				key: key,
				fold: contextview.InsightsSessionFold{
					Path:     s.Path,
					Preview:  s.Preview,
					SpaceID:  s.Space,
					Archived: strings.Contains(filepath.ToSlash(s.Path), "/archive/") || strings.Contains(filepath.ToSlash(s.Path), "\archive\\"),
					ModTime:  s.ModTime.Unix(),
					Timeline: contextview.FoldTimeline(entries, 0, 1),
				},
			}
			if t := titles[filepath.Base(s.Path)]; t != "" {
				c.fold.Title = t
			}
			insightsCacheMu.Lock()
			if len(insightsCache) > maxInsightsSessions {
				insightsCache = map[string]cachedInsightsFold{}
			}
			insightsCache[s.Path] = c
			insightsCacheMu.Unlock()
		}
		folds = append(folds, c.fold)
	}
	out := contextview.BuildInsights(rangeKey, time.Now(), folds)
	out.Truncated = truncated
	return out, nil
}

// ── 家族事件流（v4.542，对标 dsh-context Fleet 事件检查完全体）──────
// 主会话 + 全部子代理 transcript 折叠为合并事件流。transcript 读取独立
// 缓存（size+mtime 失效，同 insights 纪律）；上限 20 份 transcript 诚实
// 截断（Truncated 返回）。

type cachedFamilyEntries struct {
	key     string
	entries []session.LogEntry
}

var (
	familyCacheMu sync.Mutex
	familyCache   = map[string]cachedFamilyEntries{}
)

const maxFamilyTranscripts = 20

// GaeaFamilyEvents 返回会话的家族事件流（main + 子代理 transcript 合并，
// Ts 升序，上限 400 条）。sessionPath 可选，语义同 GaeaContextView。
func (a *App) GaeaFamilyEvents(sessionPath ...string) (trajectory.FamilyEventStream, error) {
	path := resolveGaeaSessionPath(sessionPath)
	if path == "" {
		return trajectory.EmptyFamilyEventStream(), nil
	}
	parentEntries, err := readEntriesForCached(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trajectory.EmptyFamilyEventStream(), nil
		}
		return trajectory.EmptyFamilyEventStream(), err
	}
	// 子代理 transcript 枚举：<会话目录>/subagents/*.jsonl
	dir := sessionDirForPath(path)
	subs := map[string][]session.LogEntry{}
	truncated := 0
	if dir != "" {
		matches, _ := filepath.Glob(filepath.Join(dir, "subagents", "*.jsonl"))
		sort.Slice(matches, func(i, j int) bool { return matches[i] > matches[j] }) // 名字含时间戳，新→旧
		if len(matches) > maxFamilyTranscripts {
			truncated = len(matches) - maxFamilyTranscripts
			matches = matches[:maxFamilyTranscripts]
		}
		for _, transcript := range matches {
			ref := strings.TrimSuffix(filepath.Base(transcript), ".jsonl")
			key := insightsCacheKey(transcript)
			familyCacheMu.Lock()
			c, ok := familyCache[transcript]
			familyCacheMu.Unlock()
			if !ok || c.key != key {
				entries, err := session.ReadLogRepaired(transcript)
				if err != nil {
					continue // 单份读取失败诚实跳过
				}
				c = cachedFamilyEntries{key: key, entries: entries}
				familyCacheMu.Lock()
				if len(familyCache) > maxFamilyTranscripts*2 {
					familyCache = map[string]cachedFamilyEntries{}
				}
				familyCache[transcript] = c
				familyCacheMu.Unlock()
			}
			subs[ref] = c.entries
		}
	}
	events := trajectory.FoldFamilyEvents(parentEntries, subs)
	totals := trajectory.FamilyTotals{SubCount: len(subs)}
	totals.RootTokens, totals.RootCost, totals.OutputTokens = trajectory.UsageOf(parentEntries)
	for _, se := range subs {
		t, c, o := trajectory.UsageOf(se)
		totals.SubTokens += t
		totals.SubCost += c
		totals.OutputTokens += o
	}
	return trajectory.FamilyEventStream{
		Ok:        true,
		Events:    events,
		Scanned:   len(subs) + 1,
		Truncated: truncated,
		Refs:      trajectory.FamilyEventRefs(events),
		Totals:    totals,
		Agents:    trajectory.FamilyAgentRows(parentEntries, subs),
	}, nil
}
