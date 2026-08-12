package app

import (
	"github.com/gaea/gaea/internal/office"
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
