package app

import (
	"fmt"
	"log/slog"

	"github.com/wubigork/wubigork/internal/office"
)

var (
	officeJobMgr    = office.NewJobManager(nil)
	officeModeStore = office.NewSessionModeStore()
)

func (a *App) OfficeExecute(action string, path, target, query, url, content string) office.ExecResult {
	slog.Info("[office] execute", "action", action, "path", path)
	result := office.Execute(office.DesktopAgentAction(action), path, target, query, url, content)
	slog.Info("[office] result", "success", result.Success, "summary", result.Summary)
	return result
}

func (a *App) OfficeIsTask(text string) bool { return office.IsTask(text) }

func (a *App) OfficeListFolder(path string) office.ExecResult {
	if path == "" { path = "C:\\" }
	return a.OfficeExecute("list_folder", path, "", "", "", "")
}

func (a *App) OfficeReadFile(path string) office.ExecResult {
	return a.OfficeExecute("read_text", path, "", "", "", "")
}

func (a *App) OfficeGetJobState(sessionID string) *office.AgentJobState { return officeJobMgr.GetState(sessionID) }
func (a *App) OfficeCancelJob(sessionID string)                         { officeJobMgr.Cancel(sessionID) }
func (a *App) OfficeGetMode(sessionID string) bool                      { return officeModeStore.GetMode(sessionID) }
func (a *App) OfficeSetMode(sessionID string, enabled bool)             { officeModeStore.SetMode(sessionID, enabled) }

func (a *App) OfficeRunTask(taskDesc string) (map[string]interface{}, error) {
	if a.client == nil { return nil, fmt.Errorf("model client not initialized") }
	slog.Info("[office] run task", "desc", taskDesc)
	reply, err := a.client.ChatSimpleStream(a.ctx, "", `You are a desktop assistant. Provide a concise plan as JSON: {"title":"...","steps":[{"index":1,"action":"list_folder|read_text|search_file|stat_file|open_file|web_search","description":"...","path":"..."}]}`, taskDesc)
	if err != nil { return nil, fmt.Errorf("LLM call failed: %w", err) }
	return map[string]interface{}{"plan": reply, "reply": "Plan:\n\n" + reply}, nil
}
