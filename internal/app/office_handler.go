package app

import (
	officecore "github.com/gaea/gaea/internal/core"
)

var (
	jm = officecore.NewJobManager(nil)
	sm = officecore.NewSessionModeStore()
)

func (a *officeState) OfficeExecute(act, path, tgt, q, url, content string) officecore.ExecResult {
	return officecore.Execute(officecore.DesktopAgentAction(act), path, tgt, q, url, content)
}
func (a *officeState) OfficeIsTask(text string) bool { return officecore.IsTask(text) }
func (a *officeState) OfficeListFolder(p string) officecore.ExecResult {
	if p == "" {
		p = "C:\\"
	}
	return a.OfficeExecute("list_folder", p, "", "", "", "")
}
func (a *officeState) OfficeReadFile(p string) officecore.ExecResult {
	return a.OfficeExecute("read_text", p, "", "", "", "")
}
func (a *officeState) OfficeGetJobState(s string) *officecore.AgentJobState { return jm.GetState(s) }
func (a *officeState) OfficeCancelJob(s string)                            { jm.Cancel(s) }
func (a *officeState) OfficeGetMode(s string) bool                         { return sm.GetMode(s) }
func (a *officeState) OfficeSetMode(s string, e bool)                      { sm.SetMode(s, e) }