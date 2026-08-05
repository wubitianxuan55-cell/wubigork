package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/office"
	"github.com/gaea/gaea/internal/office/proposal"
)

var (
	jm = office.NewJobManager(nil)
	sm = office.NewSessionModeStore()
)

func (a *officeState) OfficeExecute(act, path, tgt, q, url, content string) office.ExecResult {
	return office.Execute(office.DesktopAgentAction(act), path, tgt, q, url, content)
}
func (a *officeState) OfficeIsTask(text string) bool { return office.IsTask(text) }
func (a *officeState) OfficeListFolder(p string) office.ExecResult {
	if p == "" {
		p = "C:\\"
	}
	return a.OfficeExecute("list_folder", p, "", "", "", "")
}
func (a *officeState) OfficeReadFile(p string) office.ExecResult {
	return a.OfficeExecute("read_text", p, "", "", "", "")
}
func (a *officeState) OfficeGetJobState(s string) *office.AgentJobState { return jm.GetState(s) }
func (a *officeState) OfficeCancelJob(s string)                         { jm.Cancel(s) }
func (a *officeState) OfficeGetMode(s string) bool                      { return sm.GetMode(s) }
func (a *officeState) OfficeSetMode(s string, e bool)                   { sm.SetMode(s, e) }

func (a *officeState) ProposalTemplates() []map[string]interface{} {
	if a.proposalSvc == nil {
		return nil
	}
	var r []map[string]interface{}
	for _, t := range a.proposalSvc.ListTemplates() {
		r = append(r, map[string]interface{}{"id": t.ID, "name": t.Name, "description": t.Description, "sections": t.Sections})
	}
	return r
}
func (a *officeState) ProposalCreate(title, tmpl, req, cat string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.Create(title, tmpl, req, cat)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalList() ([]map[string]interface{}, error) {
	l, err := a.proposalSvc.List()
	if err != nil {
		return nil, err
	}
	var r []map[string]interface{}
	for i := range l {
		r = append(r, toMap(&l[i]))
	}
	return r, nil
}
func (a *officeState) ProposalGet(id string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.Get(id)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalUpdate(m map[string]interface{}) error {
	return a.proposalSvc.Update(fromMap(m))
}
func (a *officeState) ProposalDelete(id string) error { return a.proposalSvc.Delete(id) }
func (a *officeState) ProposalGenerateOutline(pid, req string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.GenerateOutline(a.ctx, pid, req)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalGenerateSection(pid, sid, inst string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.GenerateSection(a.ctx, pid, sid, inst)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalPolish(pid, sid, c, op string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.Polish(a.ctx, pid, sid, c, op)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalExport(pid string) (string, error) {
	return a.proposalSvc.ExportMarkdown(pid)
}
func (a *officeState) ProposalGenerateChart(pid, sid, ct string) (map[string]interface{}, error) {
	p, mc, err := a.proposalSvc.GenerateChart(a.ctx, pid, sid, ct)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"proposal": toMap(p), "mermaidCode": mc}, nil
}
func (a *officeState) ProposalParseBidFile(pid string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.ParseBidFile(a.ctx, pid)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalSaveRawText(pid, fp string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.SaveRawText(pid, fp)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalConvertFiles(pid string) {
	if a.proposalSvc == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("office: convert files goroutine panic recovered", "panic", r)
			}
		}()
		a.proposalSvc.ConvertFiles(pid, func(cur, total int, name, status string) {
			a.emit("proposal-convert-progress", map[string]interface{}{"current": cur, "total": total, "filename": name, "status": status})
		})
	}()
}
func (a *officeState) ProposalRemoveRawFile(pid string, idx int) (map[string]interface{}, error) {
	p, err := a.proposalSvc.RemoveRawFile(pid, idx)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalCheckCoverage(pid string) (map[string]interface{}, error) {
	p, r, err := a.proposalSvc.CheckCoverage(a.ctx, pid)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"proposal": toMap(p), "coverage": r}, nil
}
func (a *officeState) ProposalCheckCompliance(pid string) (map[string]interface{}, error) {
	p, items, err := a.proposalSvc.CheckCompliance(a.ctx, pid)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"proposal": toMap(p), "items": items}, nil
}
func (a *officeState) ProposalReadFile(path string) (string, error) {
	return proposal.ReadTextFile(path)
}
func (a *officeState) ProposalExportDocx(pid string) (string, error) {
	return a.proposalSvc.ExportDocx(pid)
}
func (a *officeState) ProposalSaveUploadedFile(pid, name string, data []byte) (map[string]interface{}, error) {
	p, err := a.proposalSvc.SaveUploadedFile(pid, name, data)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalAddSection(pid, parentID, title string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.AddSection(pid, parentID, title)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalRemoveSection(pid, sid string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.RemoveSection(pid, sid)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}
func (a *officeState) ProposalRenameSection(pid, sid, title string) (map[string]interface{}, error) {
	p, err := a.proposalSvc.RenameSection(pid, sid, title)
	if err != nil {
		return nil, err
	}
	return toMap(p), nil
}

func (a *officeState) ProposalGenerateSectionStream(pid, sid, inst string) {
	if a.proposalSvc == nil || a.client == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("office: section stream goroutine panic recovered", "panic", r)
				a.emit("proposal-stream", map[string]interface{}{"type": "error", "error": "生成异常: " + fmt.Sprint(r)})
			}
		}()
		p, err := a.proposalSvc.Get(pid)
		if err != nil {
			a.emit("proposal-stream", map[string]interface{}{"type": "error", "error": "加载方案失败: " + err.Error()})
			return
		}
		var ts *proposal.ProposalSection
		for _, sec := range fSRP(p.Sections) {
			if sec.ID == sid {
				ts = sec
				break
			}
		}
		if ts == nil {
			a.emit("proposal-stream", map[string]interface{}{"type": "error", "error": "章节未找到"})
			return
		}
		// 构建上下文：需求 + 大纲 + 招标要点 + 前一章节
		var ctx []string
		ctx = append(ctx, "方案："+p.Title)
		ctx = append(ctx, "方案类型："+p.Template)
		if p.Requirements != "" {
			ctx = append(ctx, "需求描述："+p.Requirements)
		}
		if p.BidSummary != nil {
			if len(p.BidSummary.TechScoring) > 0 {
				ctx = append(ctx, "【招标评分标准】")
				for _, sc := range p.BidSummary.TechScoring {
					ctx = append(ctx, fmt.Sprintf("- %s(%s分):%s", sc.Name, sc.MaxScore, sc.Requirement))
				}
			}
			if len(p.BidSummary.KeyRequirements) > 0 {
				ctx = append(ctx, "【核心要求】")
				for _, r := range p.BidSummary.KeyRequirements {
					ctx = append(ctx, "- "+r)
				}
			}
			if len(p.BidSummary.RedLines) > 0 {
				ctx = append(ctx, "【废标条款（严禁违反）】")
				for _, r := range p.BidSummary.RedLines {
					ctx = append(ctx, "- "+r)
				}
			}
			if p.BidSummary.Overview != "" {
				ctx = append(ctx, "【项目概况】"+p.BidSummary.Overview)
			}
			if p.BidSummary.Duration != "" {
				ctx = append(ctx, "【工期】"+p.BidSummary.Duration)
			}
		}
		ctx = append(ctx, "方案大纲：")
		for _, sec := range fSRP(p.Sections) {
			ctx = append(ctx, fmt.Sprintf("%s%d. %s", strings.Repeat("  ", max(0, sec.Level-1)), sec.Index+1, sec.Title))
		}
		var prevContent string
		for _, sec := range fSRP(p.Sections) {
			if sec.ID == sid {
				break
			}
			if sec.Content != "" {
				prevContent = sec.Content
			}
		}
		if prevContent != "" {
			ctx = append(ctx, "前一章节内容参考："+prevContent)
		}
		sp := eSP(p.Template, fmt.Sprintf("撰写「%s」章节。专业、Markdown，紧扣标题。字数500-1500字（核心章节更详细）。直接输出章节正文，不需要标题。", ts.Title))
		var body string
		for attempt := 0; attempt <= 3; attempt++ {
			cp := strings.Join(ctx, "\n")
			if body != "" {
				cp = fmt.Sprintf("%s\n\n【已生成内容，请自然续写，不要重复前面内容】\n%s", cp, body)
			}
			// 办公功能级模型绑定（func_office_engine/model）；未绑定回退全局模型。
			// 直接用 a.cfg.Model 会把历史残留模型名发到活跃引擎（如 xAI）导致 404。
			eng, model := a.featureModel("office")
			if model == "" {
				model = a.cfg.Model
			}
			chunks, err := a.client.ChatStream(a.ctx, &ai.ChatRequest{Model: model, EngineID: eng, Messages: []ai.ChatMessage{{Role: "system", Content: sp}, {Role: "user", Content: cp}}})
			if err != nil {
				if body == "" {
					a.emit("proposal-stream", map[string]interface{}{"type": "error", "error": "AI 生成失败: " + err.Error()})
				}
				break
			}
			a.emit("xai-output", map[string]interface{}{"type": "request", "model": a.cfg.Model, "system": sp, "user": cp})
			for c := range chunks {
				if c.Error != "" || c.Done {
					break
				}
				body += c.Content
				a.emit("proposal-stream", map[string]interface{}{"type": "chunk", "content": c.Content, "total": len([]rune(body))})
			}
			if len([]rune(body)) >= 100 {
				break
			}
		}
		ts.Content = strings.TrimSpace(body)
		ts.Status = "completed"
		p.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
		if err := a.proposalSvc.Update(p); err != nil {
			a.emit("proposal-stream", map[string]interface{}{"type": "error", "error": "保存失败: " + err.Error()})
			return
		}
		a.emit("proposal-stream", map[string]interface{}{"type": "done", "content": ts.Content, "sectionId": sid, "total": len([]rune(ts.Content))})
		a.emit("xai-output", map[string]interface{}{"type": "response", "content": ts.Content, "length": len([]rune(ts.Content))})
	}()
}
func fSRP(ss []proposal.ProposalSection) []*proposal.ProposalSection {
	var r []*proposal.ProposalSection
	for i := range ss {
		r = append(r, &ss[i])
		r = append(r, fSRP(ss[i].Children)...)
	}
	return r
}
func eSP(t, b string) string {
	if t == "soil-remediation-bid" {
		return b + proposal.SoilRemediationKB
	}
	return b
}

func toMap(p *proposal.Proposal) map[string]interface{} {
	ss := make([]map[string]interface{}, len(p.Sections))
	for i, s := range p.Sections {
		ss[i] = stm(s)
	}
	r := map[string]interface{}{"id": p.ID, "title": p.Title, "category": p.Category, "template": p.Template, "requirements": p.Requirements, "status": p.Status, "sections": ss, "createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt}
	if p.BidSummary != nil {
		r["bidSummary"] = btm(p.BidSummary)
	}
	return r
}
func stm(s proposal.ProposalSection) map[string]interface{} {
	r := map[string]interface{}{"id": s.ID, "proposalId": s.ProposalID, "parentId": s.ParentID, "index": s.Index, "level": s.Level, "title": s.Title, "content": s.Content, "status": s.Status}
	if len(s.Children) > 0 {
		ch := make([]map[string]interface{}, len(s.Children))
		for i, c := range s.Children {
			ch[i] = stm(c)
		}
		r["children"] = ch
	}
	return r
}
func btm(bs *proposal.BidSummary) map[string]interface{} {
	sc := make([]map[string]interface{}, len(bs.TechScoring))
	for i, x := range bs.TechScoring {
		sc[i] = map[string]interface{}{"name": x.Name, "maxScore": x.MaxScore, "requirement": x.Requirement}
	}
	return map[string]interface{}{"techScoring": sc, "keyRequirements": bs.KeyRequirements, "duration": bs.Duration, "redLines": bs.RedLines, "overview": bs.Overview, "extra": bs.Extra, "rawMarkdown": bs.RawMarkdown, "rawFiles": ftm(bs.RawFiles)}
}
func fromMap(m map[string]interface{}) *proposal.Proposal {
	p := &proposal.Proposal{}
	if v, ok := m["id"].(string); ok {
		p.ID = v
	}
	if v, ok := m["title"].(string); ok {
		p.Title = v
	}
	if v, ok := m["category"].(string); ok {
		p.Category = v
	}
	if v, ok := m["template"].(string); ok {
		p.Template = v
	}
	if v, ok := m["requirements"].(string); ok {
		p.Requirements = v
	}
	if v, ok := m["status"].(string); ok {
		p.Status = v
	}
	if v, ok := m["createdAt"].(string); ok {
		p.CreatedAt = v
	}
	if v, ok := m["updatedAt"].(string); ok {
		p.UpdatedAt = v
	}
	if bs, ok := m["bidSummary"].(map[string]interface{}); ok {
		p.BidSummary = bsf(bs)
	}
	if ss, ok := m["sections"].([]interface{}); ok {
		for _, si := range ss {
			if sm, ok := si.(map[string]interface{}); ok {
				p.Sections = append(p.Sections, sfm(sm))
			}
		}
	}
	return p
}
func sfm(m map[string]interface{}) proposal.ProposalSection {
	s := proposal.ProposalSection{}
	if v, ok := m["id"].(string); ok {
		s.ID = v
	}
	if v, ok := m["proposalId"].(string); ok {
		s.ProposalID = v
	}
	if v, ok := m["parentId"].(string); ok {
		s.ParentID = v
	}
	if v, ok := m["index"].(float64); ok {
		s.Index = int(v)
	}
	if v, ok := m["level"].(float64); ok {
		s.Level = int(v)
	}
	if v, ok := m["title"].(string); ok {
		s.Title = v
	}
	if v, ok := m["content"].(string); ok {
		s.Content = v
	}
	if v, ok := m["status"].(string); ok {
		s.Status = v
	}
	if ch, ok := m["children"].([]interface{}); ok {
		for _, ci := range ch {
			if cm, ok := ci.(map[string]interface{}); ok {
				s.Children = append(s.Children, sfm(cm))
			}
		}
	}
	return s
}
func bsf(m map[string]interface{}) *proposal.BidSummary {
	bs := &proposal.BidSummary{Extra: make(map[string]string)}
	if v, ok := m["duration"].(string); ok {
		bs.Duration = v
	}
	if v, ok := m["overview"].(string); ok {
		bs.Overview = v
	}
	if v, ok := m["rawMarkdown"].(string); ok {
		bs.RawMarkdown = v
	}
	if arr, ok := m["keyRequirements"].([]interface{}); ok {
		for _, x := range arr {
			if s, ok := x.(string); ok {
				bs.KeyRequirements = append(bs.KeyRequirements, s)
			}
		}
	}
	if arr, ok := m["redLines"].([]interface{}); ok {
		for _, x := range arr {
			if s, ok := x.(string); ok {
				bs.RedLines = append(bs.RedLines, s)
			}
		}
	}
	if arr, ok := m["techScoring"].([]interface{}); ok {
		for _, x := range arr {
			if sm, ok := x.(map[string]interface{}); ok {
				si := proposal.ScoringItem{}
				if v, ok := sm["name"].(string); ok {
					si.Name = v
				}
				if v, ok := sm["maxScore"].(string); ok {
					si.MaxScore = v
				}
				if v, ok := sm["requirement"].(string); ok {
					si.Requirement = v
				}
				bs.TechScoring = append(bs.TechScoring, si)
			}
		}
	}
	if ex, ok := m["extra"].(map[string]interface{}); ok {
		for k, v := range ex {
			if s, ok := v.(string); ok {
				bs.Extra[k] = s
			}
		}
	}
	if arr, ok := m["rawFiles"].([]interface{}); ok {
		for _, x := range arr {
			if fm, ok := x.(map[string]interface{}); ok {
				fd := proposal.FileDoc{}
				if v, ok := fm["name"].(string); ok {
					fd.Name = v
				}
				if v, ok := fm["path"].(string); ok {
					fd.Path = v
				}
				if v, ok := fm["markdown"].(string); ok {
					fd.Markdown = v
				}
				if v, ok := fm["size"].(float64); ok {
					fd.Size = int(v)
				}
				bs.RawFiles = append(bs.RawFiles, fd)
			}
		}
	}
	return bs
}
func ftm(files []proposal.FileDoc) []map[string]interface{} {
	r := make([]map[string]interface{}, len(files))
	for i, f := range files {
		r[i] = map[string]interface{}{"name": f.Name, "path": f.Path, "markdown": f.Markdown, "size": f.Size}
	}
	return r
}
