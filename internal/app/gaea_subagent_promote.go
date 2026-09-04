package app

// GaeaPromoteSubagent —— 子代理会话 tab「保存为新会话」（v4.68，dsh Side Chat
// 的 promote 语义）：把一次子代理运行（<sessionDir>/subagents/<ref>.jsonl）
// 复制提升为一个独立顶层会话，可从会话列表打开、可继续对话。
//
// 语义要点：
//   - 复制而非移动：原子代理运行与其 meta sidecar 完全不动（只读源文件）；
//   - 每次提升都产生全新副本（不复用旧提升结果）。理由：提升是「此刻快照」，
//     子代理之后可能经追问（RunFollowUp）继续增长，重复提升应捕捉最新状态；
//     且两个副本各自独立续聊互不干扰——复用同一目标会反而制造隐式别名；
//   - 忠实投影：transcript（provider.Message JSONL）转换为主会话事件日志能
//     投影（ProjectMessages）的标准条目（session.ToLogEntries 单点转换，带
//     turn 边界与合成 request_header）；写盘前后各做一次投影往返校验，
//     不等价则不落盘/清理落盘物并报错——绝不写出会破坏续聊的日志；
//   - 诚实降级：无法忠实投影的内容（孤立/重复 tool 结果、未响应的工具调用、
//     子代理 system 提示）按 sanitize 策略丢弃并 slog 说明，绝不伪造记录。

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// GaeaPromoteSubagent 把 ref 指定的子代理运行提升为新顶层会话，返回新
// sessionPath。sessionPath 是子代理所属会话（经 sessionDirForPath 校验，防
// 穿越）；ref 必须是 sa_ 前缀的子代理引用（mt_ 本地模型工具记录不是可续聊
// 会话，明确拒绝）。运行中（meta.Status=running）的子代理拒绝提升——其
// transcript 是移动快照，提升半程内容会误成一个「看起来完整」的会话。
// meta 缺失时按 completed 容错放行（与 GaeaSubagentTranscript 的读端容忍一致，
// transcript 本身即提升内容的真相源）。
func (a *App) GaeaPromoteSubagent(sessionPath, ref string) (string, error) {
	if sessionPath == "" || sessionDirForPath(sessionPath) == "" {
		return "", fmt.Errorf("非法会话路径: %s", sessionPath)
	}
	if !gaeaAgent.ValidRunRef(ref) || !gaeaAgent.IsSubagentRef(ref) {
		return "", fmt.Errorf("仅子代理运行（sa_ 前缀）可提升: %q", ref)
	}

	// 源：transcript 与 meta sidecar（与 GaeaSubagentTranscript 同点取兄弟目录，
	// archive 中的会话其 subagents/ 也随 archive 目录）。
	srcDir := filepath.Dir(filepath.Clean(sessionPath))
	transcriptPath := filepath.Join(srcDir, "subagents", ref+".jsonl")

	// 目标目录：同空间的会话目录族成员；archive 内的提升落到其所属空间目录。
	targetDir := sessionDirForPath(sessionPath)
	if filepath.Base(targetDir) == "archive" {
		targetDir = filepath.Dir(targetDir)
	}

	// meta：状态守卫 + 标题来源。缺失按 completed 容错（读端容忍先例）。
	metaStatus, metaTitle, metaModel := "", "", ""
	if b, err := os.ReadFile(filepath.Join(srcDir, "subagents", ref+".meta.json")); err == nil {
		var m struct {
			Status string `json:"status"`
			Title  string `json:"title"`
			Model  string `json:"model"`
		}
		if json.Unmarshal(b, &m) == nil {
			metaStatus, metaTitle, metaModel = m.Status, m.Title, m.Model
		}
	}
	if metaStatus == string(gaeaAgent.SubagentRunning) {
		return "", fmt.Errorf("子代理 %s 仍在运行，结束后再提升", ref)
	}

	// 读 transcript（provider.Message JSONL，与 session.Save 落盘格式一致）。
	msgs, err := session.Load(transcriptPath)
	if err != nil {
		return "", fmt.Errorf("读取子代理 transcript 失败: %w", err)
	}

	// 忠实投影前的诚实降级（策略注记返回给 slog，见文件头）。
	clean, notes := sanitizePromotedMessages(msgs.Messages)
	if !promoteHasUserMessage(clean) {
		return "", errors.New("子代理 transcript 无可提升内容（无用户消息）")
	}

	// 新会话：唯一 id（文件名时间戳 + 冲突重试），标题取任务摘要。
	newPath := uniquePromotedSessionPath(targetDir, promotedModelLabel(metaModel))
	title := promoteTitle(metaTitle, clean)

	// 事件日志条目（ToLogEntries 单点：turn 边界 + 合成 request_header），
	// 写盘前先做投影往返校验——不等价即拒绝（fail-closed，不落半截日志）。
	entries := session.ToLogEntries(clean)
	if got := session.ProjectMessages(entries); !promoteMessagesEqual(got, clean) {
		return "", errors.New("投影往返校验失败（转录内容无法忠实投影为会话日志），已放弃写入")
	}

	// 空间自描述与控制器 writeSpaceFor 同规则：play 分区恒 play（防恢复空间
	// 校验拒绝）；work/平铺按 space.mode——on 写 work，off 不写字段（旧行为形态）。
	writeSpace := promotedWriteSpace(newPath)

	// 落盘：事件日志 + legacy 镜像 JSONL（saveEventMode 同款双写——列表发现
	// （preview/turns）与 legacy 模式恢复读镜像，事件模式恢复读日志）。
	logPath := session.LogPathFor(newPath)
	w, err := session.OpenLog(logPath, "", writeSpace)
	if err != nil {
		return "", fmt.Errorf("创建会话日志失败: %w", err)
	}
	for _, e := range entries {
		if _, err := w.AppendRaw(e.Kind, e.Payload); err != nil {
			w.Close()
			removePromotedSessionFiles(newPath)
			return "", fmt.Errorf("写会话日志失败: %w", err)
		}
	}
	if err := w.Close(); err != nil {
		removePromotedSessionFiles(newPath)
		return "", fmt.Errorf("关闭会话日志失败: %w", err)
	}
	mirror := session.NewFromRestore(clean, 0, "")
	if err := mirror.Save(newPath); err != nil {
		removePromotedSessionFiles(newPath)
		return "", fmt.Errorf("写会话镜像失败: %w", err)
	}

	// 落盘后复验：按真实恢复链（Restore = checkpoint 缺省 + 日志全量重放投影）
	// 读回，不等价则清理本次产物并报错——绝不留下会破坏续聊的日志。
	restored, _, err := session.Restore(session.CheckpointPathFor(newPath), logPath)
	if err != nil || !promoteMessagesEqual(restored, clean) {
		removePromotedSessionFiles(newPath)
		return "", fmt.Errorf("恢复校验失败（err=%v），已清理本次提升产物", err)
	}

	// 标题注册表（<dir>/.titles.json，与 GaeaRenameSession 同点）。
	if err := setSessionTitle(targetDir, newPath, title); err != nil {
		// 标题注册失败不回滚会话本体（列表仍可见，只是无自定义标题）。
		slog.Warn("子代理提升：标题注册失败", "ref", ref, "error", err)
	}
	if len(notes) > 0 {
		// 诚实降级说明：丢什么、为什么（策略见 sanitizePromotedMessages）。
		slog.Info("子代理提升完成（含降级策略）", "ref", ref, "newSession", newPath,
			"policy", strings.Join(notes, "; "))
	}
	return newPath, nil
}

// sanitizePromotedMessages 把子代理 transcript（provider.Message 流）清洗为
// 可忠实投影的消息序列。策略（每条降级都记入 notes，绝不静默伪造）：
//  1. system 消息不随迁——子代理 system 提示描述的是「父代理派生的单任务
//     工具人」身份，提升后的顶层会话恢复时由运行时注入自己的系统提示，
//     随迁只会双重注入污染上下文；
//  2. 丢弃孤立 tool 结果（ToolCallID 为空或对不上任何前置 assistant 工具
//     调用）与重复 tool 结果——投影后喂模型会被 provider 400；
//  3. 剥离未获得 tool 结果响应的工具调用（中断/崩溃残留）：保留 assistant
//     正文与 reasoning，仅去掉调用本身；若剥后消息完全为空则整条丢弃——
//     「有 tool_calls 无 tool 结果」的回合是续聊 400 的头号来源；
//  4. 丢弃空 assistant 占位（无正文/无 reasoning/无调用）。
func sanitizePromotedMessages(msgs []provider.Message) (clean []provider.Message, notes []string) {
	seen := map[string]bool{} // 全部 assistant 工具调用 id（含后面才响应的）
	for _, m := range msgs {
		if m.Role != provider.RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if id := strings.TrimSpace(tc.ID); id != "" {
				seen[id] = true
			}
		}
	}

	var (
		dropSystem, dropTool, dropEmpty int
		strippedCalls                   int
		answered                        = map[string]bool{}
	)
	for _, m := range msgs {
		switch m.Role {
		case provider.RoleSystem:
			dropSystem++
			continue
		case provider.RoleTool:
			id := strings.TrimSpace(m.ToolCallID)
			if id == "" || !seen[id] {
				dropTool++ // 孤立结果
				continue
			}
			if answered[id] {
				dropTool++ // 同 id 重复结果
				continue
			}
			answered[id] = true
		}
		clean = append(clean, m)
	}

	out := clean[:0:0]
	for _, m := range clean {
		if m.Role == provider.RoleAssistant && len(m.ToolCalls) > 0 {
			kept := make([]provider.ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				id := strings.TrimSpace(tc.ID)
				if id == "" || !answered[id] {
					strippedCalls++ // 未响应（或无法配对）的调用
					continue
				}
				kept = append(kept, tc)
			}
			m.ToolCalls = kept
		}
		if promoteMessageEmpty(m) {
			dropEmpty++
			continue
		}
		out = append(out, m)
	}
	clean = out

	if dropSystem > 0 {
		notes = append(notes, fmt.Sprintf("丢弃 system 提示 %d 条（提升会话由运行时注入自己的系统提示）", dropSystem))
	}
	if dropTool > 0 {
		notes = append(notes, fmt.Sprintf("丢弃孤立/重复 tool 结果 %d 条（无配对调用，投影后无法续聊）", dropTool))
	}
	if strippedCalls > 0 {
		notes = append(notes, fmt.Sprintf("剥离未响应的工具调用 %d 个（中断残留，仅保留正文与 reasoning）", strippedCalls))
	}
	if dropEmpty > 0 {
		notes = append(notes, fmt.Sprintf("丢弃空 assistant 占位 %d 条", dropEmpty))
	}
	return clean, notes
}

// promoteMessageEmpty 报告一条消息是否不含任何可投影内容。
func promoteMessageEmpty(m provider.Message) bool {
	return strings.TrimSpace(m.Content) == "" &&
		strings.TrimSpace(m.ReasoningContent) == "" &&
		strings.TrimSpace(m.ReasoningSignature) == "" &&
		strings.TrimSpace(m.ToolCallID) == "" &&
		strings.TrimSpace(m.Name) == "" &&
		len(m.ToolCalls) == 0
}

// promoteHasUserMessage 报告清洗后的序列至少含一条非空 user 消息（提升的
// 会话必须在列表里可识别、可续聊）。
func promoteHasUserMessage(msgs []provider.Message) bool {
	for _, m := range msgs {
		if m.Role == provider.RoleUser && strings.TrimSpace(m.Content) != "" {
			return true
		}
	}
	return false
}

// promoteTitle 决定提升会话标题：meta.Title（任务摘要）优先，回退 transcript
// 首条 user 消息；超长按现有标题规则截断（truncateRunes 120，与子代理面板
// 任务摘要同一规则）。仍为空则返回空（列表回退 preview 展示，不编造标题）。
func promoteTitle(metaTitle string, msgs []provider.Message) string {
	title := strings.TrimSpace(metaTitle)
	if title == "" {
		for _, m := range msgs {
			if m.Role == provider.RoleUser && strings.TrimSpace(m.Content) != "" {
				title = strings.TrimSpace(m.Content)
				break
			}
		}
	}
	return truncateRunes(title, 120)
}

// promotedModelLabel 返回新会话文件名的模型标签（NewPath 语义：文件名暗示
// 对话所用模型）：meta.Model 优先，无则回退 "gaea"（与运行期 label 一致）。
func promotedModelLabel(metaModel string) string {
	if s := strings.TrimSpace(metaModel); s != "" {
		return s
	}
	return "gaea"
}

// promotedWriteSpace 按会话路径归属折算日志行空间自描述（与控制器
// writeSpaceForLocked 同规则；路径是唯一真相源）。
func promotedWriteSpace(sessionPath string) string {
	if session.SpaceForPath(sessionPath) == spaces.SpacePlay {
		return spaces.SpacePlay
	}
	if cfg := gaeaCfgSnapshot(); cfg != nil && cfg.SpaceModeIsOn() {
		return spaces.SpaceWork
	}
	return "" // space.mode=off：平铺日志不写 space 字段（旧行为形态）
}

// uniquePromotedSessionPath 生成不与现有产物冲突的新会话路径：NewPath 的
// 纳秒时间戳本身几乎不可能撞名，仍对 <p>/.gaea-log/.gaea-checkpoint/.state/
// .meta 全族做存在性检查，撞名时追加序号重试，兜底随机后缀。
func uniquePromotedSessionPath(dir, model string) string {
	base := gaeaAgent.NewSessionPath(dir, model)
	for i := 0; i < 64; i++ {
		p := base
		if i > 0 {
			p = strings.TrimSuffix(base, ".jsonl") + "-" + strconv.Itoa(i) + ".jsonl"
		}
		if !promotedPathTaken(p) {
			return p
		}
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.TrimSuffix(base, ".jsonl") + "-" + hex.EncodeToString(b) + ".jsonl"
}

// promotedPathTaken 报告路径或其任一侧车产物已存在。
func promotedPathTaken(p string) bool {
	for _, cand := range []string{
		p,
		session.LogPathFor(p),
		session.CheckpointPathFor(p),
		session.StatePath(p),
		p + ".meta",
	} {
		if _, err := os.Stat(cand); err == nil {
			return true
		}
	}
	return false
}

// removePromotedSessionFiles 清理一次失败提升落下的产物（best-effort，仅删
// 本次新建路径族，绝不触碰源 transcript）。
func removePromotedSessionFiles(sessionPath string) {
	for _, cand := range []string{
		sessionPath,
		session.LogPathFor(sessionPath),
		session.CheckpointPathFor(sessionPath),
		session.StatePath(sessionPath),
	} {
		if err := os.Remove(cand); err != nil && !os.IsNotExist(err) {
			slog.Warn("子代理提升：清理失败产物失败", "path", cand, "error", err)
		}
	}
}

// promoteMessagesEqual 逐条比较两个消息序列在投影语义下的等价性
// （role/正文/reasoning/签名/工具配对/工具调用三元组全等且保序）。
func promoteMessagesEqual(a, b []provider.Message) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.Role != y.Role || x.Content != y.Content ||
			x.ReasoningContent != y.ReasoningContent ||
			x.ReasoningSignature != y.ReasoningSignature ||
			x.ToolCallID != y.ToolCallID || x.Name != y.Name ||
			len(x.ToolCalls) != len(y.ToolCalls) {
			return false
		}
		for j := range x.ToolCalls {
			if x.ToolCalls[j] != y.ToolCalls[j] {
				return false
			}
		}
	}
	return true
}
