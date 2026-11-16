package app

// wenxue_bridge_extra.go — 图文桥面语义补件（2026-11-17 审查批6）。
//
// 能力面板的 MCP 管理（增/删/重连/启停）与「从会话沉淀技能」此前经 mappings
// 落 OfficeB：MCP 四件操作 gaeaCtrl() 注册表（wenxue 面板管的是办公引擎的
// MCP）、SkillDraftFromSession 读 GaeaHistory()（蒸馏的是办公当前会话）——
// 均为用户可感知的跨板块误操作（OfficeB 消费面调查实证，批5 报告 §②）。
// 本文件五件换锚图文引擎/图文会话；方法体与办公版同构
// （gaea_ui_extra.go:1180-1231 / office_skill_record.go:152-198），差异逐条注释。
//
// SetAgentParams 不在此列：cfg.Agent.* 是双引擎共享配置且 gaeaApplyCfg 重建
// 跟随（wx 引擎同批生效），属共享面正确语义；Accept*Suggestion 不在此列：
// wx 引擎不挂建议机制（TurnDone 办公域钩子有意不挂），图文建议队列恒空，
// accept 路径实际不可达（低危留档）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/skill"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/util"
)

// WenxueAddMCPServer 图文引擎添加 MCP 服务器（wx ctrl 注册表；同 GaeaAddMCPServer）。
func (a *App) WenxueAddMCPServer(input MCPServerInput) (int, error) {
	c := wenxueCtrl()
	if c == nil {
		return 0, fmt.Errorf("图文引擎未初始化")
	}
	entry := gaeaConfig.PluginEntry{Name: input.Name, Command: input.Command, Args: input.Args, Env: input.Env}
	if input.URL != "" {
		entry.URL = input.URL
	}
	return c.AddMCPServer(entry)
}

// WenxueRemoveMCPServer 移除图文引擎 MCP 服务器。
func (a *App) WenxueRemoveMCPServer(name string) error {
	if c := wenxueCtrl(); c != nil {
		_, err := c.RemoveMCPServer(name)
		return err
	}
	return nil
}

// WenxueRetryMCPServer 重连图文引擎已配置的 MCP 服务器。
func (a *App) WenxueRetryMCPServer(name string) error {
	if c := wenxueCtrl(); c != nil {
		_, err := c.ConnectConfiguredMCPServer(name)
		return err
	}
	return nil
}

// WenxueSetMCPServerEnabled 启用/停用图文引擎 MCP 服务器。
func (a *App) WenxueSetMCPServerEnabled(name string, enabled bool) error {
	c := wenxueCtrl()
	if c == nil {
		return nil
	}
	if enabled {
		_, err := c.ConnectConfiguredMCPServer(name)
		return err
	}
	c.DisconnectMCPServer(name)
	return nil
}

// WenxueSkillDraftFromSession 把图文当前会话回放蒸馏为技能草稿（读
// WenxueHistory；路由 routeOfficeLocal("wenxue")=功能绑定→本地优先→常规，
// 账目标签 Feature:"wenxue"——办公版读 GaeaHistory/标 office）。草稿只回
// 不落盘——落盘必经 GaeaSkillDraftSave（共享面，用户审阅后）。
func (a *App) WenxueSkillDraftFromSession() (SkillRecordResult, error) {
	if wenxueCtrl() == nil {
		return SkillRecordResult{}, fmt.Errorf("图文引擎未初始化")
	}
	msgs := a.WenxueHistory()
	replay := buildSkillReplay(msgs)
	if replay == "" {
		return SkillRecordResult{}, fmt.Errorf("当前会话为空——先完成一次任务再录制技能")
	}
	if a.eng == nil {
		return SkillRecordResult{}, fmt.Errorf("提示词引擎未初始化")
	}
	tmpl := a.eng.Get("skill-from-session")
	if tmpl == nil {
		return SkillRecordResult{}, fmt.Errorf("缺少 skill-from-session 模板文件")
	}
	if a.clientRef() == nil {
		return SkillRecordResult{}, fmt.Errorf("ai client unavailable")
	}
	engID, model, _ := a.routeOfficeLocal("wenxue")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.clientRef().ChatSimpleStreamWithOptions(ctx, model, sys, usr,
			ai.ChatSimpleOptions{EngineID: engID, Feature: "wenxue", Temperature: 0.2, MaxTokens: 1500, TimeoutMinutes: 3})
	}
	system := tmpl.BuildSystemPrompt("")
	user := tmpl.BuildUserPrompt(map[string]string{"replay": replay})
	jsonStr, err := util.RetryJSON(ctx, caller, system, user, 2)
	if err != nil {
		return SkillRecordResult{}, fmt.Errorf("蒸馏技能草稿失败: %w", err)
	}
	var draft SkillDraft
	if err := json.Unmarshal([]byte(jsonStr), &draft); err != nil {
		return SkillRecordResult{}, fmt.Errorf("解析技能草稿失败: %w", err)
	}
	if err := parseSkillDraft(&draft); err != nil {
		return SkillRecordResult{}, err
	}
	return SkillRecordResult{
		Draft:   draft,
		Replay:  replay,
		Preview: skill.RenderSkillFile(draft.Name, draft.Description, renderSkillDraftBody(draft)),
	}, nil
}

// ── 证据链读侧三件（2026-11-17 审查批7 刀A）────────────────────────
// 写侧本就按书目录落盘（boot.go JournalDir=opts.Cwd/.gaea/work/journal），
// 读侧此前只有办公版（gaeaCwd 锚办公工作区）——图文 UI 的版本时间线/回滚
// 读错账本（恒空/错位）。三件换锚 wenxueWorkspace()，核心逻辑经
// verifyRecordAt/rollbackRecordAt 参数化复用（办公版委托同一实现，逐字节
// 行为保持）。

// wenxueJournalStore 图文账本（书工作空间 .gaea/work/journal）。
func wenxueJournalStore() *evidence.JournalStore {
	st, err := evidence.OpenJournal(filepath.Join(wenxueWorkspace(), ".gaea", "work", "journal"))
	if err != nil {
		return nil
	}
	return st
}

// WenxueJournalList 图文证据卡最近列表（同 GaeaJournalList，换根）。
func (a *App) WenxueJournalList(limit int) []evidence.ChangeRecord {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	dir := filepath.Join(wenxueWorkspace(), ".gaea", "work", "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []evidence.ChangeRecord{}
	}
	st, err := evidence.OpenJournal(dir)
	if err != nil {
		return []evidence.ChangeRecord{}
	}
	var all []evidence.ChangeRecord
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		recs, err := st.List(e.Name()[:len(e.Name())-len(".jsonl")])
		if err != nil {
			continue
		}
		all = append(all, recs...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].At > all[j].At })
	if len(all) > limit {
		all = all[:limit]
	}
	return all
}

// WenxueVerifyRecord 图文证据卡复核（同 GaeaVerifyRecord，账本与根换锚书目录；
// xlsx 重算/视觉产物目录随 cwd 锚书工作区）。
func (a *App) WenxueVerifyRecord(id string) (evidence.Verdict, error) {
	st := wenxueJournalStore()
	if st == nil {
		return evidence.Verdict{}, fmt.Errorf("Journal 不可用")
	}
	rec, ok := st.FindByID(id)
	if !ok {
		return evidence.Verdict{}, fmt.Errorf("证据卡 %s 不存在", id)
	}
	return verifyRecordAt(st, rec, id, wenxueWorkspace())
}

// WenxueRollbackRecord 图文证据卡回滚（同 GaeaRollbackRecord，账本与根换锚
// 书目录；恢复前快照落书侧基线同目录）。
func (a *App) WenxueRollbackRecord(id string) error {
	st := wenxueJournalStore()
	if st == nil {
		return fmt.Errorf("Journal 不可用")
	}
	rec, ok := st.FindByID(id)
	if !ok {
		return fmt.Errorf("证据卡 %s 不存在", id)
	}
	return rollbackRecordAt(st, rec, wenxueWorkspace())
}
